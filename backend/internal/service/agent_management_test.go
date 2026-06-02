//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
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
	enterpriseProfiles map[int64]EnterpriseProfile
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
	groupDelegations      []agentGroupDelegationRecord
	inviteGroupDefaults   []agentGroupDelegationRecord
	upsertInviteDefaults  []agentGroupDelegationRecord
	deletedInviteDefaults []agentGroupDelegationRecord
}

type agentEnterpriseCleanupRepoStub struct {
	hardDeletedEnterpriseIDs []int64
	affectedUserIDsByID     map[int64][]int64
	err                     error
}

func (r *agentEnterpriseCleanupRepoStub) HardDeleteEnterpriseWithEmployees(_ context.Context, enterpriseID int64) ([]int64, error) {
	r.hardDeletedEnterpriseIDs = append(r.hardDeletedEnterpriseIDs, enterpriseID)
	if r.err != nil {
		return nil, r.err
	}
	if r.affectedUserIDsByID != nil {
		return append([]int64(nil), r.affectedUserIDsByID[enterpriseID]...), nil
	}
	return []int64{enterpriseID}, nil
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
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	total := len(out)
	out = paginateServiceSlice(out, params)
	pages := 0
	if total > 0 {
		pages = total / params.Limit()
		if total%params.Limit() > 0 {
			pages++
		}
	}
	return out, &pagination.PaginationResult{Total: int64(total), Page: params.Page, PageSize: params.Limit(), Pages: pages}, nil
}

func (r *agentManagementRepoStub) ListDirectChildrenWithSearch(_ context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]User, *pagination.PaginationResult, error) {
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
		if len(roleSet) > 0 {
			if _, ok := roleSet[user.Role]; !ok {
				continue
			}
		}
		if search != "" && !strings.Contains(strings.ToLower(user.Email), search) && !strings.Contains(strings.ToLower(user.Username), search) {
			continue
		}
		out = append(out, *user)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.Limit(), Pages: 1}, nil
}

