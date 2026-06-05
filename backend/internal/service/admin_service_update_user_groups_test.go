//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_UpdateUser_RemovingAllowedGroupCleansDelegatedAccessAndRate(t *testing.T) {
	base := &userRepoStub{user: &User{
		ID:            42,
		Email:         "child@example.com",
		AllowedGroups: []int64{4, 8},
		RPMLimit:      100,
	}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	rateRepo := &agentManagementUserGroupRateRepoStub{
		rates: map[int64]map[int64]float64{
			42: {4: 1.25, 8: 1.5},
		},
	}
	cleanup := &agentUserDeletionCleanupRepoStub{groupAccessAffectedUserIDs: []int64{42, 99}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:                 repo,
		redeemCodeRepo:           &redeemRepoStub{},
		userGroupRateRepo:        rateRepo,
		agentDeletionCleanupRepo: cleanup,
		authCacheInvalidator:     invalidator,
	}

	staleRemovedRate := 1.4
	allowedGroups := []int64{8}
	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		AllowedGroups: &allowedGroups,
		GroupRates: map[int64]*float64{
			4: &staleRemovedRate,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, []int64{8}, repo.lastUpdated.AllowedGroups)
	require.Len(t, cleanup.groupAccessCalls, 1)
	require.Equal(t, int64(42), cleanup.groupAccessCalls[0].userID)
	require.Equal(t, []int64{4}, cleanup.groupAccessCalls[0].groupIDs)
	require.Len(t, rateRepo.syncs, 1)
	require.Contains(t, rateRepo.syncs[0].rates, int64(4))
	require.Nil(t, rateRepo.syncs[0].rates[4], "被取消的分组必须强制清空专属倍率，不能被旧前端提交值重新写回")
	require.ElementsMatch(t, []int64{42, 99}, invalidator.userIDs)
}

func TestAdminService_UpdateUser_RaisingAgentGroupRateRaisesManagedFloor(t *testing.T) {
	base := &userRepoStub{user: &User{
		ID:            42,
		Email:         "agent@example.com",
		Role:          RoleAgentLevel1,
		AllowedGroups: []int64{4},
		RPMLimit:      100,
		GroupRates: map[int64]float64{
			4: 1.2,
		},
	}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	newRate := 1.6
	rateRepo := &agentManagementUserGroupRateRepoStub{
		rates: map[int64]map[int64]float64{
			42: {4: 1.2},
		},
	}
	cleanup := &agentUserDeletionCleanupRepoStub{rateFloorAffectedUserIDs: []int64{42, 77, 88}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:                 repo,
		redeemCodeRepo:           &redeemRepoStub{},
		userGroupRateRepo:        rateRepo,
		agentDeletionCleanupRepo: cleanup,
		authCacheInvalidator:     invalidator,
	}

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		GroupRates: map[int64]*float64{
			4: &newRate,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Len(t, cleanup.raiseGroupRateFloorCalls, 1)
	require.Equal(t, int64(42), cleanup.raiseGroupRateFloorCalls[0].userID)
	require.Equal(t, int64(4), cleanup.raiseGroupRateFloorCalls[0].groupID)
	require.Equal(t, 1.6, cleanup.raiseGroupRateFloorCalls[0].minimumRate)
	require.ElementsMatch(t, []int64{42, 77, 88}, invalidator.userIDs)
}
