//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type enterpriseManagementRepoStub struct {
	users  map[int64]*User
	nextID int64

	profiles              map[int64]EnterpriseProfile
	quotaUsage            QuotaUsageSummary
	createEmployeeCalls   []User
	updateAllocationCalls []EmployeeAllocationUpdate
	deleteEmployeeCalls   []int64
	groupDelegations      []agentGroupDelegationRecord
}

func newEnterpriseManagementRepoStub(users ...*User) *enterpriseManagementRepoStub {
	out := &enterpriseManagementRepoStub{
		users:    map[int64]*User{},
		nextID:   1000,
		profiles: map[int64]EnterpriseProfile{},
	}
	for _, user := range users {
		clone := *user
		out.users[user.ID] = &clone
		if user.ID >= out.nextID {
			out.nextID = user.ID + 1
		}
	}
	return out
}

func (r *enterpriseManagementRepoStub) GetEnterpriseProfile(_ context.Context, userID int64) (*EnterpriseProfile, error) {
	profile, ok := r.profiles[userID]
	if !ok {
		return nil, nil
	}
	clone := profile
	return &clone, nil
}

func (r *enterpriseManagementRepoStub) UpsertEnterpriseProfile(_ context.Context, userID int64, poolConcurrency int, poolRPM int) error {
	r.profiles[userID] = EnterpriseProfile{UserID: userID, PoolConcurrency: poolConcurrency, PoolRPM: poolRPM}
	return nil
}

func (r *enterpriseManagementRepoStub) GetEnterpriseEmployeeQuotaUsage(context.Context, int64, *int64) (QuotaUsageSummary, error) {
	return r.quotaUsage, nil
}

func (r *enterpriseManagementRepoStub) RecalculateEnterpriseQuota(context.Context, int64) error {
	return nil
}

func (r *enterpriseManagementRepoStub) CreateEmployee(_ context.Context, enterpriseID int64, _ int64, user *User) error {
	r.nextID++
	user.ID = r.nextID
	user.Role = RoleEmployee
	user.ParentUserID = &enterpriseID
	user.Status = StatusActive
	user.AllocatedConcurrency = user.Concurrency
	user.AllocatedRPM = user.RPMLimit
	clone := *user
	r.users[user.ID] = &clone
	r.createEmployeeCalls = append(r.createEmployeeCalls, clone)
	return nil
}

func (r *enterpriseManagementRepoStub) UpdateEmployeeAllocation(_ context.Context, enterpriseID int64, employeeID int64, _ int64, target EmployeeAllocationUpdate) (*EmployeeAllocationResult, error) {
	user, ok := r.users[employeeID]
	if !ok || user.ParentUserID == nil || *user.ParentUserID != enterpriseID || user.Role != RoleEmployee {
		return nil, ErrEnterpriseManagementNotEmployee
	}
	user.Balance = target.Balance
	user.Concurrency = target.Concurrency
	user.RPMLimit = target.RPM
	user.AllocatedConcurrency = target.Concurrency
	user.AllocatedRPM = target.RPM
	r.updateAllocationCalls = append(r.updateAllocationCalls, target)
	clone := *user
	return &EmployeeAllocationResult{User: &clone, Allocation: AllocationSummary{TotalConcurrency: 0, TotalRPM: 0, UnlimitedCapacity: true, UnlimitedConcurrency: true, UnlimitedRPM: true}}, nil
}

func (r *enterpriseManagementRepoStub) DeleteEmployeeAndReturnAllocation(_ context.Context, _ int64, employeeID int64, _ int64) ([]int64, error) {
	delete(r.users, employeeID)
	r.deleteEmployeeCalls = append(r.deleteEmployeeCalls, employeeID)
	return []int64{employeeID}, nil
}

func (r *enterpriseManagementRepoStub) HardDeleteEnterpriseWithEmployees(context.Context, int64) ([]int64, error) {
	panic("unexpected HardDeleteEnterpriseWithEmployees")
}

func (r *enterpriseManagementRepoStub) CascadeEnterpriseStatus(context.Context, int64, string) ([]int64, error) {
	panic("unexpected CascadeEnterpriseStatus")
}

func (r *enterpriseManagementRepoStub) ListDirectChildren(_ context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return r.ListDirectChildrenWithSearch(context.Background(), parentID, roles, params, "")
}

func (r *enterpriseManagementRepoStub) ListDirectChildrenWithSearch(_ context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]User, *pagination.PaginationResult, error) {
	roleSet := map[string]struct{}{}
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}
	search = strings.ToLower(strings.TrimSpace(search))
	var out []User
	for _, user := range r.users {
		if user.ParentUserID == nil || *user.ParentUserID != parentID {
			continue
		}
		if _, ok := roleSet[user.Role]; !ok {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(user.Email), search) && !strings.Contains(strings.ToLower(user.Username), search) {
			continue
		}
		out = append(out, *user)
	}
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
}