func paginateServiceSlice[T any](items []T, params pagination.PaginationParams) []T {
	offset := params.Offset()
	if offset >= len(items) {
		return []T{}
	}
	end := offset + params.Limit()
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
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

func (r *agentManagementRepoStub) GetEnterpriseProfile(_ context.Context, userID int64) (*EnterpriseProfile, error) {
	if r.enterpriseProfiles == nil {
		return nil, nil
	}
	profile, ok := r.enterpriseProfiles[userID]
	if !ok {
		return nil, nil
	}
	clone := profile
	return &clone, nil
}

func (r *agentManagementRepoStub) UpsertEnterpriseProfile(_ context.Context, userID int64, poolConcurrency int, poolRPM int) error {
	if r.enterpriseProfiles == nil {
		r.enterpriseProfiles = map[int64]EnterpriseProfile{}
	}
	if poolConcurrency < 0 {
		poolConcurrency = 0
	}
	if poolRPM < 0 {
		poolRPM = 0
	}
	r.enterpriseProfiles[userID] = EnterpriseProfile{UserID: userID, PoolConcurrency: poolConcurrency, PoolRPM: poolRPM}
	return nil
}

func (r *agentManagementRepoStub) GetEnterpriseEmployeeQuotaUsage(_ context.Context, enterpriseID int64, excludeEmployeeID *int64) (QuotaUsageSummary, error) {
	usage := QuotaUsageSummary{}
	for _, child := range r.users {
		if child.ParentUserID == nil || *child.ParentUserID != enterpriseID || child.Role != RoleEmployee {
			continue
		}
		if excludeEmployeeID != nil && child.ID == *excludeEmployeeID {
			continue
		}
		usage = usage.WithRequest(child.Concurrency, child.RPMLimit)
	}
	return usage, nil
}

func (r *agentManagementRepoStub) UpdateAgentInviteDefaults(_ context.Context, userID int64, inviteConcurrency int, inviteRPM int) error {
	if r.agentProfiles == nil {
		r.agentProfiles = map[int64]AgentProfile{}
	}
	profile := r.agentProfiles[userID]
	profile.UserID = userID
	profile.InviteDefaultConcurrency = inviteConcurrency
	profile.InviteDefaultRPM = inviteRPM
	r.agentProfiles[userID] = profile
	return nil
}

func (r *agentManagementRepoStub) GetDirectChildQuotaUsage(_ context.Context, parentID int64, excludeChildID *int64) (QuotaUsageSummary, error) {
	usage := QuotaUsageSummary{}
	for _, child := range r.users {
		if child.ParentUserID == nil || *child.ParentUserID != parentID {
			continue
		}
		if excludeChildID != nil && child.ID == *excludeChildID {
			continue
		}
		concurrency := child.Concurrency
		rpm := child.RPMLimit
		if isAgentManagerRole(child.Role) && child.Role != RoleAdmin {
			if profile, ok := r.agentProfiles[child.ID]; ok {
				concurrency = profile.PoolConcurrency
				rpm = profile.PoolRPM
			}
		} else if child.Role == RoleEnterprise {
			if profile, ok := r.enterpriseProfiles[child.ID]; ok {
				concurrency = profile.PoolConcurrency
				rpm = profile.PoolRPM
			}
		}
		usage = usage.WithRequest(concurrency, rpm)
	}
	return usage, nil
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
	if _, ok := r.users[agentID]; !ok {
		return ErrUserNotFound
	}
	for _, child := range r.users {
		if child.ParentUserID == nil || *child.ParentUserID != agentID {
			continue
		}
		child.ParentUserID = &rootAdminID
		if child.Role == RoleAgentLevel2 {
			child.Role = RoleAgentLevel1
		}
	}
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

func (r *agentManagementRepoStub) ListInviteGroupDefaults(_ context.Context, agentID int64) ([]AgentInviteGroupDefault, error) {
	out := make([]AgentInviteGroupDefault, 0)
	for _, record := range r.inviteGroupDefaults {
		if record.managerID != agentID {
			continue
		}
		group := Group{ID: record.groupID}
		out = append(out, AgentInviteGroupDefault{
			AgentUserID:    record.managerID,
			GroupID:        record.groupID,
			RateMultiplier: record.rateMultiplier,
			Group:          &group,
		})
	}
	return out, nil
}

func (r *agentManagementRepoStub) UpsertInviteGroupDefault(_ context.Context, agentID int64, groupID int64, rateMultiplier float64) error {
	record := agentGroupDelegationRecord{managerID: agentID, groupID: groupID, rateMultiplier: rateMultiplier}
	r.upsertInviteDefaults = append(r.upsertInviteDefaults, record)
	for i := range r.inviteGroupDefaults {
		if r.inviteGroupDefaults[i].managerID == agentID && r.inviteGroupDefaults[i].groupID == groupID {
			r.inviteGroupDefaults[i].rateMultiplier = rateMultiplier
			return nil
		}
	}
	r.inviteGroupDefaults = append(r.inviteGroupDefaults, record)
	return nil
}

func (r *agentManagementRepoStub) DeleteInviteGroupDefault(_ context.Context, agentID int64, groupID int64) error {
	r.deletedInviteDefaults = append(r.deletedInviteDefaults, agentGroupDelegationRecord{managerID: agentID, groupID: groupID})
	filtered := r.inviteGroupDefaults[:0]
	for _, record := range r.inviteGroupDefaults {
		if record.managerID == agentID && record.groupID == groupID {
			continue
		}
		filtered = append(filtered, record)
	}
	r.inviteGroupDefaults = filtered
	return nil
}

func (r *agentManagementRepoStub) DeleteAgentForAdminUserDeletion(_ context.Context, user *User) ([]int64, error) {
	if user == nil {
		return nil, nil
	}
	root, err := r.GetRootAdmin(context.Background())
	if err != nil {
		return nil, err
	}
	affected := []int64{user.ID}
	if user.ParentUserID != nil {
		affected = append(affected, *user.ParentUserID)
	}
	if user.Role == RoleAgentLevel1 {
		for _, child := range r.users {
			if child.ParentUserID == nil || *child.ParentUserID != user.ID {
				continue
			}
			child.ParentUserID = &root.ID
			if child.Role == RoleAgentLevel2 {
				child.Role = RoleAgentLevel1
			}
			affected = append(affected, child.ID)
		}
		if stored, ok := r.users[user.ID]; ok {
			stored.Role = RoleUser
			stored.ParentUserID = &root.ID
		}
	} else if user.Role == RoleAgentLevel2 {
		if err := r.SetRoleAndParent(context.Background(), user.ID, RoleAgentLevel1, &root.ID); err != nil {
			return nil, err
		}
	}
	return affected, nil
}

func (r *agentManagementRepoStub) RecalculateAgentQuota(context.Context, int64) error {
	return nil
}

type agentManagementUserRepoStub struct {
	*mockUserRepo
	users map[int64]*User

	deletedUserIDs     []int64
	hardDeletedUserIDs []int64
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

func (r *agentManagementUserRepoStub) Delete(_ context.Context, id int64) error {
	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}
	r.deletedUserIDs = append(r.deletedUserIDs, id)
	delete(r.users, id)
	return nil
}

func (r *agentManagementUserRepoStub) HardDelete(_ context.Context, id int64) error {
	if _, ok := r.users[id]; !ok {
		return ErrUserNotFound
	}
	r.hardDeletedUserIDs = append(r.hardDeletedUserIDs, id)
	delete(r.users, id)
	return nil
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

type agentManagementUserGroupRateRepoStub struct {
	rates map[int64]map[int64]float64
	syncs []struct {
		userID int64
		rates  map[int64]*float64
	}
}

func (r *agentManagementUserGroupRateRepoStub) GetByUserID(_ context.Context, userID int64) (map[int64]float64, error) {
	if r.rates == nil {
		return map[int64]float64{}, nil
	}
	userRates := r.rates[userID]
	out := make(map[int64]float64, len(userRates))
	for groupID, rate := range userRates {
		out[groupID] = rate
	}
	return out, nil
}

func (r *agentManagementUserGroupRateRepoStub) GetByUserAndGroup(_ context.Context, userID, groupID int64) (*float64, error) {
	if r.rates == nil {
		return nil, nil
	}
	if rate, ok := r.rates[userID][groupID]; ok {
		return &rate, nil
	}
	return nil, nil
}

func (r *agentManagementUserGroupRateRepoStub) GetDelegatedRateByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	return nil, nil
}

func (r *agentManagementUserGroupRateRepoStub) GetRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	return nil, nil
}

func (r *agentManagementUserGroupRateRepoStub) GetByGroupID(context.Context, int64) ([]UserGroupRateEntry, error) {
	return nil, nil
}

func (r *agentManagementUserGroupRateRepoStub) SyncUserGroupRates(_ context.Context, userID int64, rates map[int64]*float64) error {
	if r.rates == nil {
		r.rates = map[int64]map[int64]float64{}
	}
	if _, ok := r.rates[userID]; !ok {
		r.rates[userID] = map[int64]float64{}
	}
	clone := make(map[int64]*float64, len(rates))
	for groupID, rate := range rates {
		if rate == nil {
			delete(r.rates[userID], groupID)
			clone[groupID] = nil
			continue
		}
		value := *rate
		r.rates[userID][groupID] = value
		clone[groupID] = &value
	}
	r.syncs = append(r.syncs, struct {
		userID int64
		rates  map[int64]*float64
	}{userID: userID, rates: clone})
	return nil
}

func (r *agentManagementUserGroupRateRepoStub) SyncGroupRateMultipliers(context.Context, int64, []GroupRateMultiplierInput) error {
	return nil
}

func (r *agentManagementUserGroupRateRepoStub) SyncGroupRPMOverrides(context.Context, int64, []GroupRPMOverrideInput) error {
	return nil
}

func (r *agentManagementUserGroupRateRepoStub) ClearGroupRPMOverrides(context.Context, int64) error {
	return nil
}

func (r *agentManagementUserGroupRateRepoStub) DeleteByGroupID(context.Context, int64) error {
	return nil
}

func (r *agentManagementUserGroupRateRepoStub) DeleteByUserID(_ context.Context, userID int64) error {
	delete(r.rates, userID)
	return nil
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
	repo.agentProfiles = map[int64]AgentProfile{
		level1ID: {UserID: level1ID, PoolConcurrency: 100, PoolRPM: 1000},
		level2ID: {UserID: level2ID, PoolConcurrency: 50, PoolRPM: 500},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	got, err := svc.UpgradeDirectUser(context.Background(), rootID, userUnderAdminID, AgentUpgradeInput{TargetRole: RoleAgentLevel1, PoolConcurrency: 100, PoolRPM: 1000})
	require.NoError(t, err)
	require.Equal(t, RoleAgentLevel1, got.Role)
	require.Contains(t, invalidator.userIDs, userUnderAdminID)

	_, err = svc.UpgradeDirectUser(context.Background(), rootID, anotherUserUnderAdminID, AgentUpgradeInput{TargetRole: RoleAgentLevel2})
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	got, err = svc.UpgradeDirectUser(context.Background(), level1ID, userUnderLevel1ID, AgentUpgradeInput{TargetRole: RoleAgentLevel2, PoolConcurrency: 50, PoolRPM: 500})
	require.NoError(t, err)
	require.Equal(t, RoleAgentLevel2, got.Role)

	_, err = svc.UpgradeDirectUser(context.Background(), level2ID, userUnderLevel2ID, AgentUpgradeInput{TargetRole: RoleAgentLevel2})
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	got, err = svc.UpgradeDirectUser(context.Background(), level2ID, userUnderLevel2ID, AgentUpgradeInput{TargetRole: RoleEnterprise, PoolConcurrency: 10, PoolRPM: 100})
	require.NoError(t, err)
	require.Equal(t, RoleEnterprise, got.Role)
}

func TestAgentManagementUpgradeToAgentCreatesProfilePool(t *testing.T) {
	rootID := int64(1)
	userID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: userID, Role: RoleUser, ParentUserID: &rootID, Concurrency: 8, RPMLimit: 80, Status: StatusActive},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	got, err := svc.UpgradeDirectUser(context.Background(), rootID, userID, AgentUpgradeInput{
		TargetRole:      RoleAgentLevel1,
		PoolConcurrency: 100,
		PoolRPM:         1000,
	})
	require.NoError(t, err)
	require.Equal(t, RoleAgentLevel1, got.Role)
	require.Equal(t, 100, repo.agentProfiles[userID].PoolConcurrency)
	require.Equal(t, 1000, repo.agentProfiles[userID].PoolRPM)
	require.Equal(t, 100, repo.users[userID].Concurrency)
	require.Equal(t, 1000, repo.users[userID].RPMLimit)
}

func TestAgentManagementUpgradeToEnterpriseCreatesProfilePool(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	userID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: userID, Role: RoleUser, ParentUserID: &level1ID, Concurrency: 8, RPMLimit: 80, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		level1ID: {UserID: level1ID, PoolConcurrency: 100, PoolRPM: 1000},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	got, err := svc.UpgradeDirectUser(context.Background(), level1ID, userID, AgentUpgradeInput{
		TargetRole:      RoleEnterprise,
		PoolConcurrency: 40,
		PoolRPM:         400,
	})
	require.NoError(t, err)
	require.Equal(t, RoleEnterprise, got.Role)
	require.Equal(t, 40, repo.enterpriseProfiles[userID].PoolConcurrency)
	require.Equal(t, 400, repo.enterpriseProfiles[userID].PoolRPM)
	require.Equal(t, 40, repo.users[userID].Concurrency)
	require.Equal(t, 400, repo.users[userID].RPMLimit)
	require.Equal(t, 60, repo.users[level1ID].Concurrency)
	require.Equal(t, 600, repo.users[level1ID].RPMLimit)
}

func TestAgentManagementAllocationCannotExceedRemaining(t *testing.T) {
	managerID := int64(2)
	childID := int64(10)
	otherChildID := int64(11)
	users := []*User{
		{ID: managerID, Role: RoleAgentLevel1},
		{ID: childID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 10, RPMLimit: 100},
		{ID: otherChildID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 60, RPMLimit: 600},
	}
	repo := newAgentManagementRepoStub(users...)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, &agentManagementAuthInvalidatorStub{})

	summary, err := svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 40, AllocatedRPM: 400})
	require.NoError(t, err)
	require.Equal(t, 100, summary.TotalConcurrency)
	require.Equal(t, 100, summary.AllocatedConcurrency)
	require.Equal(t, 0, summary.RemainingConcurrency)
	require.Equal(t, 40, repo.users[childID].Concurrency)
	require.Equal(t, 400, repo.users[childID].RPMLimit)

	_, err = svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 41, AllocatedRPM: 401})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementDirectUsersUseEffectiveQuotaFields(t *testing.T) {
	rootID := int64(1)
	childID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{
			ID:                   childID,
			Role:                 RoleUser,
			ParentUserID:         &rootID,
			Concurrency:          10,
			RPMLimit:             120,
			AllocatedConcurrency: 0,
			AllocatedRPM:         0,
			Status:               StatusActive,
		},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	result, err := svc.ListDirectUsers(context.Background(), rootID)
	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.Equal(t, 10, result.Users[0].Concurrency)
	require.Equal(t, 120, result.Users[0].RPMLimit)
	require.Equal(t, 0, result.Users[0].AllocatedConcurrency)
	require.Equal(t, 0, result.Users[0].AllocatedRPM)
}

