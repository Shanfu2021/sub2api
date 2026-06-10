package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type agentAllocationRateRepoStub struct {
	delegated map[int64]float64
	userRates map[int64]float64
}

func (r *agentAllocationRateRepoStub) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	panic("unexpected GetByUserID")
}
func (r *agentAllocationRateRepoStub) GetByUserAndGroup(_ context.Context, _ int64, groupID int64) (*float64, error) {
	if rate, ok := r.userRates[groupID]; ok {
		return &rate, nil
	}
	return nil, nil
}
func (r *agentAllocationRateRepoStub) GetDelegatedRateByUserAndGroup(_ context.Context, _ int64, groupID int64) (*float64, error) {
	if rate, ok := r.delegated[groupID]; ok {
		return &rate, nil
	}
	return nil, nil
}
func (r *agentAllocationRateRepoStub) GetRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	return nil, nil
}
func (r *agentAllocationRateRepoStub) GetByGroupID(context.Context, int64) ([]UserGroupRateEntry, error) {
	panic("unexpected GetByGroupID")
}
func (r *agentAllocationRateRepoStub) SyncUserGroupRates(context.Context, int64, map[int64]*float64) error {
	panic("unexpected SyncUserGroupRates")
}
func (r *agentAllocationRateRepoStub) SyncGroupRateMultipliers(context.Context, int64, []GroupRateMultiplierInput) error {
	panic("unexpected SyncGroupRateMultipliers")
}
func (r *agentAllocationRateRepoStub) SyncGroupRPMOverrides(context.Context, int64, []GroupRPMOverrideInput) error {
	panic("unexpected SyncGroupRPMOverrides")
}
func (r *agentAllocationRateRepoStub) ClearGroupRPMOverrides(context.Context, int64) error {
	panic("unexpected ClearGroupRPMOverrides")
}
func (r *agentAllocationRateRepoStub) DeleteByGroupID(context.Context, int64) error {
	panic("unexpected DeleteByGroupID")
}
func (r *agentAllocationRateRepoStub) DeleteByUserID(context.Context, int64) error {
	panic("unexpected DeleteByUserID")
}

var _ UserGroupRateRepository = (*agentAllocationRateRepoStub)(nil)
var _ DelegatedGroupRateRepository = (*agentAllocationRateRepoStub)(nil)

func TestAuthSnapshotUsesAllocatedConcurrencyAndRPM(t *testing.T) {
	parentID := int64(1)
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	apiKey := &APIKey{
		ID:     11,
		UserID: 22,
		Status: StatusActive,
		User: &User{
			ID:                   22,
			Role:                 RoleUser,
			ParentUserID:         &parentID,
			Status:               StatusActive,
			Balance:              10,
			Concurrency:          999,
			RPMLimit:             9999,
			AllocatedConcurrency: 7,
			AllocatedRPM:         70,
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)

	require.NotNil(t, snapshot)
	require.Equal(t, 7, snapshot.User.Concurrency)
	require.Equal(t, 70, snapshot.User.RPMLimit)
}

func TestAuthSnapshotLeavesUnmanagedUserConcurrencyAndRPM(t *testing.T) {
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	apiKey := &APIKey{
		ID:     11,
		UserID: 22,
		Status: StatusActive,
		User: &User{
			ID:                   22,
			Role:                 RoleUser,
			Status:               StatusActive,
			Balance:              10,
			Concurrency:          5,
			RPMLimit:             50,
			AllocatedConcurrency: 7,
			AllocatedRPM:         70,
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)

	require.NotNil(t, snapshot)
	require.Equal(t, 5, snapshot.User.Concurrency)
	require.Equal(t, 50, snapshot.User.RPMLimit)
}

func TestAuthSnapshotPreservesManagerRemainingAllocation(t *testing.T) {
	parentID := int64(1)
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	apiKey := &APIKey{
		ID:     11,
		UserID: 22,
		Status: StatusActive,
		User: &User{
			ID:                   22,
			Role:                 RoleAgentLevel1,
			ParentUserID:         &parentID,
			Status:               StatusActive,
			Balance:              10,
			Concurrency:          70,
			RPMLimit:             700,
			AllocatedConcurrency: 100,
			AllocatedRPM:         1000,
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)

	require.NotNil(t, snapshot)
	require.Equal(t, 70, snapshot.User.Concurrency)
	require.Equal(t, 700, snapshot.User.RPMLimit)
}

func TestExclusiveGroupUsesUserSpecificEffectiveRate(t *testing.T) {
	groupID := int64(9)
	userRate := 2.4
	rateRepo := &agentAllocationRateRepoStub{userRates: map[int64]float64{groupID: userRate}}
	svc := NewAPIKeyService(nil, nil, nil, nil, rateRepo, nil, &config.Config{})
	apiKey := &APIKey{
		ID:      11,
		UserID:  22,
		GroupID: &groupID,
		Status:  StatusActive,
		User: &User{
			ID:      22,
			Role:    RoleUser,
			Status:  StatusActive,
			Balance: 10,
		},
		Group: &Group{
			ID:               groupID,
			Name:             "exclusive",
			Platform:         PlatformAnthropic,
			Status:           StatusActive,
			IsExclusive:      true,
			SubscriptionType: SubscriptionTypeStandard,
			RateMultiplier:   0.3,
		},
	}

	snapshot := svc.snapshotFromAPIKey(context.Background(), apiKey)

	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.Group)
	require.Equal(t, userRate, snapshot.Group.RateMultiplier)
}

func TestAgentBalanceDoesNotBlockChildUsage(t *testing.T) {
	parentID := int64(1)
	cache := &billingCacheWorkerStub{
		balances: map[int64]float64{
			parentID: 0,
			22:       5,
		},
	}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	user := &User{
		ID:           22,
		Role:         RoleUser,
		ParentUserID: &parentID,
		Status:       StatusActive,
	}

	err := svc.CheckBillingEligibility(context.Background(), user, &APIKey{ID: 11}, &Group{ID: 9}, nil, "")

	require.NoError(t, err)
	require.Equal(t, []int64{22}, cache.balanceLookups)
}

func (b *billingCacheWorkerStub) balanceForUser(userID int64) (float64, error) {
	if b.balances == nil {
		return 0, errors.New("not implemented")
	}
	b.balanceLookups = append(b.balanceLookups, userID)
	return b.balances[userID], nil
}