func (r *enterpriseManagementRepoStub) ListGroupDelegationsForChild(_ context.Context, childID int64) ([]AgentGroupDelegation, error) {
	var out []AgentGroupDelegation
	for _, delegation := range r.groupDelegations {
		if delegation.childID != childID {
			continue
		}
		out = append(out, AgentGroupDelegation{
			ManagerUserID:  delegation.managerID,
			ChildUserID:    delegation.childID,
			GroupID:        delegation.groupID,
			RateMultiplier: delegation.rateMultiplier,
			CanDelegate:    delegation.canDelegate,
		})
	}
	return out, nil
}

func (r *enterpriseManagementRepoStub) GetGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64) (*AgentGroupDelegation, error) {
	for _, delegation := range r.groupDelegations {
		if delegation.managerID == managerID && delegation.childID == childID && delegation.groupID == groupID {
			return &AgentGroupDelegation{
				ManagerUserID:  managerID,
				ChildUserID:    childID,
				GroupID:        groupID,
				RateMultiplier: delegation.rateMultiplier,
				CanDelegate:    delegation.canDelegate,
			}, nil
		}
	}
	return nil, nil
}

func (r *enterpriseManagementRepoStub) UpsertGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error {
	for i := range r.groupDelegations {
		if r.groupDelegations[i].managerID == managerID && r.groupDelegations[i].childID == childID && r.groupDelegations[i].groupID == groupID {
			r.groupDelegations[i].rateMultiplier = rateMultiplier
			r.groupDelegations[i].canDelegate = canDelegate
			return nil
		}
	}
	r.groupDelegations = append(r.groupDelegations, agentGroupDelegationRecord{
		managerID:      managerID,
		childID:        childID,
		groupID:        groupID,
		rateMultiplier: rateMultiplier,
		canDelegate:    canDelegate,
	})
	return nil
}

func (r *enterpriseManagementRepoStub) DeleteGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64) error {
	filtered := r.groupDelegations[:0]
	for _, delegation := range r.groupDelegations {
		if delegation.managerID == managerID && delegation.childID == childID && delegation.groupID == groupID {
			continue
		}
		filtered = append(filtered, delegation)
	}
	r.groupDelegations = filtered
	return nil
}

func TestEnterpriseManagementCreateEmployeeRequiresEnterpriseActor(t *testing.T) {
	actorID := int64(1)
	repo := newEnterpriseManagementRepoStub(&User{ID: actorID, Role: RoleUser, Status: StatusActive})
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewEnterpriseManagementService(repo, userRepo, nil, nil)

	_, err := svc.CreateEmployee(context.Background(), actorID, EmployeeCreateInput{
		Email:       "employee@test.local",
		Password:    "password-123",
		Balance:     1,
		Concurrency: 1,
		RPM:         1,
	})

	require.ErrorIs(t, err, ErrEnterpriseManagementForbidden)
	require.Empty(t, repo.createEmployeeCalls)
}

func TestEnterpriseManagementCreateEmployeeRejectsBalanceOverEnterpriseBalance(t *testing.T) {
	enterpriseID := int64(1)
	repo := newEnterpriseManagementRepoStub(&User{ID: enterpriseID, Role: RoleEnterprise, Balance: 5, Status: StatusActive})
	repo.profiles[enterpriseID] = EnterpriseProfile{UserID: enterpriseID, PoolConcurrency: 10, PoolRPM: 100}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewEnterpriseManagementService(repo, userRepo, nil, nil)

	_, err := svc.CreateEmployee(context.Background(), enterpriseID, EmployeeCreateInput{
		Email:       "employee@test.local",
		Password:    "password-123",
		Balance:     10,
		Concurrency: 1,
		RPM:         10,
	})

	require.ErrorIs(t, err, ErrEnterpriseManagementBalanceExceeded)
	require.Empty(t, repo.createEmployeeCalls)
}

