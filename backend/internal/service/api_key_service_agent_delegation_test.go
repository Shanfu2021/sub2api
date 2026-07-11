//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type apiKeyDelegatedGroupRepoStub struct {
	groups []Group
}

func (r *apiKeyDelegatedGroupRepoStub) Create(context.Context, *Group) error {
	panic("unexpected Create")
}
func (r *apiKeyDelegatedGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	for i := range r.groups {
		if r.groups[i].ID == id {
			clone := r.groups[i]
			return &clone, nil
		}
	}
	return nil, ErrGroupNotFound
}
func (r *apiKeyDelegatedGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return r.GetByID(ctx, id)
}
func (r *apiKeyDelegatedGroupRepoStub) Update(context.Context, *Group) error {
	panic("unexpected Update")
}
func (r *apiKeyDelegatedGroupRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete")
}
func (r *apiKeyDelegatedGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade")
}
func (r *apiKeyDelegatedGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (r *apiKeyDelegatedGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters")
}
func (r *apiKeyDelegatedGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return append([]Group(nil), r.groups...), nil
}
func (r *apiKeyDelegatedGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected ListActiveByPlatform")
}
func (r *apiKeyDelegatedGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected ExistsByName")
}
func (r *apiKeyDelegatedGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount")
}
func (r *apiKeyDelegatedGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID")
}
func (r *apiKeyDelegatedGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs")
}
func (r *apiKeyDelegatedGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup")
}
func (r *apiKeyDelegatedGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders")
}

type apiKeyDelegatedSubscriptionRepoStub struct{}

func (r *apiKeyDelegatedSubscriptionRepoStub) Create(context.Context, *UserSubscription) error {
	panic("unexpected Create")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) GetByID(context.Context, int64) (*UserSubscription, error) {
	panic("unexpected GetByID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*UserSubscription, error) {
	panic("unexpected GetByIDIncludeDeleted")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	panic("unexpected GetByUserIDAndGroupID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, ErrSubscriptionNotFound
}
func (r *apiKeyDelegatedSubscriptionRepoStub) Update(context.Context, *UserSubscription) error {
	panic("unexpected Update")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) Restore(context.Context, int64, string) (*UserSubscription, error) {
	panic("unexpected Restore")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	panic("unexpected ListByUserID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return []UserSubscription{}, nil
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected List")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	panic("unexpected ExistsByUserIDAndGroupID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ExistsActiveByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	panic("unexpected ExistsActiveByUserIDAndGroupID")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ExtendExpiry(context.Context, int64, time.Time) error {
	panic("unexpected ExtendExpiry")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) UpdateStatus(context.Context, int64, string) error {
	panic("unexpected UpdateStatus")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) UpdateNotes(context.Context, int64, string) error {
	panic("unexpected UpdateNotes")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ActivateWindows(context.Context, int64, time.Time) error {
	panic("unexpected ActivateWindows")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ResetUsageWindows(context.Context, int64, bool, bool, bool, time.Time) error {
	panic("unexpected ResetUsageWindows")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ResetDailyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetDailyUsage")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ResetWeeklyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetWeeklyUsage")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) ResetMonthlyUsage(context.Context, int64, *time.Time, time.Time) error {
	panic("unexpected ResetMonthlyUsage")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) IncrementUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementUsage")
}
func (r *apiKeyDelegatedSubscriptionRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	panic("unexpected BatchUpdateExpiredStatus")
}

type apiKeyDelegatedRateRepoStub struct {
	delegated map[int64]float64
	userRates map[int64]float64
}

func (r *apiKeyDelegatedRateRepoStub) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	panic("unexpected GetByUserID")
}
func (r *apiKeyDelegatedRateRepoStub) GetByUserAndGroup(_ context.Context, _ int64, groupID int64) (*float64, error) {
	if rate, ok := r.userRates[groupID]; ok {
		return &rate, nil
	}
	return nil, nil
}
func (r *apiKeyDelegatedRateRepoStub) GetDelegatedRateByUserAndGroup(_ context.Context, _ int64, groupID int64) (*float64, error) {
	if rate, ok := r.delegated[groupID]; ok {
		return &rate, nil
	}
	return nil, nil
}
func (r *apiKeyDelegatedRateRepoStub) GetRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	panic("unexpected GetRPMOverrideByUserAndGroup")
}
func (r *apiKeyDelegatedRateRepoStub) GetByGroupID(context.Context, int64) ([]UserGroupRateEntry, error) {
	panic("unexpected GetByGroupID")
}
func (r *apiKeyDelegatedRateRepoStub) SyncUserGroupRates(context.Context, int64, map[int64]*float64) error {
	panic("unexpected SyncUserGroupRates")
}
func (r *apiKeyDelegatedRateRepoStub) SyncGroupRateMultipliers(context.Context, int64, []GroupRateMultiplierInput) error {
	panic("unexpected SyncGroupRateMultipliers")
}
func (r *apiKeyDelegatedRateRepoStub) SyncGroupRPMOverrides(context.Context, int64, []GroupRPMOverrideInput) error {
	panic("unexpected SyncGroupRPMOverrides")
}
func (r *apiKeyDelegatedRateRepoStub) ClearGroupRPMOverrides(context.Context, int64) error {
	panic("unexpected ClearGroupRPMOverrides")
}
func (r *apiKeyDelegatedRateRepoStub) DeleteByGroupID(context.Context, int64) error {
	panic("unexpected DeleteByGroupID")
}
func (r *apiKeyDelegatedRateRepoStub) DeleteByUserID(context.Context, int64) error {
	panic("unexpected DeleteByUserID")
}

func TestAPIKeyServiceDelegatedExclusiveGroupCanBeListedAndBound(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	apiKeyRepo := &authRepoStub{
		create: func(_ context.Context, key *APIKey) error {
			key.ID = 99
			return nil
		},
		existsByKey: func(context.Context, string) (bool, error) {
			return false, nil
		},
	}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: userID, Role: RoleUser, Status: StatusActive}}
	groupRepo := &apiKeyDelegatedGroupRepoStub{groups: []Group{
		{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
	}}
	rateRepo := &apiKeyDelegatedRateRepoStub{delegated: map[int64]float64{groupID: 1.8}}
	svc := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, &apiKeyDelegatedSubscriptionRepoStub{}, rateRepo, nil, &config.Config{})

	groups, err := svc.GetAvailableGroups(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, groupID, groups[0].ID)
	require.Equal(t, 1.8, groups[0].RateMultiplier)

	created, err := svc.Create(context.Background(), userID, CreateAPIKeyRequest{Name: "delegated", GroupID: &groupID})
	require.NoError(t, err)
	require.NotNil(t, created.GroupID)
	require.Equal(t, groupID, *created.GroupID)
}

func TestAPIKeyServiceUserSpecificRateTakesPrecedenceOverDelegatedFallback(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	userRepo := &mockUserRepo{getByIDUser: &User{ID: userID, Role: RoleUser, Status: StatusActive}}
	groupRepo := &apiKeyDelegatedGroupRepoStub{groups: []Group{
		{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
	}}
	rateRepo := &apiKeyDelegatedRateRepoStub{
		userRates: map[int64]float64{groupID: 2.4},
		delegated: map[int64]float64{groupID: 1.8},
	}
	svc := NewAPIKeyService(nil, userRepo, groupRepo, &apiKeyDelegatedSubscriptionRepoStub{}, rateRepo, nil, &config.Config{})

	groups, err := svc.GetAvailableGroups(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, 2.4, groups[0].RateMultiplier)
}

func TestAPIKeyServiceUserSpecificRateDoesNotGrantExclusiveGroupAccess(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	userRepo := &mockUserRepo{getByIDUser: &User{ID: userID, Role: RoleUser, Status: StatusActive}}
	groupRepo := &apiKeyDelegatedGroupRepoStub{groups: []Group{
		{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
	}}
	rateRepo := &apiKeyDelegatedRateRepoStub{
		userRates: map[int64]float64{groupID: 2.4},
	}
	svc := NewAPIKeyService(nil, userRepo, groupRepo, &apiKeyDelegatedSubscriptionRepoStub{}, rateRepo, nil, &config.Config{})

	groups, err := svc.GetAvailableGroups(context.Background(), userID)
	require.NoError(t, err)
	require.Empty(t, groups)
}

func TestAPIKeyServiceAllowedExclusiveGroupUsesUserSpecificRate(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	userRepo := &mockUserRepo{getByIDUser: &User{ID: userID, Role: RoleUser, Status: StatusActive, AllowedGroups: []int64{groupID}}}
	groupRepo := &apiKeyDelegatedGroupRepoStub{groups: []Group{
		{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
	}}
	rateRepo := &apiKeyDelegatedRateRepoStub{
		userRates: map[int64]float64{groupID: 2.4},
	}
	svc := NewAPIKeyService(nil, userRepo, groupRepo, &apiKeyDelegatedSubscriptionRepoStub{}, rateRepo, nil, &config.Config{})

	groups, err := svc.GetAvailableGroups(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, 2.4, groups[0].RateMultiplier)
}

func TestAPIKeyServiceListHidesDelegatedExclusiveGroupUpstreamRate(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	apiKeyRepo := &authRepoStub{
		listByUserID: func(_ context.Context, _ int64, _ pagination.PaginationParams, _ APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
			return []APIKey{
				{
					ID:      99,
					UserID:  userID,
					GroupID: &groupID,
					Group:   &Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
				},
			}, &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 10, Pages: 1}, nil
		},
	}
	rateRepo := &apiKeyDelegatedRateRepoStub{delegated: map[int64]float64{groupID: 1.8}}
	svc := NewAPIKeyService(apiKeyRepo, nil, nil, nil, rateRepo, nil, &config.Config{})

	keys, _, err := svc.List(context.Background(), userID, pagination.PaginationParams{Page: 1, PageSize: 10}, APIKeyListFilters{})
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.NotNil(t, keys[0].Group)
	require.Equal(t, 1.8, keys[0].Group.RateMultiplier)
}

func TestAPIKeyServiceListByCurrentConcurrencyHidesDelegatedExclusiveGroupUpstreamRate(t *testing.T) {
	userID := int64(10)
	groupID := int64(20)
	apiKeyRepo := &apiKeyRepoStub{
		allowListAllByUserID: true,
		listAllByUserIDKeys: []APIKey{
			{
				ID:      99,
				UserID:  userID,
				GroupID: &groupID,
				Group:   &Group{ID: groupID, Name: "exclusive", IsExclusive: true, Status: StatusActive, RateMultiplier: 0.5},
			},
		},
	}
	rateRepo := &apiKeyDelegatedRateRepoStub{delegated: map[int64]float64{groupID: 1.8}}
	svc := NewAPIKeyService(apiKeyRepo, nil, nil, nil, rateRepo, nil, &config.Config{})

	keys, _, err := svc.List(context.Background(), userID, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    apiKeySortCurrentConcurrency,
		SortOrder: pagination.SortOrderDesc,
	}, APIKeyListFilters{})
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.NotNil(t, keys[0].Group)
	require.Equal(t, 1.8, keys[0].Group.RateMultiplier)
}
