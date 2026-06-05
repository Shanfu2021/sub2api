package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

var (
	ErrAgentManagementForbidden             = infraerrors.Forbidden("AGENT_MANAGEMENT_FORBIDDEN", "agent management operation is not allowed")
	ErrAgentManagementNotDirectChild        = infraerrors.Forbidden("AGENT_MANAGEMENT_NOT_DIRECT_CHILD", "target user is not a direct child")
	ErrAgentManagementAllocationExceeded    = infraerrors.BadRequest("AGENT_MANAGEMENT_ALLOCATION_EXCEEDED", "allocation exceeds remaining capacity")
	ErrAgentManagementUnsupportedRole       = infraerrors.BadRequest("AGENT_MANAGEMENT_UNSUPPORTED_ROLE", "unsupported agent management role")
	ErrAgentManagementInvalidTarget         = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_TARGET", "invalid target user")
	ErrAgentManagementInvalidAllocation     = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_ALLOCATION", "concurrency must be positive and RPM must be non-negative")
	ErrAgentManagementInvalidGroupRate      = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_GROUP_RATE", "group delegation rate multiplier must be positive")
	ErrAgentManagementInvalidIncome         = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_INCOME", "agent income must be a valid number")
	ErrAgentManagementGroupRateBelowCost    = infraerrors.BadRequest("AGENT_MANAGEMENT_GROUP_RATE_BELOW_COST", "group delegation rate multiplier cannot be lower than manager cost rate")
	ErrAgentManagementInvalidGroup          = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_GROUP", "only active exclusive groups can be delegated")
	ErrAgentManagementInvalidGroupSelection = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_GROUP_SELECTION", "at least one group must be selected")
	ErrAgentManagementRootAdminNotPresent   = infraerrors.NotFound("AGENT_MANAGEMENT_ROOT_ADMIN_NOT_PRESENT", "root admin not found")
	ErrAgentManagementPoolReclaimExceeded   = infraerrors.BadRequest("AGENT_MANAGEMENT_POOL_RECLAIM_EXCEEDED", "agent pool cannot be lower than child allocations")
	ErrAgentManagementNotImplemented        = infraerrors.New(http.StatusNotImplemented, "AGENT_MANAGEMENT_NOT_IMPLEMENTED", "agent management feature is not implemented yet")
)

const (
	DefaultAgentInviteConcurrency = 1
	DefaultAgentInviteRPM         = 1
)

type AgentPoolReclaimExceededError struct {
	AllocatedConcurrency int
	RequestedConcurrency int
	AllocatedRPM         int
	RequestedRPM         int
}

func (e *AgentPoolReclaimExceededError) Error() string {
	return ErrAgentManagementPoolReclaimExceeded.Error()
}

func (e *AgentPoolReclaimExceededError) Unwrap() error {
	return ErrAgentManagementPoolReclaimExceeded
}

type AllocationUpdate struct {
	AllocatedConcurrency int  `json:"allocated_concurrency"`
	AllocatedRPM         int  `json:"allocated_rpm"`
	Concurrency          *int `json:"concurrency,omitempty"`
	RPM                  *int `json:"rpm,omitempty"`
}

type AgentProfile struct {
	UserID                   int64 `json:"user_id"`
	PoolConcurrency          int   `json:"pool_concurrency"`
	PoolRPM                  int   `json:"pool_rpm"`
	InviteDefaultConcurrency int   `json:"invite_default_concurrency"`
	InviteDefaultRPM         int   `json:"invite_default_rpm"`
}

type AgentUpgradeInput struct {
	TargetRole      string `json:"target_role"`
	PoolConcurrency int    `json:"pool_concurrency"`
	PoolRPM         int    `json:"pool_rpm"`
}

type CreateDirectUserInput struct {
	Email                string `json:"email"`
	Password             string `json:"password"`
	Username             string `json:"username"`
	AllocatedConcurrency int    `json:"allocated_concurrency"`
	AllocatedRPM         int    `json:"allocated_rpm"`
}

type AgentInviteDefaultsUpdate struct {
	InviteDefaultConcurrency int `json:"invite_default_concurrency"`
	InviteDefaultRPM         int `json:"invite_default_rpm"`
}

type AgentIncomeSetInput struct {
	AgentIncome float64 `json:"agent_income"`
	Reason      string  `json:"reason"`
}

type AgentChildNotesUpdate struct {
	Notes string `json:"notes"`
}

type AllocationSummary struct {
	TotalConcurrency     int  `json:"total_concurrency"`
	AllocatedConcurrency int  `json:"allocated_concurrency"`
	RemainingConcurrency int  `json:"remaining_concurrency"`
	TotalRPM             int  `json:"total_rpm"`
	AllocatedRPM         int  `json:"allocated_rpm"`
	RemainingRPM         int  `json:"remaining_rpm"`
	UnlimitedCapacity    bool `json:"unlimited_capacity"`
	UnlimitedConcurrency bool `json:"unlimited_concurrency"`
	UnlimitedRPM         bool `json:"unlimited_rpm"`
}

type QuotaUsageSummary struct {
	Concurrency          int
	RPM                  int
	UnlimitedConcurrency bool
	UnlimitedRPM         bool
}

type DirectChildrenResult struct {
	Users      []User                       `json:"users"`
	Pagination *pagination.PaginationResult `json:"pagination"`
}

type AdminAgentTreeResult struct {
	Items []AdminAgentTreeAgent `json:"items"`
}

type AdminAgentTreeAgent struct {
	Agent       User                       `json:"agent"`
	Users       []User                     `json:"users"`
	Enterprises []AdminAgentTreeEnterprise `json:"enterprises"`
}

type AdminAgentTreeEnterprise struct {
	Enterprise User   `json:"enterprise"`
	Employees  []User `json:"employees"`
}

type SubordinateStructureResult struct {
	OwnerOptions  []User                     `json:"owner_options"`
	SelectedOwner User                       `json:"selected_owner"`
	Users         []User                     `json:"users"`
	Enterprises   []AdminAgentTreeEnterprise `json:"enterprises"`
}

type DirectChildrenQuery struct {
	Pagination pagination.PaginationParams
	Search     string
}

type DirectChildrenGroupQuery struct {
	Pagination pagination.PaginationParams
	Search     string
	GroupID    int64
	GroupIDs   []int64
}

type AgentManagementSummary struct {
	Allocation     AllocationSummary          `json:"allocation"`
	InviteDefaults *AgentInviteDefaultsUpdate `json:"invite_defaults,omitempty"`
}

type AgentGroupRate struct {
	Group         Group   `json:"group"`
	EffectiveRate float64 `json:"effective_rate"`
	CanDelegate   bool    `json:"can_delegate"`
	Source        string  `json:"source"`
}

type ChildGroupDelegationOption struct {
	Group               Group   `json:"group"`
	EffectiveRate       float64 `json:"effective_rate"`
	CanDelegate         bool    `json:"can_delegate"`
	Source              string  `json:"source"`
	Assigned            bool    `json:"assigned"`
	ChildRateMultiplier float64 `json:"child_rate_multiplier"`
	ChildCanDelegate    bool    `json:"child_can_delegate"`
}

type AgentGroupDelegation struct {
	ID             int64
	ManagerUserID  int64
	ChildUserID    int64
	GroupID        int64
	RateMultiplier float64
	CanDelegate    bool
	Group          *Group
}

type ChildGroupDelegationInput struct {
	RateMultiplier float64 `json:"rate_multiplier"`
	CanDelegate    bool    `json:"can_delegate"`
}

type DirectChildKind string

const (
	DirectChildKindUsers       DirectChildKind = "users"
	DirectChildKindAgents      DirectChildKind = "agents"
	DirectChildKindEnterprises DirectChildKind = "enterprises"
)

type ChildGroupDelegationBatchInput struct {
	GroupIDs       []int64 `json:"group_ids"`
	All            bool    `json:"all"`
	RateMultiplier float64 `json:"rate_multiplier"`
	CanDelegate    bool    `json:"can_delegate"`
}

type DirectChildrenGroupDelegationBatchInput struct {
	GroupIDs    []int64 `json:"group_ids"`
	All         bool    `json:"all"`
	ChildIDs    []int64 `json:"child_ids"`
	AllChildren *bool   `json:"all_children"`
	CanDelegate bool    `json:"can_delegate"`
}

type DirectChildrenGroupDelegationUpdateInput struct {
	GroupID        int64    `json:"group_id"`
	ChildIDs       []int64  `json:"child_ids"`
	All            bool     `json:"all"`
	RateMultiplier *float64 `json:"rate_multiplier,omitempty"`
	CanDelegate    *bool    `json:"can_delegate,omitempty"`
}

type DirectChildrenGroupDelegationUpdateResult struct {
	Kind              DirectChildKind `json:"kind"`
	GroupID           int64           `json:"group_id"`
	RequestedChildIDs []int64         `json:"requested_child_ids"`
	All               bool            `json:"all"`
	UpdatedChildren   int             `json:"updated_children"`
	SkippedChildren   int             `json:"skipped_children"`
}

type DirectChildrenGroupDelegationReclaimInput struct {
	GroupID  int64   `json:"group_id"`
	ChildIDs []int64 `json:"child_ids"`
	All      bool    `json:"all"`
}

type DirectChildrenGroupDelegationReclaimResult struct {
	Kind              DirectChildKind `json:"kind"`
	GroupID           int64           `json:"group_id"`
	RequestedChildIDs []int64         `json:"requested_child_ids"`
	All               bool            `json:"all"`
	RemovedChildren   int             `json:"removed_children"`
	SkippedChildren   int             `json:"skipped_children"`
}

type AgentInviteGroupDefault struct {
	ID             int64
	AgentUserID    int64
	GroupID        int64
	RateMultiplier float64
	Group          *Group
}

type AgentInviteGroupDefaultInput struct {
	RateMultiplier float64 `json:"rate_multiplier"`
}