func TestEnterpriseManagementCreateEmployeeSetsEmployeeRoleAndParent(t *testing.T) {
	enterpriseID := int64(1)
	repo := newEnterpriseManagementRepoStub(&User{ID: enterpriseID, Role: RoleEnterprise, Balance: 100, Status: StatusActive})
	repo.profiles[enterpriseID] = EnterpriseProfile{UserID: enterpriseID, PoolConcurrency: 10, PoolRPM: 100}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewEnterpriseManagementService(repo, userRepo, nil, &agentManagementAuthInvalidatorStub{})

	employee, err := svc.CreateEmployee(context.Background(), enterpriseID, EmployeeCreateInput{
		Email:       "employee@test.local",
		Password:    "password-123",
		Username:    "employee",
		Balance:     10,
		Concurrency: 2,
		RPM:         20,
	})

	require.NoError(t, err)
	require.Equal(t, RoleEmployee, employee.Role)
	require.NotNil(t, employee.ParentUserID)
	require.Equal(t, enterpriseID, *employee.ParentUserID)
	require.Equal(t, 2, employee.Concurrency)
	require.Equal(t, 20, employee.RPMLimit)
	require.Len(t, repo.createEmployeeCalls, 1)
}

func TestEnterpriseManagementUpdateEmployeeAllocationTreatsZeroQuotaAsUnlimited(t *testing.T) {
	enterpriseID := int64(1)
	employeeID := int64(2)
	repo := newEnterpriseManagementRepoStub(
		&User{ID: enterpriseID, Role: RoleEnterprise, Balance: 100, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Balance: 10, Status: StatusActive},
	)
	repo.profiles[enterpriseID] = EnterpriseProfile{UserID: enterpriseID, PoolConcurrency: 0, PoolRPM: 0}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewEnterpriseManagementService(repo, userRepo, nil, nil)

	summary, err := svc.UpdateEmployeeAllocation(context.Background(), enterpriseID, employeeID, EmployeeAllocationUpdate{
		Balance:     10,
		Concurrency: 0,
		RPM:         0,
	})

	require.NoError(t, err)
	require.True(t, summary.UnlimitedConcurrency)
	require.True(t, summary.UnlimitedRPM)
	require.Len(t, repo.updateAllocationCalls, 1)
}

func TestEnterpriseManagementUpdateEmployeeAllocationRejectsFinitePoolOverAllocation(t *testing.T) {
	enterpriseID := int64(1)
	employeeID := int64(2)
	repo := newEnterpriseManagementRepoStub(
		&User{ID: enterpriseID, Role: RoleEnterprise, Balance: 100, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Balance: 10, Status: StatusActive},
	)
	repo.profiles[enterpriseID] = EnterpriseProfile{UserID: enterpriseID, PoolConcurrency: 10, PoolRPM: 100}
	repo.quotaUsage = QuotaUsageSummary{Concurrency: 8, RPM: 80}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewEnterpriseManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateEmployeeAllocation(context.Background(), enterpriseID, employeeID, EmployeeAllocationUpdate{
		Balance:     10,
		Concurrency: 3,
		RPM:         30,
	})

	require.ErrorIs(t, err, ErrEnterpriseManagementAllocationExceeded)
	require.Empty(t, repo.updateAllocationCalls)
}

func TestEffectiveAPIUsageCapacityTreatsNegativeEnterpriseRemainingAsZero(t *testing.T) {
	concurrency, rpm := EffectiveAPIUsageCapacity(&User{
		ID:                   1,
		Role:                 RoleEnterprise,
		Concurrency:          -1,
		RPMLimit:             -1,
		AllocatedConcurrency: 10,
		AllocatedRPM:         100,
	})

	require.Equal(t, 0, concurrency)
	require.Equal(t, 0, rpm)
}

func TestEnterpriseManagementSetEmployeeGroupUsesEnterpriseRateWithoutDelegation(t *testing.T) {
	enterpriseID := int64(1)
	employeeID := int64(2)
	groupID := int64(100)
	upstreamID := int64(9)
	repo := newEnterpriseManagementRepoStub(
		&User{ID: enterpriseID, Role: RoleEnterprise, ParentUserID: &upstreamID, Balance: 100, Status: StatusActive, AllowedGroups: []int64{groupID}},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Balance: 10, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{{
		managerID:      upstreamID,
		childID:        enterpriseID,
		groupID:        groupID,
		rateMultiplier: 1.8,
		canDelegate:    true,
	}}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", Status: StatusActive, IsExclusive: true, RateMultiplier: 2})
	svc := NewEnterpriseManagementService(repo, userRepo, groupRepo, nil)

	err := svc.SetEmployeeGroup(context.Background(), enterpriseID, employeeID, groupID, true)

	require.NoError(t, err)
	require.Len(t, repo.groupDelegations, 2)
	delegation, err := repo.GetGroupDelegation(context.Background(), enterpriseID, employeeID, groupID)
	require.NoError(t, err)
	require.NotNil(t, delegation)
	require.Equal(t, 1.8, delegation.RateMultiplier)
	require.False(t, delegation.CanDelegate)
	require.Equal(t, []int64{groupID}, repo.users[employeeID].AllowedGroups)
}
