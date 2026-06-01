//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type agentManagementRepoStub struct {
	users map[int64]*User

	setAllocations []AllocationUpdate
	setParents     []struct {
		userID   int64
		parentID *int64
	}
	setRoleAndParents []struct {
		userID   int64
		role     string
		parentID *int64
	}
	deleteLevel1Calls []struct {
		agentID     int64
		rootAdminID int64
	}
}

func newAgentManagementRepoStub(users ...*User) *agentManagementRepoStub {
	out := &agentManagementRepoStub{users: map[int64]*User{}}
	for _, user := range users {
		clone := *user
		out.users[user.ID] = &clone
	}
	return out
}

func (r *agentManagementRepoStub) GetRootAdmin(context.Context) (*User, error) {
	for _, user := range r.users {
		if user.Role == RoleAdmin {
			clone := *user
			return &clone, nil
		}
	}
	return nil, ErrUserNotFound
}

func (r *agentManagementRepoStub) ListDirectChildren(_ context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	roleSet := map[string]struct{}{}
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}
	var out []User
	for _, user := range r.users {
		if user.ParentUserID == nil || *user.ParentUserID != parentID {
			continue
		}
		if len(roleSet) > 0 {
			if _, ok := roleSet[user.Role]; !ok {
				continue
			}
		}
		out = append(out, *user)
	}
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
}

func (r *agentManagementRepoStub) SumDirectChildAllocations(_ context.Context, parentID int64, excludeChildID *int64) (int, int, error) {
	concurrency := 0
	rpm := 0
	for _, user := range r.users {
		if user.ParentUserID == nil || *user.ParentUserID != parentID {
			continue
		}
		if excludeChildID != nil && user.ID == *excludeChildID {
			continue
		}
		concurrency += user.AllocatedConcurrency
		rpm += user.AllocatedRPM
	}
	return concurrency, rpm, nil
}

func (r *agentManagementRepoStub) SetParent(_ context.Context, userID int64, parentID *int64) error {
	user, ok := r.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	r.setParents = append(r.setParents, struct {
		userID   int64
		parentID *int64
	}{userID: userID, parentID: parentID})
	user.ParentUserID = parentID
	return nil
}

func (r *agentManagementRepoStub) SetRoleAndParent(_ context.Context, userID int64, role string, parentID *int64) error {
	user, ok := r.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	r.setRoleAndParents = append(r.setRoleAndParents, struct {
		userID   int64
		role     string
		parentID *int64
	}{userID: userID, role: role, parentID: parentID})
	user.Role = role
	user.ParentUserID = parentID
	return nil
}

func (r *agentManagementRepoStub) SetAllocation(_ context.Context, userID int64, concurrency int, rpm int) error {
	user, ok := r.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	r.setAllocations = append(r.setAllocations, AllocationUpdate{AllocatedConcurrency: concurrency, AllocatedRPM: rpm})
	user.AllocatedConcurrency = concurrency
	user.AllocatedRPM = rpm
	user.Concurrency = concurrency
	user.RPMLimit = rpm
	return nil
}

func (r *agentManagementRepoStub) DeleteLevel1AgentAndMoveChildren(_ context.Context, agentID int64, rootAdminID int64) error {
	r.deleteLevel1Calls = append(r.deleteLevel1Calls, struct {
		agentID     int64
		rootAdminID int64
	}{agentID: agentID, rootAdminID: rootAdminID})
	delete(r.users, agentID)
	return nil
}

type agentManagementUserRepoStub struct {
	*mockUserRepo
	users map[int64]*User
}