func TestAgentManagementDirectAgentsIncludeProfilePool(t *testing.T) {
	rootID := int64(1)
	childAgentID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{
			ID:           childAgentID,
			Role:         RoleAgentLevel1,
			ParentUserID: &rootID,
			Concurrency:  5,
			RPMLimit:     50,
			Status:       StatusActive,
		},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		childAgentID: {UserID: childAgentID, PoolConcurrency: 30, PoolRPM: 300},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	result, err := svc.ListDirectAgents(context.Background(), rootID)
	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.NotNil(t, result.Users[0].AgentProfile)
	require.Equal(t, 30, result.Users[0].AgentProfile.PoolConcurrency)
	require.Equal(t, 300, result.Users[0].AgentProfile.PoolRPM)
	require.Equal(t, 5, result.Users[0].Concurrency)
	require.Equal(t, 50, result.Users[0].RPMLimit)
}

func TestAgentManagementDirectEnterprisesIncludeProfilePool(t *testing.T) {
	rootID := int64(1)
	enterpriseID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{
			ID:           enterpriseID,
			Role:         RoleEnterprise,
			ParentUserID: &rootID,
			Concurrency:  5,
			RPMLimit:     50,
			Status:       StatusActive,
		},
	)
	repo.enterpriseProfiles = map[int64]EnterpriseProfile{
		enterpriseID: {UserID: enterpriseID, PoolConcurrency: 30, PoolRPM: 300},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	result, err := svc.ListDirectEnterprises(context.Background(), rootID)
	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.NotNil(t, result.Users[0].EnterpriseProfile)
	require.Equal(t, 30, result.Users[0].EnterpriseProfile.PoolConcurrency)
	require.Equal(t, 300, result.Users[0].EnterpriseProfile.PoolRPM)
	require.Equal(t, 5, result.Users[0].Concurrency)
	require.Equal(t, 50, result.Users[0].RPMLimit)
}

