package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestManagerOwnCapacityUsesOfficialRemainingQuota(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()

	root, err := client.User.Create().
		SetEmail("root-manager-remaining@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleAdmin).
		SetStatus(service.StatusActive).
		SetConcurrency(1000).
		SetRpmLimit(10000).
		Save(ctx)
	require.NoError(t, err)

	manager, err := client.User.Create().
		SetEmail("level1-manager-remaining@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleAgentLevel1).
		SetStatus(service.StatusActive).
		SetParentUserID(root.ID).
		SetConcurrency(70).
		SetRpmLimit(700).
		SetAllocatedConcurrency(100).
		SetAllocatedRpm(1000).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetEmail("child-manager-remaining@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetParentUserID(manager.ID).
		SetAllocatedConcurrency(30).
		SetAllocatedRpm(300).
		SetConcurrency(30).
		SetRpmLimit(300).
		Save(ctx)
	require.NoError(t, err)

	key := &service.APIKey{
		UserID: manager.ID,
		Key:    "sk-manager-remaining",
		Name:   "manager remaining",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, key))

	got, err := repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.NotNil(t, got.User)
	require.Equal(t, 70, got.User.Concurrency)
	require.Equal(t, 700, got.User.RPMLimit)
}

func TestManagerOwnFiniteRPMExhaustionIsNotUnlimited(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()

	root, err := client.User.Create().
		SetEmail("root-manager-rpm-exhausted@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleAdmin).
		SetStatus(service.StatusActive).
		SetConcurrency(1000).
		SetRpmLimit(10000).
		Save(ctx)
	require.NoError(t, err)

	manager, err := client.User.Create().
		SetEmail("level1-manager-rpm-exhausted@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleAgentLevel1).
		SetStatus(service.StatusActive).
		SetParentUserID(root.ID).
		SetConcurrency(1).
		SetRpmLimit(-1).
		SetAllocatedConcurrency(10).
		SetAllocatedRpm(100).
		Save(ctx)
	require.NoError(t, err)

	key := &service.APIKey{
		UserID: manager.ID,
		Key:    "sk-manager-rpm-exhausted",
		Name:   "manager rpm exhausted",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, key))

	got, err := repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.NotNil(t, got.User)
	require.Equal(t, -1, got.User.RPMLimit)
}