func (r *agentManagementUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

type agentManagementAuthInvalidatorStub struct {
	userIDs []int64
}

func (s *agentManagementAuthInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {}
func (s *agentManagementAuthInvalidatorStub) InvalidateAuthCacheByGroupID(context.Context, int64) {
}
func (s *agentManagementAuthInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func TestAgentManagementUpgradeRules(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	userUnderAdminID := int64(10)
	anotherUserUnderAdminID := int64(13)
	userUnderLevel1ID := int64(11)
	userUnderLevel2ID := int64(12)
	users := []*User{
		{ID: rootID, Role: RoleAdmin, AllocatedConcurrency: 1000, AllocatedRPM: 10000},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllocatedConcurrency: 100, AllocatedRPM: 1000},
		{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, AllocatedConcurrency: 50, AllocatedRPM: 500},
		{ID: userUnderAdminID, Role: RoleUser, ParentUserID: &rootID},
		{ID: anotherUserUnderAdminID, Role: RoleUser, ParentUserID: &rootID},
		{ID: userUnderLevel1ID, Role: RoleUser, ParentUserID: &level1ID},
		{ID: userUnderLevel2ID, Role: RoleUser, ParentUserID: &level2ID},
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	got, err := svc.UpgradeDirectUser(context.Background(), rootID, userUnderAdminID, RoleAgentLevel1)
	require.NoError(t, err)
	require.Equal(t, RoleAgentLevel1, got.Role)
	require.Equal(t, []int64{userUnderAdminID}, invalidator.userIDs)

	_, err = svc.UpgradeDirectUser(context.Background(), rootID, anotherUserUnderAdminID, RoleAgentLevel2)
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	got, err = svc.UpgradeDirectUser(context.Background(), level1ID, userUnderLevel1ID, RoleAgentLevel2)
	require.NoError(t, err)
	require.Equal(t, RoleAgentLevel2, got.Role)

	_, err = svc.UpgradeDirectUser(context.Background(), level2ID, userUnderLevel2ID, RoleAgentLevel2)
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	got, err = svc.UpgradeDirectUser(context.Background(), level2ID, userUnderLevel2ID, RoleEnterprise)
	require.NoError(t, err)
	require.Equal(t, RoleEnterprise, got.Role)
}

func TestAgentManagementAllocationCannotExceedRemaining(t *testing.T) {
	managerID := int64(2)
	childID := int64(10)
	otherChildID := int64(11)
	users := []*User{
		{ID: managerID, Role: RoleAgentLevel1, AllocatedConcurrency: 100, AllocatedRPM: 1000},
		{ID: childID, Role: RoleUser, ParentUserID: &managerID, AllocatedConcurrency: 10, AllocatedRPM: 100},
		{ID: otherChildID, Role: RoleUser, ParentUserID: &managerID, AllocatedConcurrency: 60, AllocatedRPM: 600},
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, &agentManagementAuthInvalidatorStub{})

	summary, err := svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 40, AllocatedRPM: 400})
	require.NoError(t, err)
	require.Equal(t, 100, summary.TotalConcurrency)
	require.Equal(t, 100, summary.AllocatedConcurrency)
	require.Equal(t, 0, summary.RemainingConcurrency)
	require.Equal(t, 40, repo.users[childID].AllocatedConcurrency)

	_, err = svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 41, AllocatedRPM: 401})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementAdminAllocationIsUnconstrained(t *testing.T) {
	adminID := int64(1)
	childID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: adminID, Role: RoleAdmin, Concurrency: 5, RPMLimit: 50},
		&User{ID: childID, Role: RoleUser, ParentUserID: &adminID},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	summary, err := svc.UpdateAllocation(context.Background(), adminID, childID, AllocationUpdate{AllocatedConcurrency: 500, AllocatedRPM: 5000})
	require.NoError(t, err)
	require.Equal(t, 500, repo.users[childID].AllocatedConcurrency)
	require.Equal(t, 5000, repo.users[childID].AllocatedRPM)
	require.Equal(t, 500, summary.AllocatedConcurrency)
	require.Equal(t, 5000, summary.AllocatedRPM)
}

func TestAgentManagementDeleteRules(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	userID := int64(10)
	level1UnderAdminID := int64(20)
	users := []*User{
		{ID: rootID, Role: RoleAdmin},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID},
		{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID},
		{ID: userID, Role: RoleUser, ParentUserID: &level1ID},
		{ID: level1UnderAdminID, Role: RoleAgentLevel1, ParentUserID: &rootID},
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), level1ID, userID))
	require.Equal(t, rootID, *repo.users[userID].ParentUserID)
	require.Equal(t, RoleUser, repo.users[userID].Role)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), level1ID, level2ID))
	require.Equal(t, rootID, *repo.users[level2ID].ParentUserID)
	require.Equal(t, RoleAgentLevel1, repo.users[level2ID].Role)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), rootID, level1UnderAdminID))
	require.Len(t, repo.deleteLevel1Calls, 1)
	require.Equal(t, level1UnderAdminID, repo.deleteLevel1Calls[0].agentID)
	require.Equal(t, rootID, repo.deleteLevel1Calls[0].rootAdminID)
	require.Contains(t, invalidator.userIDs, userID)
	require.Contains(t, invalidator.userIDs, level2ID)
	require.Contains(t, invalidator.userIDs, level1UnderAdminID)
}

func TestAgentManagementRejectsNonDirectChild(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	userID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID},
		&User{ID: userID, Role: RoleUser, ParentUserID: &rootID},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpgradeDirectUser(context.Background(), level1ID, userID, RoleEnterprise)
	require.True(t, errors.Is(err, ErrAgentManagementNotDirectChild))
}

func TestAgentManagementResolveInvitationParent(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	ordinaryUnderLevel2ID := int64(4)
	ordinaryUnderAdminID := int64(5)
	users := []*User{
		{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, Status: StatusActive},
		{ID: ordinaryUnderLevel2ID, Role: RoleUser, ParentUserID: &level2ID, Status: StatusActive},
		{ID: ordinaryUnderAdminID, Role: RoleUser, ParentUserID: &rootID, Status: StatusActive},
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	got, err := svc.ResolveInvitationParent(context.Background(), level1ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, level1ID, *got)

	got, err = svc.ResolveInvitationParent(context.Background(), ordinaryUnderLevel2ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, level2ID, *got)

	got, err = svc.ResolveInvitationParent(context.Background(), ordinaryUnderAdminID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, rootID, *got)
}