func TestAgentManagementAgentPoolControlsManagerCapacity(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childID := int64(10)
	otherUserID := int64(11)
	childAgentID := int64(12)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllocatedConcurrency: 0, AllocatedRPM: 0, Concurrency: 100, RPMLimit: 1000, Status: StatusActive},
		&User{ID: childID, Role: RoleUser, ParentUserID: &managerID, AllocatedConcurrency: 0, AllocatedRPM: 0, Concurrency: 10, RPMLimit: 100, Status: StatusActive},
		&User{ID: otherUserID, Role: RoleEnterprise, ParentUserID: &managerID, AllocatedConcurrency: 0, AllocatedRPM: 0, Concurrency: 20, RPMLimit: 200, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 30, PoolRPM: 300},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	summary, err := svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 40, AllocatedRPM: 400})
	require.NoError(t, err)
	require.Equal(t, 100, summary.TotalConcurrency)
	require.Equal(t, 90, summary.AllocatedConcurrency)
	require.Equal(t, 10, summary.RemainingConcurrency)
	require.Equal(t, 10, repo.users[managerID].Concurrency)
	require.Equal(t, 100, repo.users[managerID].RPMLimit)
	require.Equal(t, 40, repo.users[childID].Concurrency)
	require.Equal(t, 400, repo.users[childID].RPMLimit)
	require.Equal(t, 0, repo.users[childID].AllocatedConcurrency)
	require.Equal(t, 0, repo.users[childID].AllocatedRPM)

	_, err = svc.UpdateAllocation(context.Background(), managerID, childID, AllocationUpdate{AllocatedConcurrency: 51, AllocatedRPM: 501})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementUpdateDirectEnterprisePoolUpdatesProfileAndEffectiveQuota(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	enterpriseID := int64(3)
	employeeID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: enterpriseID, Role: RoleEnterprise, ParentUserID: &managerID, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Concurrency: 20, RPMLimit: 200, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
	}
	repo.enterpriseProfiles = map[int64]EnterpriseProfile{
		enterpriseID: {UserID: enterpriseID, PoolConcurrency: 40, PoolRPM: 400},
	}
	invalidator := &agentManagementAuthInvalidatorStub{}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	summary, err := svc.UpdateAllocation(context.Background(), managerID, enterpriseID, AllocationUpdate{
		Concurrency: intPtr(50),
		RPM:         intPtr(500),
	})
	require.NoError(t, err)
	require.Equal(t, 50, repo.enterpriseProfiles[enterpriseID].PoolConcurrency)
	require.Equal(t, 500, repo.enterpriseProfiles[enterpriseID].PoolRPM)
	require.Equal(t, 30, repo.users[enterpriseID].Concurrency)
	require.Equal(t, 300, repo.users[enterpriseID].RPMLimit)
	require.Equal(t, 50, repo.users[managerID].Concurrency)
	require.Equal(t, 500, repo.users[managerID].RPMLimit)
	require.Equal(t, 50, summary.AllocatedConcurrency)
	require.Equal(t, 500, summary.AllocatedRPM)
	require.Contains(t, invalidator.userIDs, enterpriseID)
	require.Contains(t, invalidator.userIDs, managerID)
}

func TestAgentManagementUpdateDirectEnterprisePoolRejectsReclaimBelowEmployeeUsage(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	enterpriseID := int64(3)
	employeeID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: enterpriseID, Role: RoleEnterprise, ParentUserID: &managerID, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, Concurrency: 20, RPMLimit: 200, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
	}
	repo.enterpriseProfiles = map[int64]EnterpriseProfile{
		enterpriseID: {UserID: enterpriseID, PoolConcurrency: 40, PoolRPM: 400},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, enterpriseID, AllocationUpdate{
		Concurrency: intPtr(10),
		RPM:         intPtr(100),
	})
	require.ErrorIs(t, err, ErrAgentManagementPoolReclaimExceeded)
	var details *AgentPoolReclaimExceededError
	require.ErrorAs(t, err, &details)
	require.Equal(t, 20, details.AllocatedConcurrency)
	require.Equal(t, 10, details.RequestedConcurrency)
	require.Equal(t, 200, details.AllocatedRPM)
	require.Equal(t, 100, details.RequestedRPM)
	require.Equal(t, 40, repo.enterpriseProfiles[enterpriseID].PoolConcurrency)
	require.Equal(t, 400, repo.enterpriseProfiles[enterpriseID].PoolRPM)
}

func TestAgentManagementUpdateDirectAgentPoolUpdatesProfileAndEffectiveQuota(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childAgentID := int64(3)
	grandchildID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
		&User{ID: grandchildID, Role: RoleUser, ParentUserID: &childAgentID, Concurrency: 20, RPMLimit: 200, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 40, PoolRPM: 400},
	}
	invalidator := &agentManagementAuthInvalidatorStub{}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	summary, err := svc.UpdateAllocation(context.Background(), managerID, childAgentID, AllocationUpdate{
		Concurrency: intPtr(50),
		RPM:         intPtr(500),
	})
	require.NoError(t, err)
	require.Equal(t, 50, repo.agentProfiles[childAgentID].PoolConcurrency)
	require.Equal(t, 500, repo.agentProfiles[childAgentID].PoolRPM)
	require.Equal(t, 30, repo.users[childAgentID].Concurrency)
	require.Equal(t, 300, repo.users[childAgentID].RPMLimit)
	require.Equal(t, 50, repo.users[managerID].Concurrency)
	require.Equal(t, 500, repo.users[managerID].RPMLimit)
	require.Equal(t, 50, summary.AllocatedConcurrency)
	require.Equal(t, 500, summary.AllocatedRPM)
	require.Contains(t, invalidator.userIDs, childAgentID)
	require.Contains(t, invalidator.userIDs, managerID)
}

func TestAgentManagementUpdateDirectAgentPoolRejectsReclaimBelowChildUsage(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childAgentID := int64(3)
	grandchildID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
		&User{ID: grandchildID, Role: RoleUser, ParentUserID: &childAgentID, Concurrency: 20, RPMLimit: 200, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 40, PoolRPM: 400},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, childAgentID, AllocationUpdate{
		Concurrency: intPtr(10),
		RPM:         intPtr(100),
	})
	require.ErrorIs(t, err, ErrAgentManagementPoolReclaimExceeded)
	var details *AgentPoolReclaimExceededError
	require.ErrorAs(t, err, &details)
	require.Equal(t, 20, details.AllocatedConcurrency)
	require.Equal(t, 10, details.RequestedConcurrency)
	require.Equal(t, 200, details.AllocatedRPM)
	require.Equal(t, 100, details.RequestedRPM)
	require.Equal(t, 40, repo.agentProfiles[childAgentID].PoolConcurrency)
	require.Equal(t, 400, repo.agentProfiles[childAgentID].PoolRPM)
}

func TestAgentManagementFiniteAgentCannotAllocateUnlimitedPool(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childAgentID := int64(3)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 40, PoolRPM: 400},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, childAgentID, AllocationUpdate{
		Concurrency: intPtr(0),
		RPM:         intPtr(0),
	})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementUnlimitedAgentCanAllocateUnlimitedPool(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childAgentID := int64(3)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 0, PoolRPM: 0},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 40, PoolRPM: 400},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, childAgentID, AllocationUpdate{
		Concurrency: intPtr(0),
		RPM:         intPtr(0),
	})
	require.NoError(t, err)
	require.Equal(t, 0, repo.agentProfiles[childAgentID].PoolConcurrency)
	require.Equal(t, 0, repo.agentProfiles[childAgentID].PoolRPM)
}

