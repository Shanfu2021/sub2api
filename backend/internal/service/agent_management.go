package service

import (
	"context"
	"fmt"
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrAgentManagementForbidden           = infraerrors.Forbidden("AGENT_MANAGEMENT_FORBIDDEN", "agent management operation is not allowed")
	ErrAgentManagementNotDirectChild      = infraerrors.Forbidden("AGENT_MANAGEMENT_NOT_DIRECT_CHILD", "target user is not a direct child")
	ErrAgentManagementAllocationExceeded  = infraerrors.BadRequest("AGENT_MANAGEMENT_ALLOCATION_EXCEEDED", "allocation exceeds remaining capacity")
	ErrAgentManagementUnsupportedRole     = infraerrors.BadRequest("AGENT_MANAGEMENT_UNSUPPORTED_ROLE", "unsupported agent management role")
	ErrAgentManagementInvalidTarget       = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_TARGET", "invalid target user")
	ErrAgentManagementInvalidAllocation   = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_ALLOCATION", "allocation must be non-negative")
	ErrAgentManagementInvalidGroupRate    = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_GROUP_RATE", "group delegation rate multiplier must be positive")
	ErrAgentManagementInvalidGroup        = infraerrors.BadRequest("AGENT_MANAGEMENT_INVALID_GROUP", "only active exclusive groups can be delegated")
	ErrAgentManagementRootAdminNotPresent = infraerrors.NotFound("AGENT_MANAGEMENT_ROOT_ADMIN_NOT_PRESENT", "root admin not found")
	ErrAgentManagementNotImplemented      = infraerrors.New(http.StatusNotImplemented, "AGENT_MANAGEMENT_NOT_IMPLEMENTED", "agent management feature is not implemented yet")
)

type AllocationUpdate struct {
	AllocatedConcurrency int  `json:"allocated_concurrency"`
	AllocatedRPM         int  `json:"allocated_rpm"`
	Concurrency          *int `json:"concurrency,omitempty"`
	RPM                  *int `json:"rpm,omitempty"`
}

