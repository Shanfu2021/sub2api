package service

import "context"

type agentIncomeUserRepository interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

// AgentIncomeResolver captures first-level agent income at usage time.
type AgentIncomeResolver struct {
	userRepo          agentIncomeUserRepository
	userGroupRateRepo AgentIncomeRateRepository
}

type AgentIncomeSnapshot struct {
	AgentOwnerUserID        *int64
	UserRateMultiplier      float64
	AgentCostRateMultiplier float64
	AgentIncome             float64
}

func NewAgentIncomeResolver(userRepo agentIncomeUserRepository, userGroupRateRepo AgentIncomeRateRepository) *AgentIncomeResolver {
	return &AgentIncomeResolver{userRepo: userRepo, userGroupRateRepo: userGroupRateRepo}
}

func (r *AgentIncomeResolver) Resolve(ctx context.Context, user *User, groupID *int64, actualCost float64, userRate float64, groupDefaultRate float64) AgentIncomeSnapshot {
	snapshot := AgentIncomeSnapshot{UserRateMultiplier: userRate}
	if r == nil || user == nil || groupID == nil || *groupID <= 0 {
		return snapshot
	}

	agentID := r.resolveFirstLevelAgentID(ctx, user)
	if agentID == nil {
		return snapshot
	}

	agentRate := groupDefaultRate
	if r.userGroupRateRepo != nil {
		if rate, err := r.userGroupRateRepo.GetByUserAndGroup(ctx, *agentID, *groupID); err == nil && rate != nil {
			agentRate = *rate
		}
	}

	snapshot.AgentOwnerUserID = agentID
	snapshot.AgentCostRateMultiplier = agentRate
	if actualCost <= 0 || userRate <= 0 {
		return snapshot
	}
	snapshot.AgentIncome = actualCost * (userRate - agentRate) / userRate
	return snapshot
}

func (r *AgentIncomeResolver) resolveFirstLevelAgentID(ctx context.Context, user *User) *int64 {
	if user == nil || user.ParentUserID == nil {
		return nil
	}
	switch user.Role {
	case RoleUser, RoleEnterprise, "":
		return r.parentIfLevel1Agent(ctx, *user.ParentUserID)
	case RoleEmployee:
		if r.userRepo == nil {
			return nil
		}
		parent, err := r.userRepo.GetByID(ctx, *user.ParentUserID)
		if err != nil || parent == nil || parent.Role != RoleEnterprise || parent.ParentUserID == nil {
			return nil
		}
		return r.parentIfLevel1Agent(ctx, *parent.ParentUserID)
	default:
		return nil
	}
}

func (r *AgentIncomeResolver) parentIfLevel1Agent(ctx context.Context, parentID int64) *int64 {
	if parentID <= 0 || r.userRepo == nil {
		return nil
	}
	parent, err := r.userRepo.GetByID(ctx, parentID)
	if err != nil || parent == nil || parent.Role != RoleAgentLevel1 {
		return nil
	}
	id := parent.ID
	return &id
}