func TestAgentManagementUpdateInviteDefaults(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childID := int64(3)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 6, RPMLimit: 60, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 10, PoolRPM: 100, InviteDefaultConcurrency: 1, InviteDefaultRPM: 1},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	profile, err := svc.UpdateInviteDefaults(context.Background(), managerID, AgentInviteDefaultsUpdate{
		InviteDefaultConcurrency: 4,
		InviteDefaultRPM:         40,
	})
	require.NoError(t, err)
	require.Equal(t, 4, profile.InviteDefaultConcurrency)
	require.Equal(t, 40, profile.InviteDefaultRPM)
	require.Equal(t, 4, repo.agentProfiles[managerID].InviteDefaultConcurrency)
	require.Equal(t, 40, repo.agentProfiles[managerID].InviteDefaultRPM)

	_, err = svc.UpdateInviteDefaults(context.Background(), managerID, AgentInviteDefaultsUpdate{
		InviteDefaultConcurrency: 5,
		InviteDefaultRPM:         50,
	})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)

	_, err = svc.UpdateInviteDefaults(context.Background(), managerID, AgentInviteDefaultsUpdate{
		InviteDefaultConcurrency: 0,
		InviteDefaultRPM:         0,
	})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementFiniteAgentCannotAllocateWhenExistingChildIsUnlimited(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	unlimitedChildID := int64(3)
	ordinaryChildID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: unlimitedChildID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 0, RPMLimit: 0, Status: StatusActive},
		&User{ID: ordinaryChildID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 10, RPMLimit: 100, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 100, PoolRPM: 1000},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, ordinaryChildID, AllocationUpdate{
		Concurrency: intPtr(20),
		RPM:         intPtr(200),
	})
	require.ErrorIs(t, err, ErrAgentManagementAllocationExceeded)
}

func TestAgentManagementCannotReclaimAgentPoolToFiniteWhenChildUsageIsUnlimited(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childAgentID := int64(3)
	grandchildID := int64(4)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childAgentID, Role: RoleAgentLevel2, ParentUserID: &managerID, Status: StatusActive},
		&User{ID: grandchildID, Role: RoleUser, ParentUserID: &childAgentID, Concurrency: 0, RPMLimit: 0, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID:    {UserID: managerID, PoolConcurrency: 0, PoolRPM: 0},
		childAgentID: {UserID: childAgentID, PoolConcurrency: 0, PoolRPM: 0},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	_, err := svc.UpdateAllocation(context.Background(), managerID, childAgentID, AllocationUpdate{
		Concurrency: intPtr(50),
		RPM:         intPtr(500),
	})
	require.ErrorIs(t, err, ErrAgentManagementPoolReclaimExceeded)
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
	require.Equal(t, 500, repo.users[childID].Concurrency)
	require.Equal(t, 5000, repo.users[childID].RPMLimit)
	require.Equal(t, 500, summary.AllocatedConcurrency)
	require.Equal(t, 5000, summary.AllocatedRPM)
}

func TestAgentManagementAdminSummaryUsesUnlimitedCapacity(t *testing.T) {
	adminID := int64(1)
	childID := int64(10)
	otherChildID := int64(11)
	repo := newAgentManagementRepoStub(
		&User{ID: adminID, Role: RoleAdmin, Concurrency: 5, RPMLimit: 50},
		&User{ID: childID, Role: RoleUser, ParentUserID: &adminID, Concurrency: 500, RPMLimit: 5000},
		&User{ID: otherChildID, Role: RoleAgentLevel1, ParentUserID: &adminID},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		otherChildID: {UserID: otherChildID, PoolConcurrency: 600, PoolRPM: 6000},
	}
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

func TestAgentManagementAgentSummaryIncludesInviteDefaults(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	childID := int64(3)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 6, RPMLimit: 60, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {
			UserID:                   managerID,
			PoolConcurrency:          10,
			PoolRPM:                  100,
			InviteDefaultConcurrency: 4,
			InviteDefaultRPM:         40,
		},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	summary, err := svc.GetSummary(context.Background(), managerID)
	require.NoError(t, err)
	require.NotNil(t, summary.InviteDefaults)
	require.Equal(t, 4, summary.InviteDefaults.InviteDefaultConcurrency)
	require.Equal(t, 40, summary.InviteDefaults.InviteDefaultRPM)
	require.Equal(t, 4, summary.Allocation.RemainingConcurrency)
	require.Equal(t, 40, summary.Allocation.RemainingRPM)
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
	require.Equal(t, 0, created.AllocatedConcurrency)
	require.Equal(t, 0, created.AllocatedRPM)
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

func TestAgentManagementListDirectUsersWithSearchFiltersDirectChildren(t *testing.T) {
	rootID := int64(1)
	otherManagerID := int64(2)
	aliceID := int64(10)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: otherManagerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: aliceID, Role: RoleUser, ParentUserID: &rootID, Email: "alice@example.com", Username: "alpha", Status: StatusActive},
		&User{ID: 11, Role: RoleUser, ParentUserID: &rootID, Email: "bob@example.com", Username: "beta", Status: StatusActive},
		&User{ID: 12, Role: RoleUser, ParentUserID: &otherManagerID, Email: "alice-other@example.com", Username: "outside", Status: StatusActive},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)

	result, err := svc.ListDirectUsersWithQuery(context.Background(), rootID, DirectChildrenQuery{
		Pagination: pagination.DefaultPagination(),
		Search:     "alice",
	})
	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.Equal(t, aliceID, result.Users[0].ID)
}

func TestAgentManagementCreateDirectUserCannotExceedAgentRemaining(t *testing.T) {
	managerID := int64(2)
	otherChildID := int64(11)
	repo := newAgentManagementRepoStub(
		&User{ID: managerID, Role: RoleAgentLevel1},
		&User{ID: otherChildID, Role: RoleUser, ParentUserID: &managerID, Concurrency: 7, RPMLimit: 70},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 10, PoolRPM: 100},
	}
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
	require.Equal(t, 0, created.AllocatedConcurrency)
	require.Equal(t, 0, created.AllocatedRPM)
	require.Equal(t, 3, created.Concurrency)
	require.Equal(t, 30, created.RPMLimit)
}