type AgentProfile struct {
	UserID          int64 `json:"user_id"`
	PoolConcurrency int   `json:"pool_concurrency"`
	PoolRPM         int   `json:"pool_rpm"`
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

type AllocationSummary struct {
	TotalConcurrency     int  `json:"total_concurrency"`
	AllocatedConcurrency int  `json:"allocated_concurrency"`
	RemainingConcurrency int  `json:"remaining_concurrency"`
	TotalRPM             int  `json:"total_rpm"`
	AllocatedRPM         int  `json:"allocated_rpm"`
	RemainingRPM         int  `json:"remaining_rpm"`
	UnlimitedCapacity    bool `json:"unlimited_capacity"`
}

type DirectChildrenResult struct {
	Users      []User                       `json:"users"`
	Pagination *pagination.PaginationResult `json:"pagination"`
}

type AgentManagementSummary struct {
	Allocation AllocationSummary `json:"allocation"`
}

type AgentGroupRate struct {
	Group         Group   `json:"group"`
	EffectiveRate float64 `json:"effective_rate"`
	CanDelegate   bool    `json:"can_delegate"`
	Source        string  `json:"source"`
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

type AgentManagementRepository interface {
	GetRootAdmin(ctx context.Context) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	GetAgentProfile(ctx context.Context, userID int64) (*AgentProfile, error)
	UpsertAgentProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
	SumDirectChildQuotaUsage(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	SetEffectiveQuota(ctx context.Context, userID int64, concurrency int, rpm int) error
	SetParent(ctx context.Context, userID int64, parentID *int64) error
	SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error
	SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error
	DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error
	ListGroupDelegationsForChild(ctx context.Context, childID int64) ([]AgentGroupDelegation, error)
	GetGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) (*AgentGroupDelegation, error)
	UpsertGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error
	DeleteGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) error
}

type AgentManagementService struct {
	repo                 AgentManagementRepository
	userRepo             UserRepository
	groupRepo            GroupRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewAgentManagementService(repo AgentManagementRepository, userRepo UserRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *AgentManagementService {
	return &AgentManagementService{
		repo:                 repo,
		userRepo:             userRepo,
		groupRepo:            groupRepo,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *AgentManagementService) ListDirectUsers(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	return s.listDirectChildren(ctx, actorID, []string{RoleUser}, pagination.DefaultPagination())
}

func (s *AgentManagementService) ListDirectAgents(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleAgentLevel2 {
		return &DirectChildrenResult{Users: []User{}, Pagination: &pagination.PaginationResult{Page: 1, PageSize: pagination.DefaultPagination().Limit()}}, nil
	}
	return s.listDirectChildrenForActor(ctx, actor, []string{RoleAgentLevel1, RoleAgentLevel2}, pagination.DefaultPagination())
}

func (s *AgentManagementService) ListDirectEnterprises(ctx context.Context, actorID int64) (*DirectChildrenResult, error) {
	return s.listDirectChildren(ctx, actorID, []string{RoleEnterprise}, pagination.DefaultPagination())
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
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildQuotaUsage(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	return &AgentManagementSummary{
		Allocation: buildAllocationSummary(actor, totalConcurrency, totalRPM, allocatedConcurrency, allocatedRPM),
	}, nil
}

func (s *AgentManagementService) CreateDirectUser(ctx context.Context, actorID int64, input CreateDirectUserInput) (*User, error) {
	if input.AllocatedConcurrency < 0 || input.AllocatedRPM < 0 {
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
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildQuotaUsage(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (input.AllocatedConcurrency > totalConcurrency-allocatedConcurrency || input.AllocatedRPM > totalRPM-allocatedRPM) {
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
	if requestedConcurrency < 0 || requestedRPM < 0 {
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

	totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
	if err != nil {
		return nil, err
	}
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildQuotaUsage(ctx, actor.ID, &child.ID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (requestedConcurrency > totalConcurrency-allocatedConcurrency || requestedRPM > totalRPM-allocatedRPM) {
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

	allocatedConcurrency += requestedConcurrency
	allocatedRPM += requestedRPM
	summary := buildAllocationSummary(actor, totalConcurrency, totalRPM, allocatedConcurrency, allocatedRPM)
	return &summary, nil
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
	if isAgentManagerRole(targetRole) && targetRole != RoleAdmin {
		if input.PoolConcurrency < 0 || input.PoolRPM < 0 {
			return nil, ErrAgentManagementInvalidAllocation
		}
		totalConcurrency, totalRPM, err := s.managerCapacity(ctx, actor)
		if err != nil {
			return nil, err
		}
		allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildQuotaUsage(ctx, actor.ID, &child.ID)
		if err != nil {
			return nil, err
		}
		if actor.Role != RoleAdmin && (input.PoolConcurrency > totalConcurrency-allocatedConcurrency || input.PoolRPM > totalRPM-allocatedRPM) {
			return nil, ErrAgentManagementAllocationExceeded
		}
		if err := s.repo.UpsertAgentProfile(ctx, child.ID, input.PoolConcurrency, input.PoolRPM); err != nil {
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
		if err := s.repo.DeleteLevel1AgentAndMoveChildren(ctx, child.ID, rootAdmin.ID); err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		return nil
	}
	if actor.Role == RoleAgentLevel1 && child.Role == RoleAgentLevel2 {
		if err := s.repo.SetRoleAndParent(ctx, child.ID, RoleAgentLevel1, &rootAdmin.ID); err != nil {
			return err
		}
		s.invalidateUser(ctx, child.ID)
		return nil
	}
	if child.Role == RoleUser || child.Role == RoleEnterprise {
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
	if err := s.requireGroupDelegationAccess(ctx, actor, groupID); err != nil {
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
	s.invalidateUser(ctx, child.ID)
	return nil
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
	if s.userRepo != nil {
		if err := s.userRepo.RemoveGroupFromUserAllowedGroups(ctx, child.ID, groupID); err != nil {
			return err
		}
	}
	s.invalidateUser(ctx, child.ID)
	return nil
}

func (s *AgentManagementService) ResolveInvitationParent(ctx context.Context, inviterID int64) (*int64, error) {
	if inviterID <= 0 {
		return nil, ErrAgentManagementInvalidTarget
	}
	inviter, err := s.userRepo.GetByID(ctx, inviterID)
	if err != nil {
		return nil, err
	}
	if inviter.Role == RoleAdmin || isAgentManagerRole(inviter.Role) {
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
			return &parent.ID, nil
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
	actor, err := s.requireManager(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return s.listDirectChildrenForActor(ctx, actor, roles, params)
}

func (s *AgentManagementService) listDirectChildrenForActor(ctx context.Context, actor *User, roles []string, params pagination.PaginationParams) (*DirectChildrenResult, error) {
	users, page, err := s.repo.ListDirectChildren(ctx, actor.ID, roles, params)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if !isAgentManagerRole(users[i].Role) || users[i].Role == RoleAdmin {
			continue
		}
		profile, err := s.repo.GetAgentProfile(ctx, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].AgentProfile = profile
	}
	return &DirectChildrenResult{Users: users, Pagination: page}, nil
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

func (s *AgentManagementService) requireGroupDelegationAccess(ctx context.Context, actor *User, groupID int64) error {
	if actor.Role == RoleAdmin {
		return nil
	}
	if actor.ParentUserID == nil {
		return ErrAgentManagementForbidden
	}
	delegation, err := s.repo.GetGroupDelegation(ctx, *actor.ParentUserID, actor.ID, groupID)
	if err != nil {
		return err
	}
	if delegation == nil || !delegation.CanDelegate {
		return ErrAgentManagementForbidden
	}
	return nil
}

func (s *AgentManagementService) invalidateUser(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func isAgentManagerRole(role string) bool {
	return role == RoleAdmin || role == RoleAgentLevel1 || role == RoleAgentLevel2
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
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildQuotaUsage(ctx, agent.ID, nil)
	if err != nil {
		return err
	}
	remainingConcurrency := totalConcurrency - allocatedConcurrency
	remainingRPM := totalRPM - allocatedRPM
	if remainingConcurrency < 0 {
		remainingConcurrency = 0
	}
	if remainingRPM < 0 {
		remainingRPM = 0
	}
	if err := s.repo.SetEffectiveQuota(ctx, agent.ID, remainingConcurrency, remainingRPM); err != nil {
		return err
	}
	s.invalidateUser(ctx, agent.ID)
	return nil
}

func buildAllocationSummary(actor *User, totalConcurrency int, totalRPM int, allocatedConcurrency int, allocatedRPM int) AllocationSummary {
	unlimitedCapacity := actor != nil && actor.Role == RoleAdmin
	remainingConcurrency := totalConcurrency - allocatedConcurrency
	remainingRPM := totalRPM - allocatedRPM
	if unlimitedCapacity {
		remainingConcurrency = 0
		remainingRPM = 0
	}
	return AllocationSummary{
		TotalConcurrency:     totalConcurrency,
		AllocatedConcurrency: allocatedConcurrency,
		RemainingConcurrency: remainingConcurrency,
		TotalRPM:             totalRPM,
		AllocatedRPM:         allocatedRPM,
		RemainingRPM:         remainingRPM,
		UnlimitedCapacity:    unlimitedCapacity,
	}
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
		return targetRole == RoleAgentLevel2 || targetRole == RoleEnterprise
	case RoleAgentLevel2:
		return targetRole == RoleEnterprise
	default:
		return false
	}
}