type AgentInviteGroupDefaultBatchInput struct {
	GroupIDs       []int64 `json:"group_ids"`
	All            bool    `json:"all"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type AgentManagementRepository interface {
	GetRootAdmin(ctx context.Context) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	ListDirectChildrenWithSearch(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]User, *pagination.PaginationResult, error)
	SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	GetAgentProfile(ctx context.Context, userID int64) (*AgentProfile, error)
	UpsertAgentProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
	GetEnterpriseProfile(ctx context.Context, userID int64) (*EnterpriseProfile, error)
	UpsertEnterpriseProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
	GetEnterpriseEmployeeQuotaUsage(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64) (QuotaUsageSummary, error)
	UpdateAgentInviteDefaults(ctx context.Context, userID int64, inviteConcurrency int, inviteRPM int) error
	GetDirectChildQuotaUsage(ctx context.Context, parentID int64, excludeChildID *int64) (QuotaUsageSummary, error)
	SetEffectiveQuota(ctx context.Context, userID int64, concurrency int, rpm int) error
	SetParent(ctx context.Context, userID int64, parentID *int64) error
	SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error
	SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error
	DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error
	RehomeChildGroupDelegations(ctx context.Context, oldManagerID int64, newManagerID int64, childID int64) error
	ListGroupDelegationsForChild(ctx context.Context, childID int64) ([]AgentGroupDelegation, error)
	GetGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) (*AgentGroupDelegation, error)
	UpsertGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error
	RaiseManagedGroupRateFloor(ctx context.Context, agentID int64, groupID int64, minimumRate float64) error
	DeleteGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) error
	ListInviteGroupDefaults(ctx context.Context, agentID int64) ([]AgentInviteGroupDefault, error)
	GetAgentIncomeTotals(ctx context.Context, agentIDs []int64) (map[int64]float64, error)
	AddAgentIncomeAdjustment(ctx context.Context, agentID int64, adminID int64, amount float64, reason string) error
	GetChildNotes(ctx context.Context, managerID int64, childIDs []int64) (map[int64]string, error)
	UpsertChildNotes(ctx context.Context, managerID int64, childID int64, notes string) error
	UpsertInviteGroupDefault(ctx context.Context, agentID int64, groupID int64, rateMultiplier float64) error
	DeleteInviteGroupDefault(ctx context.Context, agentID int64, groupID int64) error
	DeleteAgentForAdminUserDeletion(ctx context.Context, user *User) ([]int64, error)
	RemoveUserGroupAccessForAdminUpdate(ctx context.Context, userID int64, groupIDs []int64) ([]int64, error)
	RaiseManagedGroupRateFloorForAdminUpdate(ctx context.Context, userID int64, groupID int64, minimumRate float64) ([]int64, error)
	RecalculateAgentQuota(ctx context.Context, agentID int64) error
}

type AgentUserDeletionCleanupRepository interface {
	DeleteAgentForAdminUserDeletion(ctx context.Context, user *User) ([]int64, error)
	RemoveUserGroupAccessForAdminUpdate(ctx context.Context, userID int64, groupIDs []int64) ([]int64, error)
	RaiseManagedGroupRateFloorForAdminUpdate(ctx context.Context, userID int64, groupID int64, minimumRate float64) ([]int64, error)
	RecalculateAgentQuota(ctx context.Context, agentID int64) error
}

type AgentEnterpriseDeletionCleanupRepository interface {
	HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error)
}

type AgentManagementService struct {
	repo                  AgentManagementRepository
	userRepo              UserRepository
	usageService          agentUsageQueryService
	settingRepo           SettingRepository
	groupRepo             GroupRepository
	userGroupRateRepo     UserGroupRateRepository
	authCacheInvalidator  APIKeyAuthCacheInvalidator
	enterpriseCleanupRepo AgentEnterpriseDeletionCleanupRepository
}

type agentUsageQueryService interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error)
	GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error)
}

func NewAgentManagementService(repo AgentManagementRepository, userRepo UserRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *AgentManagementService {
	return &AgentManagementService{
		repo:                 repo,
		userRepo:             userRepo,
		groupRepo:            groupRepo,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *AgentManagementService) SetSettingRepository(repo SettingRepository) {
	s.settingRepo = repo
}

func (s *AgentManagementService) SetEnterpriseCleanupRepository(repo AgentEnterpriseDeletionCleanupRepository) {
	s.enterpriseCleanupRepo = repo
}

func (s *AgentManagementService) SetUserGroupRateRepository(repo UserGroupRateRepository) {
	s.userGroupRateRepo = repo
}

func (s *AgentManagementService) SetUsageService(usageService agentUsageQueryService) {
	s.usageService = usageService
}

func (s *AgentManagementService) ListDirectUsers(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	return s.listDirectChildren(ctx, actorID, []string{RoleUser}, pagination.DefaultPagination())
}

func (s *AgentManagementService) ListDirectUsersWithQuery(ctx context.Context, actorID int64, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	return s.listDirectChildrenWithQuery(ctx, actorID, []string{RoleUser}, query)
}

func (s *AgentManagementService) ListDirectAgents(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.listDirectChildrenForActor(ctx, actor, []string{RoleAgentLevel1}, pagination.DefaultPagination())
}

func (s *AgentManagementService) ListDirectAgentsWithQuery(ctx context.Context, actorID int64, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.listDirectChildrenForActorWithQuery(ctx, actor, []string{RoleAgentLevel1}, query)
}

func (s *AgentManagementService) ListDirectEnterprises(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	return s.listDirectChildren(ctx, actorID, []string{RoleEnterprise}, pagination.DefaultPagination())
}

func (s *AgentManagementService) ListDirectEnterprisesWithQuery(ctx context.Context, actorID int64, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	return s.listDirectChildrenWithQuery(ctx, actorID, []string{RoleEnterprise}, query)
}

func (s *AgentManagementService) GetSummary(ctx context.Context, actorID int64) (*AgentManagementSummary, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	summary := &AgentManagementSummary{
		Allocation: buildAllocationSummary(actor, totalConcurrency, totalRPM, usage),
	}
	defaults, err := s.inviteDefaultsForManager(ctx, actor)
	if err != nil {
		return nil, err
	}
	summary.InviteDefaults = &defaults
	return summary, nil
}

func (s *AgentManagementService) GetAdminAgentTree(ctx context.Context, actorID int64) (*AdminAgentTreeResult, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin {
		return nil, ErrAgentManagementForbidden
	}
	rootAdmin, err := s.repo.GetRootAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
	}

	agents, err := s.listAllDirectChildrenForRoles(ctx, rootAdmin.ID, []string{RoleAgentLevel1})
	if err != nil {
		return nil, err
	}
	if err := s.populateProfileBackedUsers(ctx, agents); err != nil {
		return nil, err
	}
	if err := s.populateAgentIncomeTotals(ctx, agents); err != nil {
		return nil, err
	}

	items := make([]AdminAgentTreeAgent, 0, len(agents))
	for i := range agents {
		directUsers, err := s.listAllDirectChildrenForRoles(ctx, agents[i].ID, []string{RoleUser})
		if err != nil {
			return nil, err
		}
		enterprises, err := s.listAllDirectChildrenForRoles(ctx, agents[i].ID, []string{RoleEnterprise})
		if err != nil {
			return nil, err
		}
		if err := s.populateProfileBackedUsers(ctx, enterprises); err != nil {
			return nil, err
		}

		enterpriseNodes := make([]AdminAgentTreeEnterprise, 0, len(enterprises))
		for j := range enterprises {
			employees, err := s.listAllDirectChildrenForRoles(ctx, enterprises[j].ID, []string{RoleEmployee})
			if err != nil {
				return nil, err
			}
			enterpriseNodes = append(enterpriseNodes, AdminAgentTreeEnterprise{
				Enterprise: enterprises[j],
				Employees:  employees,
			})
		}

		items = append(items, AdminAgentTreeAgent{
			Agent:       agents[i],
			Users:       directUsers,
			Enterprises: enterpriseNodes,
		})
	}

	return &AdminAgentTreeResult{Items: items}, nil
}

func (s *AgentManagementService) GetSubordinateStructure(ctx context.Context, actorID int64, ownerID *int64) (*SubordinateStructureResult, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !isAgentManagerRole(actor.Role) {
		return nil, ErrAgentManagementForbidden
	}

	if actor.Role == RoleAdmin {
		rootAdmin, err := s.repo.GetRootAdmin(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
		}
		owners := []User{*rootAdmin}
		agents, err := s.listAllDirectChildrenForRoles(ctx, rootAdmin.ID, []string{RoleAgentLevel1})
		if err != nil {
			return nil, err
		}
		if err := s.populateProfileBackedUsers(ctx, agents); err != nil {
			return nil, err
		}
		if err := s.populateAgentIncomeTotals(ctx, agents); err != nil {
			return nil, err
		}
		owners = append(owners, agents...)

		selectedOwner := *rootAdmin
		if ownerID != nil && *ownerID > 0 {
			selectedOwner = User{}
			for i := range owners {
				if owners[i].ID == *ownerID {
					selectedOwner = owners[i]
					break
				}
			}
			if selectedOwner.ID == 0 {
				return nil, ErrAgentManagementForbidden
			}
		}
		return s.buildSubordinateStructureForOwner(ctx, owners, selectedOwner)
	}

	owners := []User{*actor}
	if err := s.populateProfileBackedUsers(ctx, owners); err != nil {
		return nil, err
	}
	if err := s.populateAgentIncomeTotals(ctx, owners); err != nil {
		return nil, err
	}
	return s.buildSubordinateStructureForOwner(ctx, owners, owners[0])
}

func (s *AgentManagementService) ListAgentUsage(ctx context.Context, actorID int64, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	if s.usageService == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if params.PageSize == 0 {
		params = pagination.DefaultPagination()
	}
	scopedFilters, err := s.scopeAgentUsageFilters(ctx, actorID, filters)
	if err != nil {
		return nil, nil, err
	}
	if scopedFilters.UserID == 0 && len(scopedFilters.UserIDs) == 0 {
		return []UsageLog{}, &pagination.PaginationResult{Total: 0, Page: params.Page, PageSize: params.Limit(), Pages: 0}, nil
	}
	return s.usageService.ListWithFilters(ctx, params, scopedFilters)
}

func (s *AgentManagementService) GetAgentUsageStats(ctx context.Context, actorID int64, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	if s.usageService == nil {
		return nil, ErrServiceUnavailable
	}
	scopedFilters, err := s.scopeAgentUsageFilters(ctx, actorID, filters)
	if err != nil {
		return nil, err
	}
	if scopedFilters.UserID == 0 && len(scopedFilters.UserIDs) == 0 {
		return &usagestats.UsageStats{}, nil
	}
	return s.usageService.GetStatsWithFilters(ctx, scopedFilters)
}

func (s *AgentManagementService) ListAgentUsageUsers(ctx context.Context, actorID int64) ([]User, error) {
	users, err := s.listCurrentUsageVisibleUsers(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AgentManagementService) CreateDirectUser(ctx context.Context, actorID int64, input CreateDirectUserInput) (*User, error) {
	if input.AllocatedConcurrency < 1 || input.AllocatedRPM < 0 {
		return nil, ErrAgentManagementInvalidAllocation
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}

	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (concurrencyRequestExceedsCapacity(totalConcurrency, usage.Concurrency, input.AllocatedConcurrency) || rpmRequestExceedsCapacity(totalRPM, usage.RPM, usage.UnlimitedRPM, input.AllocatedRPM)) {
		return nil, ErrAgentManagementAllocationExceeded
	}

	user := &User{
		Email:        input.Email,
		Username:     input.Username,
		Role:         RoleUser,
		ParentUserID: &actor.ID,
		Concurrency:  input.AllocatedConcurrency,
		RPMLimit:     input.AllocatedRPM,
		Status:       StatusActive,
	}
	if err := user.SetPassword(input.Password); err != nil {
		return nil, err
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	if err := s.ApplyInviteGroupDefaultsToChild(ctx, actor.ID, user.ID); err != nil {
		if s.userRepo != nil {
			_ = s.userRepo.HardDelete(ctx, user.ID)
		}
		return nil, err
	}
	if actor.Role != RoleAdmin {
		if err := s.recalculateAgentEffectiveQuota(ctx, actor.ID); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s *AgentManagementService) UpdateAllocation(ctx context.Context, actorID int64, childID int64, req AllocationUpdate) (*AllocationSummary, error) {
	requestedConcurrency := req.RequestedConcurrency()
	requestedRPM := req.RequestedRPM()
	if requestedConcurrency < 1 || requestedRPM < 0 {
		return nil, ErrAgentManagementInvalidAllocation
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return nil, err
	}
	if isProfileBackedChildRole(child.Role) {
		return s.updateProfileBackedChildPool(ctx, actor, child, requestedConcurrency, requestedRPM)
	}

	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, &child.ID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (concurrencyRequestExceedsCapacity(totalConcurrency, usage.Concurrency, requestedConcurrency) || rpmRequestExceedsCapacity(totalRPM, usage.RPM, usage.UnlimitedRPM, requestedRPM)) {
		return nil, ErrAgentManagementAllocationExceeded
	}
	if err := s.repo.SetEffectiveQuota(ctx, child.ID, requestedConcurrency, requestedRPM); err != nil {
		return nil, err
	}
	s.invalidateUser(ctx, child.ID)
	if actor.Role != RoleAdmin {
		if err := s.recalculateAgentEffectiveQuota(ctx, actor.ID); err != nil {
			return nil, err
		}
	}

	usage = usage.WithRequest(requestedConcurrency, requestedRPM)
	summary := buildAllocationSummary(actor, totalConcurrency, totalRPM, usage)
	return &summary, nil
}

func (s *AgentManagementService) UpdateChildNotes(ctx context.Context, actorID int64, childID int64, input AgentChildNotesUpdate) (*User, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return nil, err
	}
	notes := strings.TrimSpace(input.Notes)
	if len([]rune(notes)) > 500 {
		return nil, infraerrors.BadRequest("AGENT_MANAGEMENT_NOTES_TOO_LONG", "notes must be at most 500 characters")
	}
	if err := s.repo.UpsertChildNotes(ctx, actor.ID, child.ID, notes); err != nil {
		return nil, err
	}
	child.Notes = notes
	out := []User{*child}
	if err := s.populateProfileBackedUsers(ctx, out); err != nil {
		return nil, err
	}
	if err := s.populateAgentIncomeTotals(ctx, out); err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *AgentManagementService) UpdateInviteDefaults(ctx context.Context, actorID int64, input AgentInviteDefaultsUpdate) (*AgentProfile, error) {
	if input.InviteDefaultConcurrency < 1 || input.InviteDefaultRPM < 0 {
		return nil, ErrAgentManagementInvalidAllocation
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleAdmin {
		if err := s.updateAdminGlobalInviteDefaults(ctx, input.InviteDefaultConcurrency, input.InviteDefaultRPM); err != nil {
			return nil, err
		}
		return &AgentProfile{
			UserID:                   actor.ID,
			InviteDefaultConcurrency: input.InviteDefaultConcurrency,
			InviteDefaultRPM:         input.InviteDefaultRPM,
		}, nil
	}
	profile, err := s.repo.GetAgentProfile(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profile = &AgentProfile{
			UserID:                   actor.ID,
			InviteDefaultConcurrency: DefaultAgentInviteConcurrency,
			InviteDefaultRPM:         DefaultAgentInviteRPM,
		}
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	if concurrencyRequestExceedsCapacity(profile.PoolConcurrency, usage.Concurrency, input.InviteDefaultConcurrency) ||
		rpmRequestExceedsCapacity(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM, input.InviteDefaultRPM) {
		return nil, ErrAgentManagementAllocationExceeded
	}
	if err := s.repo.UpdateAgentInviteDefaults(ctx, actor.ID, input.InviteDefaultConcurrency, input.InviteDefaultRPM); err != nil {
		return nil, err
	}
	profile.InviteDefaultConcurrency = input.InviteDefaultConcurrency
	profile.InviteDefaultRPM = input.InviteDefaultRPM
	return profile, nil
}

func (s *AgentManagementService) inviteDefaultsForManager(ctx context.Context, actor *User) (AgentInviteDefaultsUpdate, error) {
	defaults := AgentInviteDefaultsUpdate{
		InviteDefaultConcurrency: DefaultAgentInviteConcurrency,
		InviteDefaultRPM:         DefaultAgentInviteRPM,
	}
	if actor == nil {
		return defaults, nil
	}
	if actor.Role == RoleAdmin {
		return s.adminGlobalInviteDefaults(ctx), nil
	}
	profile, err := s.repo.GetAgentProfile(ctx, actor.ID)
	if err != nil {
		return defaults, err
	}
	if profile != nil {
		defaults.InviteDefaultConcurrency = normalizedAgentInviteDefaultConcurrency(profile.InviteDefaultConcurrency, DefaultAgentInviteConcurrency)
		defaults.InviteDefaultRPM = normalizedAgentInviteDefaultRPM(profile.InviteDefaultRPM, DefaultAgentInviteRPM)
	}
	return defaults, nil
}

func (s *AgentManagementService) adminGlobalInviteDefaults(ctx context.Context) AgentInviteDefaultsUpdate {
	defaults := AgentInviteDefaultsUpdate{
		InviteDefaultConcurrency: DefaultAgentInviteConcurrency,
		InviteDefaultRPM:         DefaultAgentInviteRPM,
	}
	if s == nil || s.settingRepo == nil {
		return defaults
	}
	if value, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultConcurrency); err == nil {
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed >= 1 {
			defaults.InviteDefaultConcurrency = parsed
		}
	}
	if value, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultUserRPMLimit); err == nil {
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed >= 0 {
			defaults.InviteDefaultRPM = parsed
		}
	}
	return defaults
}

func (s *AgentManagementService) updateAdminGlobalInviteDefaults(ctx context.Context, concurrency int, rpm int) error {
	if s == nil || s.settingRepo == nil {
		return ErrServiceUnavailable
	}
	if err := s.settingRepo.Set(ctx, SettingKeyDefaultConcurrency, strconv.Itoa(concurrency)); err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingKeyDefaultUserRPMLimit, strconv.Itoa(rpm))
}

func (s *AgentManagementService) updateProfileBackedChildPool(ctx context.Context, actor *User, child *User, requestedConcurrency int, requestedRPM int) (*AllocationSummary, error) {
	childUsage, err := s.profileBackedChildUsage(ctx, child)
	if err != nil {
		return nil, err
	}
	if concurrencyRequestBelowAllocated(requestedConcurrency, childUsage.Concurrency) || rpmRequestBelowAllocated(requestedRPM, childUsage.RPM, childUsage.UnlimitedRPM) {
		return nil, &AgentPoolReclaimExceededError{
			AllocatedConcurrency: childUsage.Concurrency,
			RequestedConcurrency: requestedConcurrency,
			AllocatedRPM:         childUsage.RPM,
			RequestedRPM:         requestedRPM,
		}
	}

	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
	if err != nil {
		return nil, err
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, &child.ID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (concurrencyRequestExceedsCapacity(totalConcurrency, usage.Concurrency, requestedConcurrency) || rpmRequestExceedsCapacity(totalRPM, usage.RPM, usage.UnlimitedRPM, requestedRPM)) {
		return nil, ErrAgentManagementAllocationExceeded
	}

	if err := s.upsertProfileBackedChildPool(ctx, child.Role, child.ID, requestedConcurrency, requestedRPM); err != nil {
		return nil, err
	}
	if err := s.recalculateProfileBackedChildEffectiveQuota(ctx, child.Role, child.ID); err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin {
		if err := s.recalculateAgentEffectiveQuota(ctx, actor.ID); err != nil {
			return nil, err
		}
	}

	usage = usage.WithRequest(requestedConcurrency, requestedRPM)
	summary := buildAllocationSummary(actor, totalConcurrency, totalRPM, usage)
	return &summary, nil
}

func (s *AgentManagementService) SetAgentIncome(ctx context.Context, actorID int64, childID int64, input AgentIncomeSetInput) (*User, error) {
	if math.IsNaN(input.AgentIncome) || math.IsInf(input.AgentIncome, 0) {
		return nil, ErrAgentManagementInvalidIncome
	}
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin {
		return nil, ErrAgentManagementForbidden
	}
	rootAdmin, err := s.repo.GetRootAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
	}
	child, err := s.requireDirectChild(ctx, rootAdmin, childID)
	if err != nil {
		return nil, err
	}
	if child.Role != RoleAgentLevel1 {
		return nil, ErrAgentManagementUnsupportedRole
	}

	totals, err := s.repo.GetAgentIncomeTotals(ctx, []int64{child.ID})
	if err != nil {
		return nil, err
	}
	currentIncome := totals[child.ID]
	delta := input.AgentIncome - currentIncome
	if err := s.repo.AddAgentIncomeAdjustment(ctx, child.ID, actor.ID, delta, strings.TrimSpace(input.Reason)); err != nil {
		return nil, err
	}
	child.AgentIncome = input.AgentIncome
	return child, nil
}

func (s *AgentManagementService) profileBackedChildUsage(ctx context.Context, child *User) (QuotaUsageSummary, error) {
	if child.Role == RoleEnterprise {
		return s.repo.GetEnterpriseEmployeeQuotaUsage(ctx, child.ID, nil)
	}
	return s.repo.GetDirectChildQuotaUsage(ctx, child.ID, nil)
}

func (s *AgentManagementService) recalculateProfileBackedChildEffectiveQuota(ctx context.Context, role string, childID int64) error {
	if role == RoleEnterprise {
		return s.recalculateEnterpriseEffectiveQuota(ctx, childID)
	}
	return s.recalculateAgentEffectiveQuota(ctx, childID)
}

func (s *AgentManagementService) UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, input AgentUpgradeInput) (*User, error) {
	targetRole := input.TargetRole
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return nil, err
	}
	if child.Role != RoleUser {
		return nil, ErrAgentManagementInvalidTarget
	}
	if !canUpgradeDirectUser(actor.Role, targetRole) {
		return nil, ErrAgentManagementForbidden
	}
	if isProfileBackedChildRole(targetRole) {
		if input.PoolConcurrency < 1 || input.PoolRPM < 0 {
			return nil, ErrAgentManagementInvalidAllocation
		}
		totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
		if err != nil {
			return nil, err
		}
		usage, err := s.repo.GetDirectChildQuotaUsage(ctx, actor.ID, &child.ID)
		if err != nil {
			return nil, err
		}
		if actor.Role != RoleAdmin && (concurrencyRequestExceedsCapacity(totalConcurrency, usage.Concurrency, input.PoolConcurrency) || rpmRequestExceedsCapacity(totalRPM, usage.RPM, usage.UnlimitedRPM, input.PoolRPM)) {
			return nil, ErrAgentManagementAllocationExceeded
		}
		if err := s.upsertProfileBackedChildPool(ctx, targetRole, child.ID, input.PoolConcurrency, input.PoolRPM); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SetRoleAndParent(ctx, child.ID, targetRole, child.ParentUserID); err != nil {
		return nil, err
	}
	child.Role = targetRole
	if isAgentManagerRole(targetRole) && targetRole != RoleAdmin {
		if err := s.recalculateAgentEffectiveQuota(ctx, child.ID); err != nil {
			return nil, err
		}
	} else if targetRole == RoleEnterprise {
		if err := s.recalculateEnterpriseEffectiveQuota(ctx, child.ID); err != nil {
			return nil, err
		}
	}
	if actor.Role != RoleAdmin {
		if err := s.recalculateAgentEffectiveQuota(ctx, actor.ID); err != nil {
			return nil, err
		}
	}
	s.invalidateUser(ctx, child.ID)
	return child, nil
}

func (s *AgentManagementService) DeleteDirectChild(ctx context.Context, actorID int64, childID int64) error {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return err
	}
	rootAdmin, err := s.repo.GetRootAdmin(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
	}

	if actor.Role == RoleAdmin && child.Role == RoleAgentLevel1 {
		directChildren, listErr := s.listAllDirectChildrenForCascade(ctx, child.ID)
		if listErr != nil {
			return listErr
		}
		if err := s.repo.DeleteLevel1AgentAndMoveChildren(ctx, child.ID, rootAdmin.ID); err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		for i := range directChildren {
			s.invalidateUser(ctx, directChildren[i].ID)
		}
		return nil
	}
	if actor.Role == RoleAdmin && child.Role == RoleUser {
		if err := s.userRepo.HardDelete(ctx, child.ID); err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		return nil
	}
	if actor.Role == RoleAdmin && child.Role == RoleEnterprise {
		if s.enterpriseCleanupRepo == nil {
			return ErrAgentManagementNotImplemented
		}
		affectedUserIDs, err := s.enterpriseCleanupRepo.HardDeleteEnterpriseWithEmployees(ctx, child.ID)
		if err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		for i := range affectedUserIDs {
			s.invalidateUser(ctx, affectedUserIDs[i])
		}
		return nil
	}
	if child.Role == RoleUser || child.Role == RoleEnterprise {
		if err := s.repo.RehomeChildGroupDelegations(ctx, actor.ID, rootAdmin.ID, child.ID); err != nil {
			return err
		}
		if err := s.repo.SetParent(ctx, child.ID, &rootAdmin.ID); err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		return nil
	}

	return ErrAgentManagementForbidden
}

func (s *AgentManagementService) ListMyGroups(ctx context.Context, actorID int64) ([]AgentGroupRate, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if s.groupRepo == nil {
		return []AgentGroupRate{}, nil
	}
	if s.repo == nil {
		return []AgentGroupRate{}, nil
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentGroupRate, 0, len(groups))
	for i := range groups {
		if groups[i].IsExclusive && actor.Role != RoleAdmin {
			continue
		}
		canDelegate := false
		source := "public"
		if groups[i].IsExclusive {
			canDelegate = true
			source = "admin_exclusive"
		}
		out = append(out, AgentGroupRate{
			Group:         groups[i],
			EffectiveRate: groups[i].RateMultiplier,
			CanDelegate:   canDelegate,
			Source:        source,
		})
	}
	delegations, err := s.repo.ListGroupDelegationsForChild(ctx, actorID)
	if err != nil {
		return nil, err
	}
	groupsByID := make(map[int64]Group, len(groups))
	for i := range groups {
		groupsByID[groups[i].ID] = groups[i]
	}
	for i := range delegations {
		group, ok := groupsByID[delegations[i].GroupID]
		if !ok && delegations[i].Group != nil {
			group = *delegations[i].Group
			ok = true
		}
		if !ok {
			continue
		}
		if !group.IsExclusive || !group.IsActive() {
			continue
		}
		group.RateMultiplier = delegations[i].RateMultiplier
		out = append(out, AgentGroupRate{
			Group:         group,
			EffectiveRate: delegations[i].RateMultiplier,
			CanDelegate:   delegations[i].CanDelegate,
			Source:        "delegated",
		})
	}
	return out, nil
}

func (s *AgentManagementService) ListChildGroupDelegationOptions(ctx context.Context, actorID int64, childID int64) ([]ChildGroupDelegationOption, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return nil, err
	}
	groups, err := s.ListMyGroups(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	delegations, err := s.repo.ListGroupDelegationsForChild(ctx, child.ID)
	if err != nil {
		return nil, err
	}
	assignedByGroupID := make(map[int64]AgentGroupDelegation, len(delegations))
	for i := range delegations {
		if delegations[i].ManagerUserID != actor.ID {
			continue
		}
		assignedByGroupID[delegations[i].GroupID] = delegations[i]
	}

	out := make([]ChildGroupDelegationOption, 0, len(groups))
	for i := range groups {
		if !groups[i].Group.IsExclusive || !groups[i].CanDelegate {
			continue
		}
		group := groups[i].Group
		group.RateMultiplier = groups[i].EffectiveRate
		option := ChildGroupDelegationOption{
			Group:               group,
			EffectiveRate:       groups[i].EffectiveRate,
			CanDelegate:         groups[i].CanDelegate,
			Source:              groups[i].Source,
			ChildRateMultiplier: groups[i].EffectiveRate,
			ChildCanDelegate:    false,
		}
		if delegation, ok := assignedByGroupID[groups[i].Group.ID]; ok {
			option.Assigned = true
			option.ChildRateMultiplier = delegation.RateMultiplier
			option.ChildCanDelegate = delegation.CanDelegate
		}
		out = append(out, option)
	}
	return out, nil
}

func (s *AgentManagementService) SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, input ChildGroupDelegationInput) error {
	if input.RateMultiplier <= 0 {
		return ErrAgentManagementInvalidGroupRate
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	if s.groupRepo == nil {
		return ErrAgentManagementInvalidGroup
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return err
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	if !group.IsActive() || !group.IsExclusive {
		return ErrAgentManagementInvalidGroup
	}
	access, err := s.requireGroupDelegationAccess(ctx, actor, groupID)
	if err != nil {
		return err
	}
	if input.RateMultiplier < access.minimumRate {
		return ErrAgentManagementGroupRateBelowCost
	}
	existing, err := s.repo.GetGroupDelegation(ctx, actor.ID, child.ID, groupID)
	if err != nil {
		return err
	}
	if err := s.repo.UpsertGroupDelegation(ctx, actor.ID, child.ID, groupID, input.RateMultiplier, input.CanDelegate); err != nil {
		return err
	}
	if s.userRepo != nil {
		if err := s.userRepo.AddGroupToAllowedGroups(ctx, child.ID, groupID); err != nil {
			return err
		}
	}
	if err := s.syncDelegatedUserGroupRate(ctx, child.ID, groupID, &input.RateMultiplier); err != nil {
		return err
	}
	if err := s.raiseManagedGroupRateFloorIfNeeded(ctx, actor, child, groupID, input, existing); err != nil {
		return err
	}
	s.invalidateUser(ctx, child.ID)
	if existing != nil && existing.CanDelegate && !input.CanDelegate {
		if err := s.repo.DeleteInviteGroupDefault(ctx, child.ID, groupID); err != nil {
			return err
		}
		if err := s.removeDelegatedGroupFromDescendants(ctx, child.ID, groupID); err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentManagementService) SetChildGroupDelegationsBatch(ctx context.Context, actorID int64, childID int64, input ChildGroupDelegationBatchInput) error {
	if input.RateMultiplier <= 0 {
		return ErrAgentManagementInvalidGroupRate
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return err
	}
	groupIDs, err := s.resolveChildGroupDelegationBatchGroupIDs(ctx, actorID, childID, input.GroupIDs, input.All)
	if err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		existing, err := s.repo.GetGroupDelegation(ctx, actor.ID, child.ID, groupID)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		if err := s.SetChildGroupDelegation(ctx, actorID, childID, groupID, ChildGroupDelegationInput{
			RateMultiplier: input.RateMultiplier,
			CanDelegate:    input.CanDelegate,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentManagementService) SetDirectChildrenGroupDelegationsBatch(ctx context.Context, actorID int64, kind DirectChildKind, input DirectChildrenGroupDelegationBatchInput) (int, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return 0, err
	}
	roles, err := rolesForDirectChildKind(kind)
	if err != nil {
		return 0, err
	}
	groupIDs, err := s.resolveManagerGroupDelegationBatchGroupIDs(ctx, actor.ID, input.GroupIDs, input.All)
	if err != nil {
		return 0, err
	}
	groupRates, err := s.directChildrenGroupDelegationBatchRates(ctx, actor, groupIDs)
	if err != nil {
		return 0, err
	}
	children, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, roles)
	if err != nil {
		return 0, err
	}
	allChildren := input.AllChildren == nil || *input.AllChildren
	if !allChildren && len(input.ChildIDs) == 0 {
		return 0, ErrAgentManagementInvalidGroupSelection
	}
	if !allChildren {
		selected := map[int64]struct{}{}
		for _, id := range input.ChildIDs {
			if id > 0 {
				selected[id] = struct{}{}
			}
		}
		filtered := children[:0]
		for i := range children {
			if _, ok := selected[children[i].ID]; ok {
				filtered = append(filtered, children[i])
			}
		}
		children = filtered
	}
	updatedChildren := 0
	for i := range children {
		childUpdated := false
		for _, groupID := range groupIDs {
			existing, err := s.repo.GetGroupDelegation(ctx, actor.ID, children[i].ID, groupID)
			if err != nil {
				return 0, err
			}
			if existing != nil {
				continue
			}
			rate, ok := groupRates[groupID]
			if !ok || rate <= 0 {
				return 0, ErrAgentManagementInvalidGroupRate
			}
			if err := s.SetChildGroupDelegation(ctx, actor.ID, children[i].ID, groupID, ChildGroupDelegationInput{
				RateMultiplier: rate,
				CanDelegate:    input.CanDelegate,
			}); err != nil {
				return 0, err
			}
			childUpdated = true
		}
		if childUpdated {
			updatedChildren++
		}
	}
	return updatedChildren, nil
}

func (s *AgentManagementService) directChildrenGroupDelegationBatchRates(ctx context.Context, actor *User, groupIDs []int64) (map[int64]float64, error) {
	options, err := s.ListInviteGroupDefaultOptions(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	selected := make(map[int64]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID > 0 {
			selected[groupID] = struct{}{}
		}
	}
	rates := make(map[int64]float64, len(selected))
	for i := range options {
		groupID := options[i].Group.ID
		if _, ok := selected[groupID]; !ok {
			continue
		}
		if !options[i].Group.IsExclusive || !options[i].CanDelegate {
			continue
		}
		rate := options[i].ChildRateMultiplier
		if rate <= 0 {
			rate = options[i].EffectiveRate
		}
		access, err := s.requireGroupDelegationAccess(ctx, actor, groupID)
		if err != nil {
			return nil, err
		}
		if rate < access.minimumRate {
			rate = access.minimumRate
		}
		if rate <= 0 {
			return nil, ErrAgentManagementInvalidGroupRate
		}
		rates[groupID] = rate
	}
	if len(rates) != len(selected) {
		return nil, ErrAgentManagementInvalidGroupSelection
	}
	return rates, nil
}

func (s *AgentManagementService) ListDirectChildrenWithGroupDelegation(ctx context.Context, actorID int64, kind DirectChildKind, query DirectChildrenGroupQuery) (*DirectChildrenResult, error) {
	if query.GroupID <= 0 {
		return nil, ErrAgentManagementInvalidGroup
	}
	return s.listDirectChildrenByGroupDelegationState(ctx, actorID, kind, query, func(managerID int64, child User) (bool, map[int64]float64, error) {
		existing, err := s.repo.GetGroupDelegation(ctx, managerID, child.ID, query.GroupID)
		if err != nil {
			return false, nil, err
		}
		if existing == nil {
			return false, nil, nil
		}
		return true, map[int64]float64{query.GroupID: existing.RateMultiplier}, nil
	})
}

func (s *AgentManagementService) ListDirectChildrenWithoutGroupDelegation(ctx context.Context, actorID int64, kind DirectChildKind, query DirectChildrenGroupQuery) (*DirectChildrenResult, error) {
	groupIDs, err := normalizeBatchGroupIDs(append(query.GroupIDs, query.GroupID))
	if err != nil {
		return nil, err
	}
	query.GroupIDs = groupIDs
	return s.listDirectChildrenByGroupDelegationState(ctx, actorID, kind, query, func(managerID int64, child User) (bool, map[int64]float64, error) {
		assigned := map[int64]float64{}
		for _, groupID := range groupIDs {
			existing, err := s.repo.GetGroupDelegation(ctx, managerID, child.ID, groupID)
			if err != nil {
				return false, nil, err
			}
			if existing == nil {
				return true, nil, nil
			}
			assigned[groupID] = existing.RateMultiplier
		}
		return false, assigned, nil
	})
}

func (s *AgentManagementService) listDirectChildrenByGroupDelegationState(ctx context.Context, actorID int64, kind DirectChildKind, query DirectChildrenGroupQuery, include func(int64, User) (bool, map[int64]float64, error)) (*DirectChildrenResult, error) {
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	roles, err := rolesForDirectChildKind(kind)
	if err != nil {
		return nil, err
	}
	allChildren, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, roles)
	if err != nil {
		return nil, err
	}
	search := strings.TrimSpace(strings.ToLower(query.Search))
	filtered := make([]User, 0, len(allChildren))
	for i := range allChildren {
		child := allChildren[i]
		if search != "" {
			email := strings.ToLower(child.Email)
			username := strings.ToLower(child.Username)
			if !strings.Contains(email, search) && !strings.Contains(username, search) {
				continue
			}
		}
		matched, groupRates, err := include(actor.ID, child)
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		if len(groupRates) > 0 {
			child.GroupRates = groupRates
		}
		filtered = append(filtered, child)
	}
	if err := s.populateProfileBackedUsers(ctx, filtered); err != nil {
		return nil, err
	}
	if err := s.populateAgentIncomeTotals(ctx, filtered); err != nil {
		return nil, err
	}
	page := query.Pagination.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.Pagination.PageSize
	if pageSize <= 0 {
		pageSize = pagination.DefaultPagination().PageSize
	}
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := append([]User(nil), filtered[start:end]...)
	pages := 0
	if len(filtered) > 0 {
		pages = len(filtered) / pageSize
		if len(filtered)%pageSize > 0 {
			pages++
		}
	}
	return &DirectChildrenResult{
		Users:      items,
		Pagination: &pagination.PaginationResult{Total: int64(len(filtered)), Page: page, PageSize: pageSize, Pages: pages},
	}, nil
}

func (s *AgentManagementService) UpdateDirectChildrenExistingGroupDelegations(ctx context.Context, actorID int64, kind DirectChildKind, input DirectChildrenGroupDelegationUpdateInput) (*DirectChildrenGroupDelegationUpdateResult, error) {
	if input.GroupID <= 0 {
		return nil, ErrAgentManagementInvalidGroup
	}
	if input.RateMultiplier == nil && input.CanDelegate == nil {
		return nil, ErrAgentManagementInvalidGroupSelection
	}
	if input.RateMultiplier != nil && *input.RateMultiplier <= 0 {
		return nil, ErrAgentManagementInvalidGroupRate
	}
	if !input.All && len(input.ChildIDs) == 0 {
		return nil, ErrAgentManagementInvalidGroupSelection
	}

	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	roles, err := rolesForDirectChildKind(kind)
	if err != nil {
		return nil, err
	}
	children, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, roles)
	if err != nil {
		return nil, err
	}

	selected := map[int64]struct{}{}
	for _, id := range input.ChildIDs {
		if id > 0 {
			selected[id] = struct{}{}
		}
	}

	result := &DirectChildrenGroupDelegationUpdateResult{
		Kind:              kind,
		GroupID:           input.GroupID,
		RequestedChildIDs: append([]int64(nil), input.ChildIDs...),
		All:               input.All,
	}
	for i := range children {
		if !input.All {
			if _, ok := selected[children[i].ID]; !ok {
				continue
			}
		}
		existing, err := s.repo.GetGroupDelegation(ctx, actor.ID, children[i].ID, input.GroupID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			result.SkippedChildren++
			continue
		}
		rate := existing.RateMultiplier
		if input.RateMultiplier != nil {
			rate = *input.RateMultiplier
		}
		canDelegate := existing.CanDelegate
		if input.CanDelegate != nil {
			canDelegate = *input.CanDelegate
		}
		if err := s.SetChildGroupDelegation(ctx, actor.ID, children[i].ID, input.GroupID, ChildGroupDelegationInput{
			RateMultiplier: rate,
			CanDelegate:    canDelegate,
		}); err != nil {
			return nil, err
		}
		result.UpdatedChildren++
	}
	return result, nil
}

func (s *AgentManagementService) RemoveDirectChildrenGroupDelegationsBatch(ctx context.Context, actorID int64, kind DirectChildKind, input DirectChildrenGroupDelegationReclaimInput) (*DirectChildrenGroupDelegationReclaimResult, error) {
	if input.GroupID <= 0 {
		return nil, ErrAgentManagementInvalidGroup
	}
	if !input.All && len(input.ChildIDs) == 0 {
		return nil, ErrAgentManagementInvalidGroupSelection
	}

	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	roles, err := rolesForDirectChildKind(kind)
	if err != nil {
		return nil, err
	}
	children, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, roles)
	if err != nil {
		return nil, err
	}

	selected := map[int64]struct{}{}
	for _, id := range input.ChildIDs {
		if id > 0 {
			selected[id] = struct{}{}
		}
	}

	result := &DirectChildrenGroupDelegationReclaimResult{
		Kind:              kind,
		GroupID:           input.GroupID,
		RequestedChildIDs: append([]int64(nil), input.ChildIDs...),
		All:               input.All,
	}
	for i := range children {
		if !input.All {
			if _, ok := selected[children[i].ID]; !ok {
				continue
			}
		}
		existing, err := s.repo.GetGroupDelegation(ctx, actor.ID, children[i].ID, input.GroupID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			result.SkippedChildren++
			continue
		}
		if err := s.RemoveChildGroupDelegation(ctx, actor.ID, children[i].ID, input.GroupID); err != nil {
			return nil, err
		}
		result.RemovedChildren++
	}
	return result, nil
}

func (s *AgentManagementService) RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteGroupDelegation(ctx, actor.ID, child.ID, groupID); err != nil {
		return err
	}
	if err := s.removeGroupFromUserAndDelegatedDescendants(ctx, child.ID, groupID); err != nil {
		return err
	}
	return nil
}

func (s *AgentManagementService) ListInviteGroupDefaultOptions(ctx context.Context, actorID int64) ([]ChildGroupDelegationOption, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	groups, err := s.ListMyGroups(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	defaults, err := s.repo.ListInviteGroupDefaults(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	defaultsByGroupID := make(map[int64]AgentInviteGroupDefault, len(defaults))
	for i := range defaults {
		defaultsByGroupID[defaults[i].GroupID] = defaults[i]
	}

	out := make([]ChildGroupDelegationOption, 0, len(groups))
	for i := range groups {
		if !groups[i].Group.IsExclusive || !groups[i].CanDelegate {
			continue
		}
		group := groups[i].Group
		group.RateMultiplier = groups[i].EffectiveRate
		option := ChildGroupDelegationOption{
			Group:               group,
			EffectiveRate:       groups[i].EffectiveRate,
			CanDelegate:         groups[i].CanDelegate,
			Source:              groups[i].Source,
			ChildRateMultiplier: groups[i].EffectiveRate,
			ChildCanDelegate:    false,
		}
		if configured, ok := defaultsByGroupID[groups[i].Group.ID]; ok {
			option.Assigned = true
			option.ChildRateMultiplier = configured.RateMultiplier
		}
		out = append(out, option)
	}
	return out, nil
}

func (s *AgentManagementService) SetInviteGroupDefault(ctx context.Context, actorID int64, groupID int64, input AgentInviteGroupDefaultInput) error {
	if input.RateMultiplier <= 0 {
		return ErrAgentManagementInvalidGroupRate
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	if s.groupRepo == nil {
		return ErrAgentManagementInvalidGroup
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return err
	}
	if !group.IsActive() || !group.IsExclusive {
		return ErrAgentManagementInvalidGroup
	}
	access, err := s.requireGroupDelegationAccess(ctx, actor, groupID)
	if err != nil {
		return err
	}
	if input.RateMultiplier < access.minimumRate {
		return ErrAgentManagementGroupRateBelowCost
	}
	return s.repo.UpsertInviteGroupDefault(ctx, actor.ID, groupID, input.RateMultiplier)
}

func (s *AgentManagementService) SetInviteGroupDefaultsBatch(ctx context.Context, actorID int64, input AgentInviteGroupDefaultBatchInput) error {
	if input.RateMultiplier <= 0 {
		return ErrAgentManagementInvalidGroupRate
	}
	groupIDs, err := s.resolveInviteGroupDefaultBatchGroupIDs(ctx, actorID, input.GroupIDs, input.All)
	if err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if err := s.SetInviteGroupDefault(ctx, actorID, groupID, AgentInviteGroupDefaultInput{RateMultiplier: input.RateMultiplier}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentManagementService) RemoveInviteGroupDefault(ctx context.Context, actorID int64, groupID int64) error {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	return s.repo.DeleteInviteGroupDefault(ctx, actor.ID, groupID)
}

func (s *AgentManagementService) ApplyInviteGroupDefaultsToChild(ctx context.Context, actorID int64, childID int64) error {
	if actorID <= 0 || childID <= 0 || s == nil || s.repo == nil {
		return nil
	}
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return err
	}
	child, err := s.requireDirectChild(ctx, actor, childID)
	if err != nil {
		return err
	}
	defaults, err := s.repo.ListInviteGroupDefaults(ctx, actor.ID)
	if err != nil {
		return err
	}
	for i := range defaults {
		if defaults[i].RateMultiplier <= 0 {
			continue
		}
		rate := defaults[i].RateMultiplier
		access, accessErr := s.requireGroupDelegationAccess(ctx, actor, defaults[i].GroupID)
		if accessErr != nil && !errors.Is(accessErr, ErrAgentManagementForbidden) {
			return accessErr
		}
		if accessErr == nil && rate < access.minimumRate {
			rate = access.minimumRate
		}
		if err := s.repo.UpsertGroupDelegation(ctx, actor.ID, child.ID, defaults[i].GroupID, rate, false); err != nil {
			return err
		}
		if s.userRepo != nil {
			if err := s.userRepo.AddGroupToAllowedGroups(ctx, child.ID, defaults[i].GroupID); err != nil {
				return err
			}
		}
		if err := s.syncDelegatedUserGroupRate(ctx, child.ID, defaults[i].GroupID, &rate); err != nil {
			return err
		}
	}
	s.invalidateUser(ctx, child.ID)
	return nil
}

func (s *AgentManagementService) syncDelegatedUserGroupRate(ctx context.Context, userID int64, groupID int64, rate *float64) error {
	if s == nil || s.userGroupRateRepo == nil || userID <= 0 || groupID <= 0 {
		return nil
	}
	return s.userGroupRateRepo.SyncUserGroupRates(ctx, userID, map[int64]*float64{groupID: rate})
}

func (s *AgentManagementService) removeGroupFromUserAndDelegatedDescendants(ctx context.Context, userID int64, groupID int64) error {
	if isAgentManagerRoleForInviteDefaults(ctx, s.userRepo, userID) {
		if err := s.repo.DeleteInviteGroupDefault(ctx, userID, groupID); err != nil {
			return err
		}
	}
	if s.userRepo != nil {
		if err := s.userRepo.RemoveGroupFromUserAllowedGroups(ctx, userID, groupID); err != nil {
			return err
		}
	}
	if err := s.syncDelegatedUserGroupRate(ctx, userID, groupID, nil); err != nil {
		return err
	}
	s.invalidateUser(ctx, userID)

	children, err := s.listAllDirectChildrenForCascade(ctx, userID)
	if err != nil {
		return err
	}
	for i := range children {
		if err := s.repo.DeleteGroupDelegation(ctx, userID, children[i].ID, groupID); err != nil {
			return err
		}
		if err := s.removeGroupFromUserAndDelegatedDescendants(ctx, children[i].ID, groupID); err != nil {
			return err
		}
	}
	return nil
}

func isAgentManagerRoleForInviteDefaults(ctx context.Context, repo UserRepository, userID int64) bool {
	if repo == nil || userID <= 0 {
		return false
	}
	user, err := repo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	return user.Role == RoleAdmin || user.Role == RoleAgentLevel1
}

func (s *AgentManagementService) removeDelegatedGroupFromDescendants(ctx context.Context, managerID int64, groupID int64) error {
	children, err := s.listAllDirectChildrenForCascade(ctx, managerID)
	if err != nil {
		return err
	}
	for i := range children {
		if err := s.repo.DeleteGroupDelegation(ctx, managerID, children[i].ID, groupID); err != nil {
			return err
		}
		if err := s.removeGroupFromUserAndDelegatedDescendants(ctx, children[i].ID, groupID); err != nil {
			return err
		}
	}
	return nil
}

func (s *AgentManagementService) listAllDirectChildrenForCascade(ctx context.Context, parentID int64) ([]User, error) {
	return s.listAllDirectChildrenForRoles(ctx, parentID, []string{RoleUser, RoleEnterprise, RoleAgentLevel1, RoleEmployee})
}

func (s *AgentManagementService) listAllDirectChildrenForRoles(ctx context.Context, parentID int64, roles []string) ([]User, error) {
	const pageSize = 1000
	var out []User
	for page := 1; ; page++ {
		children, result, err := s.repo.ListDirectChildren(ctx, parentID, roles, pagination.PaginationParams{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		out = append(out, children...)
		if len(children) == 0 || result == nil || page >= result.Pages {
			break
		}
	}
	return out, nil
}

func (s *AgentManagementService) buildSubordinateStructureForOwner(ctx context.Context, ownerOptions []User, selectedOwner User) (*SubordinateStructureResult, error) {
	users, err := s.listAllDirectChildrenForRoles(ctx, selectedOwner.ID, []string{RoleUser})
	if err != nil {
		return nil, err
	}
	enterprises, err := s.listAllDirectChildrenForRoles(ctx, selectedOwner.ID, []string{RoleEnterprise})
	if err != nil {
		return nil, err
	}
	if err := s.populateProfileBackedUsers(ctx, enterprises); err != nil {
		return nil, err
	}
	enterpriseNodes := make([]AdminAgentTreeEnterprise, 0, len(enterprises))
	for i := range enterprises {
		employees, err := s.listAllDirectChildrenForRoles(ctx, enterprises[i].ID, []string{RoleEmployee})
		if err != nil {
			return nil, err
		}
		enterpriseNodes = append(enterpriseNodes, AdminAgentTreeEnterprise{
			Enterprise: enterprises[i],
			Employees:  employees,
		})
	}
	return &SubordinateStructureResult{
		OwnerOptions:  ownerOptions,
		SelectedOwner: selectedOwner,
		Users:         users,
		Enterprises:   enterpriseNodes,
	}, nil
}

func (s *AgentManagementService) scopeAgentUsageFilters(ctx context.Context, actorID int64, filters usagestats.UsageLogFilters) (usagestats.UsageLogFilters, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return filters, err
	}
	if actor.Role != RoleAgentLevel1 {
		return filters, ErrAgentManagementForbidden
	}
	visibleUsers, err := s.listCurrentUsageVisibleUsersForActor(ctx, actor)
	if err != nil {
		return filters, err
	}
	visibleIDs := make([]int64, 0, len(visibleUsers))
	visibleSet := make(map[int64]struct{}, len(visibleUsers))
	for i := range visibleUsers {
		visibleIDs = append(visibleIDs, visibleUsers[i].ID)
		visibleSet[visibleUsers[i].ID] = struct{}{}
	}

	filters.UserIDs = nil
	if filters.UserID > 0 {
		if _, ok := visibleSet[filters.UserID]; !ok {
			return filters, ErrAgentManagementForbidden
		}
		return filters, nil
	}
	filters.UserIDs = visibleIDs
	return filters, nil
}

func (s *AgentManagementService) listCurrentUsageVisibleUsers(ctx context.Context, actorID int64) ([]User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAgentLevel1 {
		return nil, ErrAgentManagementForbidden
	}
	return s.listCurrentUsageVisibleUsersForActor(ctx, actor)
}

func (s *AgentManagementService) listCurrentUsageVisibleUsersForActor(ctx context.Context, actor *User) ([]User, error) {
	if actor == nil || actor.Role != RoleAgentLevel1 {
		return nil, ErrAgentManagementForbidden
	}
	users, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, []string{RoleUser})
	if err != nil {
		return nil, err
	}
	enterprises, err := s.listAllDirectChildrenForRoles(ctx, actor.ID, []string{RoleEnterprise})
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(users)+len(enterprises))
	out = append(out, users...)
	for i := range enterprises {
		out = append(out, enterprises[i])
		employees, err := s.listAllDirectChildrenForRoles(ctx, enterprises[i].ID, []string{RoleEmployee})
		if err != nil {
			return nil, err
		}
		out = append(out, employees...)
	}
	return out, nil
}

func (s *AgentManagementService) ResolveInvitationParent(ctx context.Context, inviterID int64) (*int64, error) {
	if inviterID <= 0 {
		return nil, ErrAgentManagementInvalidTarget
	}
	inviter, err := s.userRepo.GetByID(ctx, inviterID)
	if err != nil {
		return nil, err
	}
	if inviter.Role == RoleAdmin {
		rootAdmin, err := s.repo.GetRootAdmin(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
		}
		return &rootAdmin.ID, nil
	}
	if isAgentManagerRole(inviter.Role) {
		return &inviter.ID, nil
	}

	visited := map[int64]struct{}{inviter.ID: {}}
	parentID := inviter.ParentUserID
	for parentID != nil {
		if _, ok := visited[*parentID]; ok {
			break
		}
		visited[*parentID] = struct{}{}
		parent, err := s.userRepo.GetByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if isAgentManagerRole(parent.Role) && parent.Role != RoleAdmin {
			return &parent.ID, nil
		}
		if parent.Role == RoleAdmin {
			rootAdmin, err := s.repo.GetRootAdmin(ctx)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
			}
			return &rootAdmin.ID, nil
		}
		parentID = parent.ParentUserID
	}

	rootAdmin, err := s.repo.GetRootAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
	}
	return &rootAdmin.ID, nil
}

func (s *AgentManagementService) listDirectChildren(ctx context.Context, actorID int64, roles []string, params pagination.PaginationParams) (*DirectChildrenResult, error) {
	return s.listDirectChildrenWithQuery(ctx, actorID, roles, DirectChildrenQuery{Pagination: params})
}

func (s *AgentManagementService) listDirectChildrenWithQuery(ctx context.Context, actorID int64, roles []string, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.listDirectChildrenForActorWithQuery(ctx, actor, roles, query)
}

func (s *AgentManagementService) listDirectChildrenForActor(ctx context.Context, actor *User, roles []string, params pagination.PaginationParams) (*DirectChildrenResult, error) {
	return s.listDirectChildrenForActorWithQuery(ctx, actor, roles, DirectChildrenQuery{Pagination: params})
}

func (s *AgentManagementService) listDirectChildrenForActorWithQuery(ctx context.Context, actor *User, roles []string, query DirectChildrenQuery) (*DirectChildrenResult, error) {
	if query.Pagination.PageSize == 0 {
		query.Pagination = pagination.DefaultPagination()
	}
	users, page, err := s.repo.ListDirectChildrenWithSearch(ctx, actor.ID, roles, query.Pagination, query.Search)
	if err != nil {
		return nil, err
	}
	if err := s.populateProfileBackedUsers(ctx, users); err != nil {
		return nil, err
	}
	if err := s.populateAgentIncomeTotals(ctx, users); err != nil {
		return nil, err
	}
	if err := s.populateChildNotes(ctx, actor.ID, users); err != nil {
		return nil, err
	}
	return &DirectChildrenResult{Users: users, Pagination: page}, nil
}

func (s *AgentManagementService) populateChildNotes(ctx context.Context, managerID int64, users []User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].ID)
	}
	notesByChildID, err := s.repo.GetChildNotes(ctx, managerID, ids)
	if err != nil {
		return err
	}
	for i := range users {
		users[i].Notes = notesByChildID[users[i].ID]
	}
	return nil
}

func (s *AgentManagementService) populateProfileBackedUsers(ctx context.Context, users []User) error {
	for i := range users {
		if isAgentManagerRole(users[i].Role) && users[i].Role != RoleAdmin {
			profile, err := s.repo.GetAgentProfile(ctx, users[i].ID)
			if err != nil {
				return err
			}
			users[i].AgentProfile = profile
			continue
		}
		if users[i].Role == RoleEnterprise {
			profile, err := s.repo.GetEnterpriseProfile(ctx, users[i].ID)
			if err != nil {
				return err
			}
			users[i].EnterpriseProfile = profile
		}
	}
	return nil
}

func (s *AgentManagementService) populateAgentIncomeTotals(ctx context.Context, users []User) error {
	ids := make([]int64, 0, len(users))
	for i := range users {
		if users[i].Role == RoleAgentLevel1 {
			ids = append(ids, users[i].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	totals, err := s.repo.GetAgentIncomeTotals(ctx, ids)
	if err != nil {
		return err
	}
	for i := range users {
		if income, ok := totals[users[i].ID]; ok {
			users[i].AgentIncome = income
		}
	}
	return nil
}

func (s *AgentManagementService) requireManager(ctx context.Context, actorID int64) (*User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !isAgentManagerRole(actor.Role) {
		return nil, ErrAgentManagementForbidden
	}
	if actor.Role == RoleAdmin {
		rootAdmin, err := s.repo.GetRootAdmin(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAgentManagementRootAdminNotPresent, err)
		}
		return rootAdmin, nil
	}
	return actor, nil
}

func (s *AgentManagementService) requireDirectChild(ctx context.Context, actor *User, childID int64) (*User, error) {
	child, err := s.userRepo.GetByID(ctx, childID)
	if err != nil {
		return nil, err
	}
	if child.ParentUserID == nil || *child.ParentUserID != actor.ID {
		return nil, ErrAgentManagementNotDirectChild
	}
	return child, nil
}

type agentGroupDelegationAccess struct {
	minimumRate float64
}

func (s *AgentManagementService) requireGroupDelegationAccess(ctx context.Context, actor *User, groupID int64) (agentGroupDelegationAccess, error) {
	if actor.Role == RoleAdmin {
		return agentGroupDelegationAccess{}, nil
	}
	if actor.ParentUserID == nil {
		return agentGroupDelegationAccess{}, ErrAgentManagementForbidden
	}
	delegation, err := s.repo.GetGroupDelegation(ctx, *actor.ParentUserID, actor.ID, groupID)
	if err != nil {
		return agentGroupDelegationAccess{}, err
	}
	if delegation == nil || !delegation.CanDelegate {
		return agentGroupDelegationAccess{}, ErrAgentManagementForbidden
	}
	return agentGroupDelegationAccess{minimumRate: delegation.RateMultiplier}, nil
}

func (s *AgentManagementService) resolveChildGroupDelegationBatchGroupIDs(ctx context.Context, actorID int64, childID int64, requested []int64, all bool) ([]int64, error) {
	if all {
		options, err := s.ListChildGroupDelegationOptions(ctx, actorID, childID)
		if err != nil {
			return nil, err
		}
		out := make([]int64, 0, len(options))
		for i := range options {
			if options[i].Group.ID > 0 && options[i].Group.IsExclusive && options[i].CanDelegate {
				out = append(out, options[i].Group.ID)
			}
		}
		if len(out) == 0 {
			return nil, ErrAgentManagementInvalidGroupSelection
		}
		return out, nil
	}
	return normalizeBatchGroupIDs(requested)
}

func (s *AgentManagementService) resolveManagerGroupDelegationBatchGroupIDs(ctx context.Context, actorID int64, requested []int64, all bool) ([]int64, error) {
	if all {
		groups, err := s.ListMyGroups(ctx, actorID)
		if err != nil {
			return nil, err
		}
		out := make([]int64, 0, len(groups))
		for i := range groups {
			if groups[i].Group.ID > 0 && groups[i].Group.IsExclusive && groups[i].CanDelegate {
				out = append(out, groups[i].Group.ID)
			}
		}
		if len(out) == 0 {
			return nil, ErrAgentManagementInvalidGroupSelection
		}
		return out, nil
	}
	return normalizeBatchGroupIDs(requested)
}

func (s *AgentManagementService) resolveInviteGroupDefaultBatchGroupIDs(ctx context.Context, actorID int64, requested []int64, all bool) ([]int64, error) {
	if all {
		options, err := s.ListInviteGroupDefaultOptions(ctx, actorID)
		if err != nil {
			return nil, err
		}
		out := make([]int64, 0, len(options))
		for i := range options {
			if options[i].Group.ID > 0 && options[i].Group.IsExclusive && options[i].CanDelegate {
				out = append(out, options[i].Group.ID)
			}
		}
		if len(out) == 0 {
			return nil, ErrAgentManagementInvalidGroupSelection
		}
		return out, nil
	}
	return normalizeBatchGroupIDs(requested)
}

func normalizeBatchGroupIDs(requested []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(requested))
	out := make([]int64, 0, len(requested))
	for _, id := range requested {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrAgentManagementInvalidGroupSelection
	}
	return out, nil
}

func (s *AgentManagementService) invalidateUser(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func (s *AgentManagementService) raiseManagedGroupRateFloorIfNeeded(ctx context.Context, actor *User, child *User, groupID int64, input ChildGroupDelegationInput, existing *AgentGroupDelegation) error {
	if actor == nil || child == nil || !input.CanDelegate {
		return nil
	}
	if existing != nil && existing.RateMultiplier >= input.RateMultiplier {
		return nil
	}
	if actor.Role != RoleAdmin && child.Role != RoleEnterprise {
		return nil
	}
	if child.Role != RoleAgentLevel1 && child.Role != RoleEnterprise {
		return nil
	}
	if err := s.repo.RaiseManagedGroupRateFloor(ctx, child.ID, groupID, input.RateMultiplier); err != nil {
		return err
	}
	return s.invalidateManagedGroupRateFloorUsers(ctx, child.ID, child.Role)
}

func (s *AgentManagementService) invalidateManagedGroupRateFloorUsers(ctx context.Context, ownerID int64, ownerRole string) error {
	roles := []string{RoleUser, RoleEnterprise}
	if ownerRole == RoleEnterprise {
		roles = []string{RoleEmployee}
	}
	children, err := s.listAllDirectChildrenForRoles(ctx, ownerID, roles)
	if err != nil {
		return err
	}
	for i := range children {
		s.invalidateUser(ctx, children[i].ID)
		if children[i].Role != RoleEnterprise {
			continue
		}
		employees, err := s.listAllDirectChildrenForRoles(ctx, children[i].ID, []string{RoleEmployee})
		if err != nil {
			return err
		}
		for j := range employees {
			s.invalidateUser(ctx, employees[j].ID)
		}
	}
	return nil
}

func isAgentManagerRole(role string) bool {
	return role == RoleAdmin || role == RoleAgentLevel1
}

func rolesForDirectChildKind(kind DirectChildKind) ([]string, error) {
	switch kind {
	case DirectChildKindUsers:
		return []string{RoleUser}, nil
	case DirectChildKindAgents:
		return []string{RoleAgentLevel1}, nil
	case DirectChildKindEnterprises:
		return []string{RoleEnterprise}, nil
	default:
		return nil, ErrAgentManagementInvalidTarget
	}
}

func isProfileBackedChildRole(role string) bool {
	return (isAgentManagerRole(role) && role != RoleAdmin) || role == RoleEnterprise
}

func (s *AgentManagementService) managerCapacity(ctx context.Context, user *User) (concurrency int, rpm int, err error) {
	if user.Role == RoleAdmin {
		return 0, 0, nil
	}
	profile, err := s.repo.GetAgentProfile(ctx, user.ID)
	if err != nil {
		return 0, 0, err
	}
	if profile == nil {
		return 0, 0, nil
	}
	return profile.PoolConcurrency, profile.PoolRPM, nil
}

func (s *AgentManagementService) upsertProfileBackedChildPool(ctx context.Context, role string, childID int64, poolConcurrency int, poolRPM int) error {
	if role == RoleEnterprise {
		return s.repo.UpsertEnterpriseProfile(ctx, childID, poolConcurrency, poolRPM)
	}
	return s.repo.UpsertAgentProfile(ctx, childID, poolConcurrency, poolRPM)
}

func (s *AgentManagementService) recalculateAgentEffectiveQuota(ctx context.Context, agentID int64) error {
	agent, err := s.userRepo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}
	if !isAgentManagerRole(agent.Role) || agent.Role == RoleAdmin {
		return nil
	}
	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, agent)
	if err != nil {
		return err
	}
	usage, err := s.repo.GetDirectChildQuotaUsage(ctx, agent.ID, nil)
	if err != nil {
		return err
	}
	remainingConcurrency := concurrencyEffectiveRemainingForStorage(totalConcurrency, usage.Concurrency)
	remainingRPM := rpmEffectiveRemainingForStorage(totalRPM, usage.RPM, usage.UnlimitedRPM)
	if err := s.repo.SetEffectiveQuota(ctx, agent.ID, remainingConcurrency, remainingRPM); err != nil {
		return err
	}
	s.invalidateUser(ctx, agent.ID)
	return nil
}

func (s *AgentManagementService) recalculateEnterpriseEffectiveQuota(ctx context.Context, enterpriseID int64) error {
	enterprise, err := s.userRepo.GetByID(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if enterprise.Role != RoleEnterprise {
		return nil
	}
	profile, err := s.repo.GetEnterpriseProfile(ctx, enterprise.ID)
	if err != nil {
		return err
	}
	if profile == nil {
		profile = &EnterpriseProfile{UserID: enterprise.ID}
	}
	usage, err := s.repo.GetEnterpriseEmployeeQuotaUsage(ctx, enterprise.ID, nil)
	if err != nil {
		return err
	}
	remainingConcurrency := concurrencyEffectiveRemainingForStorage(profile.PoolConcurrency, usage.Concurrency)
	remainingRPM := rpmEffectiveRemainingForStorage(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM)
	if err := s.repo.SetEffectiveQuota(ctx, enterprise.ID, remainingConcurrency, remainingRPM); err != nil {
		return err
	}
	s.invalidateUser(ctx, enterprise.ID)
	return nil
}

func buildAllocationSummary(actor *User, totalConcurrency int, totalRPM int, usage QuotaUsageSummary) AllocationSummary {
	unlimitedConcurrency := actor != nil && actor.Role == RoleAdmin
	unlimitedRPM := actor != nil && (actor.Role == RoleAdmin || totalRPM == 0)
	unlimitedCapacity := unlimitedConcurrency && unlimitedRPM
	remainingConcurrency := concurrencyRemaining(totalConcurrency, usage.Concurrency)
	remainingRPM := rpmRemaining(totalRPM, usage.RPM, usage.UnlimitedRPM)
	return AllocationSummary{
		TotalConcurrency:     totalConcurrency,
		AllocatedConcurrency: usage.Concurrency,
		RemainingConcurrency: remainingConcurrency,
		TotalRPM:             totalRPM,
		AllocatedRPM:         usage.RPM,
		RemainingRPM:         remainingRPM,
		UnlimitedCapacity:    unlimitedCapacity,
		UnlimitedConcurrency: unlimitedConcurrency,
		UnlimitedRPM:         unlimitedRPM,
	}
}

func concurrencyEffectiveRemainingForStorage(total int, allocated int) int {
	remaining := concurrencyRemaining(total, allocated)
	if remaining == 0 {
		return -1
	}
	return remaining
}

func rpmEffectiveRemainingForStorage(total int, allocated int, allocatedUnlimited bool) int {
	remaining := rpmRemaining(total, allocated, allocatedUnlimited)
	if total > 0 && remaining == 0 {
		return -1
	}
	return remaining
}

func concurrencyRequestExceedsCapacity(total int, allocated int, requested int) bool {
	if requested < 1 {
		return true
	}
	return requested >= concurrencyRemaining(total, allocated)
}

func rpmRequestExceedsCapacity(total int, allocated int, allocatedUnlimited bool, requested int) bool {
	if requested < 0 {
		return true
	}
	if total == 0 {
		return false
	}
	if allocatedUnlimited {
		return true
	}
	if requested == 0 {
		return true
	}
	return requested >= rpmRemaining(total, allocated, false)
}

func concurrencyRequestBelowAllocated(requested int, allocated int) bool {
	if allocated <= 0 {
		return false
	}
	return requested <= allocated
}

func rpmRequestBelowAllocated(requested int, allocated int, allocatedUnlimited bool) bool {
	if requested == 0 {
		return false
	}
	if allocatedUnlimited {
		return true
	}
	if allocated == 0 {
		return false
	}
	return requested <= allocated
}

func concurrencyRemaining(total int, allocated int) int {
	remaining := total - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

func rpmRemaining(total int, allocated int, allocatedUnlimited bool) int {
	if total == 0 {
		return 0
	}
	if allocatedUnlimited {
		return 0
	}
	remaining := total - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (u QuotaUsageSummary) WithRequest(concurrency int, rpm int) QuotaUsageSummary {
	if !u.UnlimitedConcurrency {
		u.Concurrency += concurrency
	}
	if rpm == 0 {
		u.UnlimitedRPM = true
		u.RPM = 0
	} else if !u.UnlimitedRPM {
		u.RPM += rpm
	}
	return u
}

func (u AllocationUpdate) RequestedConcurrency() int {
	if u.Concurrency != nil {
		return *u.Concurrency
	}
	return u.AllocatedConcurrency
}

func (u AllocationUpdate) RequestedRPM() int {
	if u.RPM != nil {
		return *u.RPM
	}
	return u.AllocatedRPM
}

func canUpgradeDirectUser(actorRole string, targetRole string) bool {
	switch actorRole {
	case RoleAdmin:
		return targetRole == RoleAgentLevel1 || targetRole == RoleEnterprise
	case RoleAgentLevel1:
		return targetRole == RoleEnterprise
	default:
		return false
	}
}