func TestAgentManagementCreateDirectUserAppliesInviteGroupDefaults(t *testing.T) {
	rootID := int64(1)
	managerID := int64(2)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: managerID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		managerID: {UserID: managerID, PoolConcurrency: 10, PoolRPM: 100},
	}
	repo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: managerID, groupID: groupID, rateMultiplier: 2.4},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRateRepo := &agentManagementUserGroupRateRepoStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, nil)
	svc.SetUserGroupRateRepository(groupRateRepo)

	created, err := svc.CreateDirectUser(context.Background(), managerID, CreateDirectUserInput{
		Email:                "default-group-child@example.com",
		Password:             "secret123",
		AllocatedConcurrency: 1,
		AllocatedRPM:         1,
	})
	require.NoError(t, err)

	require.Equal(t, []agentGroupDelegationRecord{
		{managerID: managerID, childID: created.ID, groupID: groupID, rateMultiplier: 2.4, canDelegate: false},
	}, repo.groupDelegations)
	require.Equal(t, []int64{groupID}, repo.users[created.ID].AllowedGroups)
	require.Equal(t, 2.4, groupRateRepo.rates[created.ID][groupID])
}

func TestAgentManagementDeleteRules(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	level1DirectUserID := int64(10)
	adminDirectUserID := int64(11)
	adminDirectEnterpriseID := int64(12)
	adminEnterpriseEmployeeID := int64(13)
	level1UnderAdminID := int64(20)
	userUnderDeletedAgentID := int64(21)
	level2UnderDeletedAgentID := int64(22)
	users := []*User{
		{ID: rootID, Role: RoleAdmin},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID},
		{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID},
		{ID: level1DirectUserID, Role: RoleUser, ParentUserID: &level1ID},
		{ID: adminDirectUserID, Role: RoleUser, ParentUserID: &rootID},
		{ID: adminDirectEnterpriseID, Role: RoleEnterprise, ParentUserID: &rootID},
		{ID: adminEnterpriseEmployeeID, Role: RoleEmployee, ParentUserID: &adminDirectEnterpriseID},
		{ID: level1UnderAdminID, Role: RoleAgentLevel1, ParentUserID: &rootID},
		{ID: userUnderDeletedAgentID, Role: RoleUser, ParentUserID: &level1UnderAdminID},
		{ID: level2UnderDeletedAgentID, Role: RoleAgentLevel2, ParentUserID: &level1UnderAdminID},
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	enterpriseCleanup := &agentEnterpriseCleanupRepoStub{
		affectedUserIDsByID: map[int64][]int64{
			adminDirectEnterpriseID: {adminDirectEnterpriseID, adminEnterpriseEmployeeID},
		},
	}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)
	svc.SetEnterpriseCleanupRepository(enterpriseCleanup)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), rootID, adminDirectUserID))
	require.Empty(t, userRepo.deletedUserIDs)
	require.Equal(t, []int64{adminDirectUserID}, userRepo.hardDeletedUserIDs)
	require.NotContains(t, repo.users, adminDirectUserID)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), rootID, adminDirectEnterpriseID))
	require.Equal(t, []int64{adminDirectEnterpriseID}, enterpriseCleanup.hardDeletedEnterpriseIDs)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), level1ID, level1DirectUserID))
	require.Equal(t, rootID, *repo.users[level1DirectUserID].ParentUserID)
	require.Equal(t, RoleUser, repo.users[level1DirectUserID].Role)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), level1ID, level2ID))
	require.Equal(t, rootID, *repo.users[level2ID].ParentUserID)
	require.Equal(t, RoleAgentLevel1, repo.users[level2ID].Role)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), rootID, level1UnderAdminID))
	require.Len(t, repo.deleteLevel1Calls, 1)
	require.Equal(t, level1UnderAdminID, repo.deleteLevel1Calls[0].agentID)
	require.Equal(t, rootID, repo.deleteLevel1Calls[0].rootAdminID)
	require.NotContains(t, repo.users, level1UnderAdminID)
	require.Equal(t, rootID, *repo.users[userUnderDeletedAgentID].ParentUserID)
	require.Equal(t, RoleUser, repo.users[userUnderDeletedAgentID].Role)
	require.Equal(t, rootID, *repo.users[level2UnderDeletedAgentID].ParentUserID)
	require.Equal(t, RoleAgentLevel1, repo.users[level2UnderDeletedAgentID].Role)
	require.Contains(t, invalidator.userIDs, adminDirectUserID)
	require.Contains(t, invalidator.userIDs, adminDirectEnterpriseID)
	require.Contains(t, invalidator.userIDs, adminEnterpriseEmployeeID)
	require.Contains(t, invalidator.userIDs, level1DirectUserID)
	require.Contains(t, invalidator.userIDs, level2ID)
	require.Contains(t, invalidator.userIDs, level1UnderAdminID)
	require.Contains(t, invalidator.userIDs, userUnderDeletedAgentID)
	require.Contains(t, invalidator.userIDs, level2UnderDeletedAgentID)
}

func TestAgentManagementDeletingLevel1AgentInvalidatesChildrenPastFirstPage(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	users := []*User{
		{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	}
	for i := int64(0); i < 1001; i++ {
		users = append(users, &User{
			ID:           100 + i,
			Role:         RoleUser,
			ParentUserID: &level1ID,
			Status:       StatusActive,
		})
	}
	repo := newAgentManagementRepoStub(users...)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), rootID, level1ID))

	require.Equal(t, rootID, *repo.users[1100].ParentUserID)
	require.Contains(t, invalidator.userIDs, level1ID)
	require.Contains(t, invalidator.userIDs, int64(1100))
}

