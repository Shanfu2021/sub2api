//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type agentGroupDelegationRecord struct {
	managerID      int64
	childID        int64
	groupID        int64
	rateMultiplier float64
	canDelegate    bool
}

type agentManagementRepoStub struct {
	users  map[int64]*User
	nextID int64

	agentProfiles      map[int64]AgentProfile
	setAllocations     []AllocationUpdate
	setEffectiveQuotas []struct {
		userID      int64
		concurrency int
		rpm         int
	}
	upsertAgentProfiles []AgentProfile
	setParents          []struct {
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
	groupDelegations []agentGroupDelegationRecord
}

func newAgentManagementRepoStub(users ...*User) *agentManagementRepoStub {
	out := &agentManagementRepoStub{users: map[int64]*User{}, nextID: 1000}
	for _, user := range users {
		clone := *user
		out.users[user.ID] = &clone
		if user.ID >= out.nextID {
			out.nextID = user.ID + 1
		}
	}
	return out
}

func (r *agentManagementRepoStub) GetRootAdmin(context.Context) (*User, error) {
	var root *User
	for _, user := range r.users {
		if user.Role == RoleAdmin {
			if root == nil || user.ID < root.ID {
				root = user
			}
		}
	}
	if root == nil {
		return nil, ErrUserNotFound
	}
	clone := *root
	return &clone, nil
}

func (r *agentManagementRepoStub) CreateUser(_ context.Context, user *User) error {
	if user == nil {
		return nil
	}
	r.nextID++
	clone := *user
	clone.ID = r.nextID
	user.ID = clone.ID
	r.users[clone.ID] = &clone
	return nil
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

func (r *agentManagementRepoStub) GetAgentProfile(_ context.Context, userID int64) (*AgentProfile, error) {
	if r.agentProfiles == nil {
		return nil, nil
	}
	profile, ok := r.agentProfiles[userID]
	if !ok {
		return nil, nil
	}
	clone := profile
	return &clone, nil
}

func (r *agentManagementRepoStub) UpsertAgentProfile(_ context.Context, userID int64, poolConcurrency int, poolRPM int) error {
	if r.agentProfiles == nil {
		r.agentProfiles = map[int64]AgentProfile{}
	}
	if poolConcurrency < 0 {
		poolConcurrency = 0
	}
	if poolRPM < 0 {
		poolRPM = 0
	}
	profile := AgentProfile{UserID: userID, PoolConcurrency: poolConcurrency, PoolRPM: poolRPM}
	r.agentProfiles[userID] = profile
	r.upsertAgentProfiles = append(r.upsertAgentProfiles, profile)
	return nil
}

func (r *agentManagementRepoStub) SumDirectChildQuotaUsage(_ context.Context, parentID int64, excludeChildID *int64) (int, int, error) {
	concurrency := 0
	rpm := 0
	for _, child := range r.users {
		if child.ParentUserID == nil || *child.ParentUserID != parentID {
			continue
		}
		if excludeChildID != nil && child.ID == *excludeChildID {
			continue
		}
		if isAgentManagerRole(child.Role) && child.Role != RoleAdmin {
			if profile, ok := r.agentProfiles[child.ID]; ok {
				concurrency += profile.PoolConcurrency
				rpm += profile.PoolRPM
				continue
			}
		}
		concurrency += child.Concurrency
		rpm += child.RPMLimit
	}
	return concurrency, rpm, nil
}

func (r *agentManagementRepoStub) SetEffectiveQuota(_ context.Context, userID int64, concurrency int, rpm int) error {
	user, ok := r.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	if concurrency < 0 {
		concurrency = 0
	}
	if rpm < 0 {
		rpm = 0
	}
	user.Concurrency = concurrency
	user.RPMLimit = rpm
	r.setEffectiveQuotas = append(r.setEffectiveQuotas, struct {
		userID      int64
		concurrency int
		rpm         int
	}{userID: userID, concurrency: concurrency, rpm: rpm})
	return nil
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

func (r *agentManagementRepoStub) ListGroupDelegationsForChild(_ context.Context, childID int64) ([]AgentGroupDelegation, error) {
	out := make([]AgentGroupDelegation, 0)
	for _, delegation := range r.groupDelegations {
		if delegation.childID != childID {
			continue
		}
		group := Group{ID: delegation.groupID}
		out = append(out, AgentGroupDelegation{
			ManagerUserID:  delegation.managerID,
			ChildUserID:    delegation.childID,
			GroupID:        delegation.groupID,
			RateMultiplier: delegation.rateMultiplier,
			CanDelegate:    delegation.canDelegate,
			Group:          &group,
		})
	}
	return out, nil
}

func (r *agentManagementRepoStub) GetGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64) (*AgentGroupDelegation, error) {
	for _, delegation := range r.groupDelegations {
		if delegation.managerID == managerID && delegation.childID == childID && delegation.groupID == groupID {
			group := Group{ID: delegation.groupID}
			return &AgentGroupDelegation{
				ManagerUserID:  delegation.managerID,
				ChildUserID:    delegation.childID,
				GroupID:        delegation.groupID,
				RateMultiplier: delegation.rateMultiplier,
				CanDelegate:    delegation.canDelegate,
				Group:          &group,
			}, nil
		}
	}
	return nil, nil
}

func (r *agentManagementRepoStub) UpsertGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error {
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

func (r *agentManagementRepoStub) DeleteGroupDelegation(_ context.Context, managerID int64, childID int64, groupID int64) error {
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

type agentManagementUserRepoStub struct {
	*mockUserRepo
	users map[int64]*User

	addedAllowedGroups []struct {
		userID  int64
		groupID int64
	}
	removedAllowedGroups []struct {
		userID  int64
		groupID int64
	}
}

func (r *agentManagementUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *agentManagementUserRepoStub) AddGroupToAllowedGroups(_ context.Context, userID int64, groupID int64) error {
	r.addedAllowedGroups = append(r.addedAllowedGroups, struct {
		userID  int64
		groupID int64
	}{userID: userID, groupID: groupID})
	if user, ok := r.users[userID]; ok {
		for _, allowedID := range user.AllowedGroups {
			if allowedID == groupID {
				return nil
			}
		}
		user.AllowedGroups = append(user.AllowedGroups, groupID)
	}
	return nil
}

func (r *agentManagementUserRepoStub) RemoveGroupFromUserAllowedGroups(_ context.Context, userID int64, groupID int64) error {
	r.removedAllowedGroups = append(r.removedAllowedGroups, struct {
		userID  int64
		groupID int64
	}{userID: userID, groupID: groupID})
	if user, ok := r.users[userID]; ok {
		filtered := user.AllowedGroups[:0]
		for _, allowedID := range user.AllowedGroups {
			if allowedID != groupID {
				filtered = append(filtered, allowedID)
			}
		}
		user.AllowedGroups = filtered
	}
	return nil
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

type agentManagementGroupRepoStub struct {
	groups []Group
	byID   map[int64]*Group
}

func newAgentManagementGroupRepoStub(groups ...Group) *agentManagementGroupRepoStub {
	out := &agentManagementGroupRepoStub{
		groups: make([]Group, 0, len(groups)),
		byID:   map[int64]*Group{},
	}
	for i := range groups {
		clone := groups[i]
		out.groups = append(out.groups, clone)
		out.byID[clone.ID] = &clone
	}
	return out
}

func (r *agentManagementGroupRepoStub) Create(context.Context, *Group) error {
	panic("unexpected Create")
}
func (r *agentManagementGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	group, ok := r.byID[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	clone := *group
	return &clone, nil
}
func (r *agentManagementGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return r.GetByID(ctx, id)
}
func (r *agentManagementGroupRepoStub) Update(context.Context, *Group) error {
	panic("unexpected Update")
}
func (r *agentManagementGroupRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete")
}
func (r *agentManagementGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade")
}
func (r *agentManagementGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (r *agentManagementGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters")
}
func (r *agentManagementGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	out := make([]Group, 0, len(r.groups))
	for i := range r.groups {
		if r.groups[i].Status == StatusActive {
			out = append(out, r.groups[i])
		}
	}
	return out, nil
}
func (r *agentManagementGroupRepoStub) ListActiveByPlatform(_ context.Context, platform string) ([]Group, error) {
	out := make([]Group, 0, len(r.groups))
	for i := range r.groups {
		if r.groups[i].Status == StatusActive && r.groups[i].Platform == platform {
			out = append(out, r.groups[i])
		}
	}
	return out, nil
}
func (r *agentManagementGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName")
}
func (r *agentManagementGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount")
}
func (r *agentManagementGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID")
}
func (r *agentManagementGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs")
}
func (r *agentManagementGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup")
}
func (r *agentManagementGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders")
}

func agentRatesByGroupID(rates []AgentGroupRate) map[int64]AgentGroupRate {
	out := make(map[int64]AgentGroupRate, len(rates))
	for i := range rates {
		out[rates[i].Group.ID] = rates[i]
	}
	return out
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

func TestAgentManagementAdminSummaryUsesUnlimitedCapacity(t *testing.T) {
	adminID := int64(1)
	childID := int64(10)
	otherChildID := int64(11)
	repo := newAgentManagementRepoStub(
		&User{ID: adminID, Role: RoleAdmin, Concurrency: 5, RPMLimit: 50},
		&User{ID: childID, Role: RoleUser, ParentUserID: &adminID, AllocatedConcurrency: 500, AllocatedRPM: 5000},
		&User{ID: otherChildID, Role: RoleAgentLevel1, ParentUserID: &adminID, AllocatedConcurrency: 600, AllocatedRPM: 6000},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	summary, err := svc.GetSummary(context.Background(), adminID)
	require.NoError(t, err)
	require.True(t, summary.Allocation.UnlimitedCapacity)
	require.Equal(t, 1100, summary.Allocation.AllocatedConcurrency)
	require.Equal(t, 11000, summary.Allocation.AllocatedRPM)
	require.GreaterOrEqual(t, summary.Allocation.RemainingConcurrency, 0)
	require.GreaterOrEqual(t, summary.Allocation.RemainingRPM, 0)
}

func TestAgentManagementCreateDirectUserForAdminIsUnconstrained(t *testing.T) {
	adminID := int64(1)
	repo := newAgentManagementRepoStub(
		&User{ID: adminID, Role: RoleAdmin, Concurrency: 5, RPMLimit: 50},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	created, err := svc.CreateDirectUser(context.Background(), adminID, CreateDirectUserInput{
		Email:                "created@example.com",
		Password:             "secret123",
		Username:             "created",
		AllocatedConcurrency: 500,
		AllocatedRPM:         5000,
	})
	require.NoError(t, err)
	require.Equal(t, RoleUser, created.Role)
	require.NotNil(t, created.ParentUserID)
	require.Equal(t, adminID, *created.ParentUserID)
	require.Equal(t, 500, created.AllocatedConcurrency)
	require.Equal(t, 5000, created.AllocatedRPM)
	require.Equal(t, 500, created.Concurrency)
	require.Equal(t, 5000, created.RPMLimit)
}

func TestAgentManagementAdminsShareRootDirectUserPool(t *testing.T) {
	rootAdminID := int64(1)
	secondAdminID := int64(42)
	rootDirectUserID := int64(10)
	secondAdminDirectUserID := int64(11)
	repo := newAgentManagementRepoStub(
		&User{ID: secondAdminID, Role: RoleAdmin, Concurrency: 1, RPMLimit: 10},
		&User{ID: rootAdminID, Role: RoleAdmin, Concurrency: 5, RPMLimit: 50},
		&User{ID: rootDirectUserID, Role: RoleUser, ParentUserID: &rootAdminID},
		&User{ID: secondAdminDirectUserID, Role: RoleUser, ParentUserID: &secondAdminID},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	result, err := svc.ListDirectUsers(context.Background(), secondAdminID)
	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.Equal(t, rootDirectUserID, result.Users[0].ID)

	created, err := svc.CreateDirectUser(context.Background(), secondAdminID, CreateDirectUserInput{
		Email:                "shared-admin-pool@example.com",
		Password:             "secret123",
		AllocatedConcurrency: 999,
		AllocatedRPM:         9999,
	})
	require.NoError(t, err)
	require.NotNil(t, created.ParentUserID)
	require.Equal(t, rootAdminID, *created.ParentUserID)
}

func TestAgentManagementCreateDirectUserCannotExceedAgentRemaining(t *testing.T) {
	managerID := int64(2)
	otherChildID := int64(11)
	repo := newAgentManagementRepoStub(
		&User{ID: managerID, Role: RoleAgentLevel1, AllocatedConcurrency: 10, AllocatedRPM: 100},
		&User{ID: otherChildID, Role: RoleUser, ParentUserID: &managerID, AllocatedConcurrency: 7, AllocatedRPM: 70},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.CreateDirectUser(context.Background(), managerID, CreateDirectUserInput{
		Email:                "too-much@example.com",
		Password:             "secret123",
		AllocatedConcurrency: 4,
		AllocatedRPM:         40,
	})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)

	created, err := svc.CreateDirectUser(context.Background(), managerID, CreateDirectUserInput{
		Email:                "within@example.com",
		Password:             "secret123",
		AllocatedConcurrency: 3,
		AllocatedRPM:         30,
	})
	require.NoError(t, err)
	require.Equal(t, managerID, *created.ParentUserID)
	require.Equal(t, 3, created.AllocatedConcurrency)
	require.Equal(t, 30, created.AllocatedRPM)
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

func TestAgentGroupsShowsPublicAndDelegatedExclusiveGroups(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: 20, rateMultiplier: 1.8, canDelegate: true},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(
		Group{ID: 10, Name: "public", RateMultiplier: 1.2, Status: StatusActive},
		Group{ID: 20, Name: "exclusive", RateMultiplier: 0.3, IsExclusive: true, Status: StatusActive},
		Group{ID: 30, Name: "hidden-exclusive", RateMultiplier: 0.4, IsExclusive: true, Status: StatusActive},
	)
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	groups, err := svc.ListMyGroups(context.Background(), level1ID)
	require.NoError(t, err)

	byID := agentRatesByGroupID(groups)
	require.Len(t, byID, 2)
	require.Equal(t, 1.2, byID[10].EffectiveRate)
	require.False(t, byID[10].CanDelegate)
	require.Equal(t, "public", byID[10].Source)
	require.Equal(t, 1.8, byID[20].EffectiveRate)
	require.True(t, byID[20].CanDelegate)
	require.Equal(t, "delegated", byID[20].Source)
}

func TestAgentGroupsAdminSeesExclusiveGroupsAsDelegable(t *testing.T) {
	rootID := int64(1)
	repo := newAgentManagementRepoStub(&User{ID: rootID, Role: RoleAdmin, Status: StatusActive})
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(
		Group{ID: 10, Name: "public", RateMultiplier: 1.2, Status: StatusActive},
		Group{ID: 20, Name: "exclusive", RateMultiplier: 0.3, IsExclusive: true, Status: StatusActive},
	)
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	groups, err := svc.ListMyGroups(context.Background(), rootID)
	require.NoError(t, err)

	byID := agentRatesByGroupID(groups)
	require.Len(t, byID, 2)
	require.Equal(t, 0.3, byID[20].EffectiveRate)
	require.True(t, byID[20].CanDelegate)
	require.Equal(t, "admin_exclusive", byID[20].Source)
}

func TestDelegateExclusiveGroupRequiresManagerAccess(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	ordinaryChildID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, Status: StatusActive},
		&User{ID: ordinaryChildID, Role: RoleUser, ParentUserID: &level2ID, Status: StatusActive},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: 20, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	err := svc.SetChildGroupDelegation(context.Background(), level1ID, level2ID, 20, ChildGroupDelegationInput{RateMultiplier: 1.8, CanDelegate: true})
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	require.NoError(t, svc.SetChildGroupDelegation(context.Background(), rootID, level1ID, 20, ChildGroupDelegationInput{RateMultiplier: 1.5, CanDelegate: true}))
	require.NoError(t, svc.SetChildGroupDelegation(context.Background(), level1ID, level2ID, 20, ChildGroupDelegationInput{RateMultiplier: 1.8, CanDelegate: true}))
	require.NoError(t, svc.SetChildGroupDelegation(context.Background(), level2ID, ordinaryChildID, 20, ChildGroupDelegationInput{RateMultiplier: 2.1, CanDelegate: false}))

	require.Len(t, userRepo.addedAllowedGroups, 3)
	require.Equal(t, ordinaryChildID, userRepo.addedAllowedGroups[2].userID)
	require.Equal(t, int64(20), userRepo.addedAllowedGroups[2].groupID)
}

func TestDelegatedExclusiveGroupHidesUpstreamRate(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: 20, rateMultiplier: 1.8, canDelegate: true},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: 20, Name: "exclusive", RateMultiplier: 0.3, IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	groups, err := svc.ListMyGroups(context.Background(), level1ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, 1.8, groups[0].EffectiveRate)
	require.Equal(t, 1.8, groups[0].Group.RateMultiplier)

	payload, err := json.Marshal(groups[0])
	require.NoError(t, err)
	require.NotContains(t, string(payload), "rate_multiplier")
	require.NotContains(t, string(payload), "0.3")
	require.Contains(t, string(payload), "effective_rate")
}
