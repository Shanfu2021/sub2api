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
	ErrAgentManagementRootAdminNotPresent = infraerrors.NotFound("AGENT_MANAGEMENT_ROOT_ADMIN_NOT_PRESENT", "root admin not found")
	ErrAgentManagementNotImplemented      = infraerrors.New(http.StatusNotImplemented, "AGENT_MANAGEMENT_NOT_IMPLEMENTED", "agent management feature is not implemented yet")
)

type AllocationUpdate struct {
	AllocatedConcurrency int `json:"allocated_concurrency"`
	AllocatedRPM         int `json:"allocated_rpm"`
}

type AllocationSummary struct {
	TotalConcurrency     int `json:"total_concurrency"`
	AllocatedConcurrency int `json:"allocated_concurrency"`
	RemainingConcurrency int `json:"remaining_concurrency"`
	TotalRPM             int `json:"total_rpm"`
	AllocatedRPM         int `json:"allocated_rpm"`
	RemainingRPM         int `json:"remaining_rpm"`
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

type ChildGroupDelegationInput struct {
	RateMultiplier float64 `json:"rate_multiplier"`
	CanDelegate    bool    `json:"can_delegate"`
}

type AgentManagementRepository interface {
	GetRootAdmin(ctx context.Context) (*User, error)
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	SetParent(ctx context.Context, userID int64, parentID *int64) error
	SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error
	SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error
	DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error
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
	totalConcurrency, totalRPM := managerCapacity(actor)
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildAllocations(ctx, actor.ID, nil)
	if err != nil {
		return nil, err
	}
	return &AgentManagementSummary{
		Allocation: AllocationSummary{
			TotalConcurrency:     totalConcurrency,
			AllocatedConcurrency: allocatedConcurrency,
			RemainingConcurrency: totalConcurrency - allocatedConcurrency,
			TotalRPM:             totalRPM,
			AllocatedRPM:         allocatedRPM,
			RemainingRPM:         totalRPM - allocatedRPM,
		},
	}, nil
}

func (s *AgentManagementService) UpdateAllocation(ctx context.Context, actorID int64, childID int64, req AllocationUpdate) (*AllocationSummary, error) {
	if req.AllocatedConcurrency < 0 || req.AllocatedRPM < 0 {
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

	totalConcurrency, totalRPM := managerCapacity(actor)
	allocatedConcurrency, allocatedRPM, err := s.repo.SumDirectChildAllocations(ctx, actor.ID, &child.ID)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && (req.AllocatedConcurrency > totalConcurrency-allocatedConcurrency || req.AllocatedRPM > totalRPM-allocatedRPM) {
		return nil, ErrAgentManagementAllocationExceeded
	}
	if err := s.repo.SetAllocation(ctx, child.ID, req.AllocatedConcurrency, req.AllocatedRPM); err != nil {
		return nil, err
	}
	s.invalidateUser(ctx, child.ID)

	allocatedConcurrency += req.AllocatedConcurrency
	allocatedRPM += req.AllocatedRPM
	return &AllocationSummary{
		TotalConcurrency:     totalConcurrency,
		AllocatedConcurrency: allocatedConcurrency,
		RemainingConcurrency: totalConcurrency - allocatedConcurrency,
		TotalRPM:             totalRPM,
		AllocatedRPM:         allocatedRPM,
		RemainingRPM:         totalRPM - allocatedRPM,
	}, nil
}

func (s *AgentManagementService) UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, targetRole string) (*User, error) {
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
	if err := s.repo.SetRoleAndParent(ctx, child.ID, targetRole, child.ParentUserID); err != nil {
		return nil, err
	}
	child.Role = targetRole
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
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return nil, err
	}
	if s.groupRepo == nil {
		return []AgentGroupRate{}, nil
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AgentGroupRate, 0, len(groups))
	for i := range groups {
		if groups[i].IsExclusive {
			continue
		}
		out = append(out, AgentGroupRate{
			Group:         groups[i],
			EffectiveRate: groups[i].RateMultiplier,
			CanDelegate:   false,
			Source:        "public",
		})
	}
	return out, nil
}

func (s *AgentManagementService) SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, input ChildGroupDelegationInput) error {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return err
	}
	return ErrAgentManagementNotImplemented
}

func (s *AgentManagementService) RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error {
	if _, err := s.requireManager(ctx, actorID); err != nil {
		return err
	}
	return ErrAgentManagementNotImplemented
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

func (s *AgentManagementService) invalidateUser(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func isAgentManagerRole(role string) bool {
	return role == RoleAdmin || role == RoleAgentLevel1 || role == RoleAgentLevel2
}

func managerCapacity(user *User) (concurrency int, rpm int) {
	if user.Role == RoleAdmin {
		return user.Concurrency, user.RPMLimit
	}
	return user.AllocatedConcurrency, user.AllocatedRPM
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