func TestAgentManagementDeletingLevel2AgentRecalculatesLevel1Quota(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	directUserID := int64(4)
	grandchildID := int64(5)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Concurrency: 60, RPMLimit: 600, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, Concurrency: 25, RPMLimit: 250, Status: StatusActive},
		&User{ID: directUserID, Role: RoleUser, ParentUserID: &level1ID, Concurrency: 10, RPMLimit: 100, Status: StatusActive},
		&User{ID: grandchildID, Role: RoleUser, ParentUserID: &level2ID, Concurrency: 7, RPMLimit: 70, Status: StatusActive},
	)
	repo.agentProfiles = map[int64]AgentProfile{
		level1ID: {UserID: level1ID, PoolConcurrency: 100, PoolRPM: 1000},
		level2ID: {UserID: level2ID, PoolConcurrency: 30, PoolRPM: 300},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	svc := NewAgentManagementService(repo, userRepo, nil, invalidator)

	require.NoError(t, svc.DeleteDirectChild(context.Background(), level1ID, level2ID))

	require.Equal(t, rootID, *repo.users[level2ID].ParentUserID)
	require.Equal(t, RoleAgentLevel1, repo.users[level2ID].Role)
	require.Equal(t, level2ID, *repo.users[grandchildID].ParentUserID)
	require.Equal(t, 30, repo.agentProfiles[level2ID].PoolConcurrency)
	require.Equal(t, 300, repo.agentProfiles[level2ID].PoolRPM)
	require.Equal(t, 90, repo.users[level1ID].Concurrency)
	require.Equal(t, 900, repo.users[level1ID].RPMLimit)
	require.Empty(t, userRepo.deletedUserIDs)
	require.Contains(t, invalidator.userIDs, level2ID)
	require.Contains(t, invalidator.userIDs, level1ID)
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

	_, err := svc.UpgradeDirectUser(context.Background(), level1ID, userID, AgentUpgradeInput{TargetRole: RoleEnterprise})
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

func TestChildGroupDelegationOptionsShowsAssignedStateForDirectChild(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: 20, rateMultiplier: 1.5, canDelegate: true},
		{managerID: level1ID, childID: level2ID, groupID: 20, rateMultiplier: 2.4, canDelegate: false},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(
		Group{ID: 10, Name: "public", RateMultiplier: 1.2, Status: StatusActive},
		Group{ID: 20, Name: "exclusive", RateMultiplier: 0.3, IsExclusive: true, Status: StatusActive},
	)
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	options, err := svc.ListChildGroupDelegationOptions(context.Background(), level1ID, level2ID)
	require.NoError(t, err)

	require.Len(t, options, 1)
	require.Equal(t, int64(20), options[0].Group.ID)
	require.Equal(t, 1.5, options[0].EffectiveRate)
	require.True(t, options[0].CanDelegate)
	require.True(t, options[0].Assigned)
	require.Equal(t, 2.4, options[0].ChildRateMultiplier)
	require.False(t, options[0].ChildCanDelegate)
}

func TestInviteGroupDefaultOptionsShowsAssignedState(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: 20, rateMultiplier: 1.5, canDelegate: true},
	}
	repo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: level1ID, groupID: 20, rateMultiplier: 2.4},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(
		Group{ID: 10, Name: "public", RateMultiplier: 1.2, Status: StatusActive},
		Group{ID: 20, Name: "exclusive", RateMultiplier: 0.3, IsExclusive: true, Status: StatusActive},
	)
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	options, err := svc.ListInviteGroupDefaultOptions(context.Background(), level1ID)
	require.NoError(t, err)

	require.Len(t, options, 1)
	require.Equal(t, int64(20), options[0].Group.ID)
	require.Equal(t, 1.5, options[0].EffectiveRate)
	require.True(t, options[0].CanDelegate)
	require.True(t, options[0].Assigned)
	require.Equal(t, 2.4, options[0].ChildRateMultiplier)
	require.False(t, options[0].ChildCanDelegate)
}

func TestSetInviteGroupDefaultRequiresDelegableAccess(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: 20, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	err := svc.SetInviteGroupDefault(context.Background(), level1ID, 20, AgentInviteGroupDefaultInput{RateMultiplier: 2.1})
	require.ErrorIs(t, err, ErrAgentManagementForbidden)

	repo.groupDelegations = append(repo.groupDelegations, agentGroupDelegationRecord{
		managerID: rootID, childID: level1ID, groupID: 20, rateMultiplier: 1.5, canDelegate: true,
	})
	require.NoError(t, svc.SetInviteGroupDefault(context.Background(), level1ID, 20, AgentInviteGroupDefaultInput{RateMultiplier: 2.1}))
	require.Equal(t, []agentGroupDelegationRecord{{managerID: level1ID, groupID: 20, rateMultiplier: 2.1}}, repo.upsertInviteDefaults)
}

func TestApplyInviteGroupDefaultsToRegisteredChild(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	childID := int64(3)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: childID, Role: RoleUser, ParentUserID: &level1ID, Status: StatusActive},
	)
	repo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: level1ID, groupID: groupID, rateMultiplier: 2.4},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)
	groupRateRepo := &agentManagementUserGroupRateRepoStub{}
	svc.SetUserGroupRateRepository(groupRateRepo)

	require.NoError(t, svc.ApplyInviteGroupDefaultsToChild(context.Background(), level1ID, childID))

	require.Equal(t, []agentGroupDelegationRecord{
		{managerID: level1ID, childID: childID, groupID: groupID, rateMultiplier: 2.4, canDelegate: false},
	}, repo.groupDelegations)
	require.Equal(t, []int64{groupID}, repo.users[childID].AllowedGroups)
	require.Equal(t, 2.4, groupRateRepo.rates[childID][groupID])
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

func TestRemoveDelegatedExclusiveGroupCascadesToDelegatedDescendants(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	ordinaryChildID := int64(4)
	directUserID := int64(5)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: ordinaryChildID, Role: RoleUser, ParentUserID: &level2ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: directUserID, Role: RoleUser, ParentUserID: &level1ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: groupID, rateMultiplier: 1.5, canDelegate: true},
		{managerID: level1ID, childID: level2ID, groupID: groupID, rateMultiplier: 1.8, canDelegate: true},
		{managerID: level2ID, childID: ordinaryChildID, groupID: groupID, rateMultiplier: 2.1, canDelegate: false},
		{managerID: level1ID, childID: directUserID, groupID: groupID, rateMultiplier: 1.9, canDelegate: false},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, invalidator)
	groupRateRepo := &agentManagementUserGroupRateRepoStub{
		rates: map[int64]map[int64]float64{
			level1ID:        {groupID: 1.5},
			level2ID:        {groupID: 1.8},
			ordinaryChildID: {groupID: 2.1},
			directUserID:    {groupID: 1.9},
		},
	}
	svc.SetUserGroupRateRepository(groupRateRepo)

	require.NoError(t, svc.RemoveChildGroupDelegation(context.Background(), rootID, level1ID, groupID))

	require.Empty(t, repo.groupDelegations)
	require.Empty(t, repo.users[level1ID].AllowedGroups)
	require.Empty(t, repo.users[level2ID].AllowedGroups)
	require.Empty(t, repo.users[ordinaryChildID].AllowedGroups)
	require.Empty(t, repo.users[directUserID].AllowedGroups)
	require.ElementsMatch(t, []int64{level1ID, level2ID, ordinaryChildID, directUserID}, invalidator.userIDs)
	require.ElementsMatch(t, []struct {
		userID  int64
		groupID int64
	}{
		{userID: level1ID, groupID: groupID},
		{userID: level2ID, groupID: groupID},
		{userID: ordinaryChildID, groupID: groupID},
		{userID: directUserID, groupID: groupID},
	}, userRepo.removedAllowedGroups)
	require.Empty(t, groupRateRepo.rates[level1ID])
	require.Empty(t, groupRateRepo.rates[level2ID])
	require.Empty(t, groupRateRepo.rates[ordinaryChildID])
	require.Empty(t, groupRateRepo.rates[directUserID])
}

func TestDisablingChildGroupDelegationCascadesFromChildDescendants(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	ordinaryChildID := int64(4)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: ordinaryChildID, Role: RoleUser, ParentUserID: &level2ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: groupID, rateMultiplier: 1.5, canDelegate: true},
		{managerID: level1ID, childID: level2ID, groupID: groupID, rateMultiplier: 1.8, canDelegate: true},
		{managerID: level2ID, childID: ordinaryChildID, groupID: groupID, rateMultiplier: 2.1, canDelegate: false},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	invalidator := &agentManagementAuthInvalidatorStub{}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, invalidator)
	groupRateRepo := &agentManagementUserGroupRateRepoStub{
		rates: map[int64]map[int64]float64{
			level1ID:        {groupID: 1.5},
			level2ID:        {groupID: 1.8},
			ordinaryChildID: {groupID: 2.1},
		},
	}
	svc.SetUserGroupRateRepository(groupRateRepo)

	require.NoError(t, svc.SetChildGroupDelegation(context.Background(), rootID, level1ID, groupID, ChildGroupDelegationInput{RateMultiplier: 1.5, CanDelegate: false}))

	require.Len(t, repo.groupDelegations, 1)
	require.Equal(t, rootID, repo.groupDelegations[0].managerID)
	require.Equal(t, level1ID, repo.groupDelegations[0].childID)
	require.False(t, repo.groupDelegations[0].canDelegate)
	require.Equal(t, []int64{groupID}, repo.users[level1ID].AllowedGroups)
	require.Empty(t, repo.users[level2ID].AllowedGroups)
	require.Empty(t, repo.users[ordinaryChildID].AllowedGroups)
	require.ElementsMatch(t, []int64{level1ID, level2ID, ordinaryChildID}, invalidator.userIDs)
	require.ElementsMatch(t, []struct {
		userID  int64
		groupID int64
	}{
		{userID: level2ID, groupID: groupID},
		{userID: ordinaryChildID, groupID: groupID},
	}, userRepo.removedAllowedGroups)
	require.Equal(t, 1.5, groupRateRepo.rates[level1ID][groupID])
	require.Empty(t, groupRateRepo.rates[level2ID])
	require.Empty(t, groupRateRepo.rates[ordinaryChildID])
}

func TestDisablingChildGroupDelegationRemovesInviteDefaultForChildAgent(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	level2ID := int64(3)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: level2ID, Role: RoleAgentLevel2, ParentUserID: &level1ID, AllowedGroups: []int64{groupID}, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: groupID, rateMultiplier: 1.5, canDelegate: true},
		{managerID: level1ID, childID: level2ID, groupID: groupID, rateMultiplier: 1.8, canDelegate: false},
	}
	repo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: level1ID, groupID: groupID, rateMultiplier: 1.7},
		{managerID: level2ID, groupID: groupID, rateMultiplier: 2.1},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	require.NoError(t, svc.SetChildGroupDelegation(context.Background(), rootID, level1ID, groupID, ChildGroupDelegationInput{RateMultiplier: 1.5, CanDelegate: false}))

	require.Empty(t, repo.inviteGroupDefaults)
	require.ElementsMatch(t, []agentGroupDelegationRecord{
		{managerID: level1ID, groupID: groupID},
		{managerID: level2ID, groupID: groupID},
	}, repo.deletedInviteDefaults)
}

func TestRemoveDelegatedExclusiveGroupCascadesPastFirstPageOfDirectChildren(t *testing.T) {
	rootID := int64(1)
	level1ID := int64(2)
	groupID := int64(20)
	users := []*User{
		{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		{ID: level1ID, Role: RoleAgentLevel1, ParentUserID: &rootID, AllowedGroups: []int64{groupID}, Status: StatusActive},
	}
	for i := int64(0); i < 1001; i++ {
		users = append(users, &User{
			ID:             100 + i,
			Role:           RoleUser,
			ParentUserID:   &level1ID,
			AllowedGroups:  []int64{groupID},
			Status:         StatusActive,
			Email:          "child@example.com",
			Username:       "child",
			Concurrency:    1,
			RPMLimit:       1,
		})
	}
	repo := newAgentManagementRepoStub(users...)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: level1ID, groupID: groupID, rateMultiplier: 1.5, canDelegate: true},
	}
	for i := int64(0); i < 1001; i++ {
		repo.groupDelegations = append(repo.groupDelegations, agentGroupDelegationRecord{
			managerID:      level1ID,
			childID:        100 + i,
			groupID:        groupID,
			rateMultiplier: 1.8,
			canDelegate:    false,
		})
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	require.NoError(t, svc.RemoveChildGroupDelegation(context.Background(), rootID, level1ID, groupID))

	require.Empty(t, repo.groupDelegations)
	require.Empty(t, repo.users[level1ID].AllowedGroups)
	require.Empty(t, repo.users[1100].AllowedGroups)
	require.Len(t, userRepo.removedAllowedGroups, 1002)
}

func TestRemoveDelegatedExclusiveGroupCascadesToEnterpriseEmployees(t *testing.T) {
	rootID := int64(1)
	enterpriseID := int64(2)
	employeeID := int64(3)
	groupID := int64(20)
	repo := newAgentManagementRepoStub(
		&User{ID: rootID, Role: RoleAdmin, Status: StatusActive},
		&User{ID: enterpriseID, Role: RoleEnterprise, ParentUserID: &rootID, AllowedGroups: []int64{groupID}, Status: StatusActive},
		&User{ID: employeeID, Role: RoleEmployee, ParentUserID: &enterpriseID, AllowedGroups: []int64{groupID}, Status: StatusActive},
	)
	repo.groupDelegations = []agentGroupDelegationRecord{
		{managerID: rootID, childID: enterpriseID, groupID: groupID, rateMultiplier: 1.5, canDelegate: true},
		{managerID: enterpriseID, childID: employeeID, groupID: groupID, rateMultiplier: 1.5, canDelegate: false},
	}
	userRepo := &agentManagementUserRepoStub{users: repo.users}
	groupRepo := newAgentManagementGroupRepoStub(Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive})
	svc := NewAgentManagementService(repo, userRepo, groupRepo, nil)

	require.NoError(t, svc.RemoveChildGroupDelegation(context.Background(), rootID, enterpriseID, groupID))

	require.Empty(t, repo.groupDelegations)
	require.Empty(t, repo.users[enterpriseID].AllowedGroups)
	require.Empty(t, repo.users[employeeID].AllowedGroups)
	require.ElementsMatch(t, []struct {
		userID  int64
		groupID int64
	}{
		{userID: enterpriseID, groupID: groupID},
		{userID: employeeID, groupID: groupID},
	}, userRepo.removedAllowedGroups)
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
