//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/userallowedgroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

type AgentManagementRepoSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   service.AgentManagementRepository
}

func (s *AgentManagementRepoSuite) SetupTest() {
	s.ctx = context.Background()
	s.client = testEntClient(s.T())
	s.repo = NewAgentManagementRepository(s.client, integrationDB)

	s.cleanupAgentManagementGroups()
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM auth_identity_channels")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM auth_identities")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM agent_group_delegations")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM user_subscriptions")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM user_allowed_groups")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM user_affiliates")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM usage_billing_dedup")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM usage_billing_dedup_archive")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM billing_usage_entries")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM usage_logs")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM usage_cleanup_tasks")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM payment_orders")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM api_keys")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM users")
}

func (s *AgentManagementRepoSuite) TearDownTest() {
	s.cleanupAgentManagementGroups()
}

func (s *AgentManagementRepoSuite) cleanupAgentManagementGroups() {
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM agent_group_delegations")
	_, _ = integrationDB.ExecContext(s.ctx, `
		DELETE FROM user_group_rate_multipliers
		WHERE group_id IN (
			SELECT id FROM groups
			WHERE name LIKE 'exclusive-delegated-%'
			   OR name LIKE 'exclusive-delete-%'
		)`)
	_, _ = integrationDB.ExecContext(s.ctx, `
		DELETE FROM user_allowed_groups
		WHERE group_id IN (
			SELECT id FROM groups
			WHERE name LIKE 'exclusive-delegated-%'
			   OR name LIKE 'exclusive-delete-%'
		)`)
	_, _ = integrationDB.ExecContext(s.ctx, `
		DELETE FROM groups
		WHERE name LIKE 'exclusive-delegated-%'
		   OR name LIKE 'exclusive-delete-%'`)
}

func TestAgentManagementRepoSuite(t *testing.T) {
	suite.Run(t, new(AgentManagementRepoSuite))
}

func (s *AgentManagementRepoSuite) mustCreateAgentUser(email string, role string, parentID *int64, allocatedConcurrency int, allocatedRPM int) *service.User {
	s.T().Helper()

	created, err := s.client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(role).
		SetStatus(service.StatusActive).
		SetConcurrency(allocatedConcurrency).
		SetRpmLimit(allocatedRPM).
		SetAllocatedConcurrency(allocatedConcurrency).
		SetAllocatedRpm(allocatedRPM).
		SetNillableParentUserID(parentID).
		Save(s.ctx)
	s.Require().NoError(err)
	return userEntityToService(created)
}

func (s *AgentManagementRepoSuite) mustCreateAgentGroup(name string, isExclusive bool, rateMultiplier float64) *service.Group {
	s.T().Helper()
	uniqueName := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())

	created, err := s.client.Group.Create().
		SetName(uniqueName).
		SetStatus(service.StatusActive).
		SetPlatform(service.PlatformAnthropic).
		SetRateMultiplier(rateMultiplier).
		SetIsExclusive(isExclusive).
		Save(s.ctx)
	s.Require().NoError(err)
	return groupEntityToService(created)
}

func (s *AgentManagementRepoSuite) countAgentManagementRows(table string, where string, args ...any) int {
	s.T().Helper()
	var count int
	err := integrationDB.QueryRowContext(s.ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+where, args...).Scan(&count)
	s.Require().NoError(err)
	return count
}

func (s *AgentManagementRepoSuite) TestListDirectChildrenByRole() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise@test.com", service.RoleEnterprise, &level1.ID, 30, 300)
	s.mustCreateAgentUser("root-user@test.com", service.RoleUser, &root.ID, 5, 50)

	users, page, err := s.repo.ListDirectChildren(s.ctx, level1.ID, []string{service.RoleUser}, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(users, 1)
	s.Require().Equal(directUser.ID, users[0].ID)
	s.Require().Equal(int64(1), page.Total)

	agents, _, err := s.repo.ListDirectChildren(s.ctx, root.ID, []string{service.RoleAgentLevel1}, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(agents, 1)
	s.Require().Equal(level1.ID, agents[0].ID)

	enterprises, _, err := s.repo.ListDirectChildren(s.ctx, level1.ID, []string{service.RoleEnterprise}, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(enterprises, 1)
	s.Require().Equal(directEnterprise.ID, enterprises[0].ID)
}

func (s *AgentManagementRepoSuite) TestSumDirectChildAllocations() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	child1 := s.mustCreateAgentUser("child1@test.com", service.RoleUser, &level1.ID, 10, 100)
	s.mustCreateAgentUser("child2@test.com", service.RoleEnterprise, &level1.ID, 20, 200)
	s.mustCreateAgentUser("nested@test.com", service.RoleUser, &child1.ID, 30, 300)

	concurrency, rpm, err := s.repo.SumDirectChildAllocations(s.ctx, level1.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(30, concurrency)
	s.Require().Equal(300, rpm)

	concurrency, rpm, err = s.repo.SumDirectChildAllocations(s.ctx, level1.ID, &child1.ID)
	s.Require().NoError(err)
	s.Require().Equal(20, concurrency)
	s.Require().Equal(200, rpm)
}

func (s *AgentManagementRepoSuite) TestAgentProfilePoolQuotaUsage() {
	root := s.mustCreateAgentUser("root-profile@test.com", service.RoleAdmin, nil, 0, 0)
	manager := s.mustCreateAgentUser("manager-profile@test.com", service.RoleAgentLevel1, &root.ID, 0, 0)
	ordinary := s.mustCreateAgentUser("ordinary-profile@test.com", service.RoleUser, &manager.ID, 10, 100)
	enterprise := s.mustCreateAgentUser("enterprise-profile@test.com", service.RoleEnterprise, &manager.ID, 0, 0)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, manager.ID, 100, 1000))
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 30, 300))

	profile, err := s.repo.GetAgentProfile(s.ctx, manager.ID)
	s.Require().NoError(err)
	s.Require().NotNil(profile)
	s.Require().Equal(100, profile.PoolConcurrency)
	s.Require().Equal(1000, profile.PoolRPM)

	usage, err := s.repo.GetDirectChildQuotaUsage(s.ctx, manager.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(40, usage.Concurrency)
	s.Require().Equal(400, usage.RPM)
	s.Require().False(usage.UnlimitedConcurrency)
	s.Require().False(usage.UnlimitedRPM)

	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 0, 0))
	usage, err = s.repo.GetDirectChildQuotaUsage(s.ctx, manager.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(10, usage.Concurrency)
	s.Require().Equal(100, usage.RPM)
	s.Require().False(usage.UnlimitedConcurrency)
	s.Require().True(usage.UnlimitedRPM)

	s.Require().NoError(s.repo.SetEffectiveQuota(s.ctx, ordinary.ID, 25, 250))
	updated, err := s.client.User.Get(s.ctx, ordinary.ID)
	s.Require().NoError(err)
	s.Require().Equal(25, updated.Concurrency)
	s.Require().Equal(250, updated.RpmLimit)
}

func (s *AgentManagementRepoSuite) TestRehomeChildToRootAdmin() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	child := s.mustCreateAgentUser("child@test.com", service.RoleUser, &level1.ID, 10, 100)

	apiKey, err := s.client.APIKey.Create().
		SetUserID(child.ID).
		SetKey("child-api-key").
		SetName("child-key").
		SetStatus(service.StatusActive).
		Save(s.ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.SetParent(s.ctx, child.ID, &root.ID))

	got, err := s.client.User.Get(s.ctx, child.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.ParentUserID)
	s.Require().Equal(root.ID, *got.ParentUserID)
	s.Require().Equal(service.RoleUser, got.Role)
	s.Require().Equal(10, got.AllocatedConcurrency)
	s.Require().Equal(100, got.AllocatedRpm)

	keyCount, err := s.client.APIKey.Query().Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, keyCount)
	reloadedKey, err := s.client.APIKey.Get(s.ctx, apiKey.ID)
	s.Require().NoError(err)
	s.Require().Equal(child.ID, reloadedKey.UserID)
}

func (s *AgentManagementRepoSuite) TestSetParentPreservesAffiliateInviterBinding() {
	root := s.mustCreateAgentUser("root-admin-affiliate-rehome@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-affiliate-rehome@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	inviter := s.mustCreateAgentUser("inviter-affiliate-rehome@test.com", service.RoleUser, &level1.ID, 10, 100)
	child := s.mustCreateAgentUser("child-affiliate-rehome@test.com", service.RoleUser, &level1.ID, 10, 100)

	_, err := integrationDB.ExecContext(s.ctx, `
INSERT INTO user_affiliates (user_id, aff_code, inviter_id, created_at, updated_at)
VALUES ($1, $2, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($3, $4, $1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		inviter.ID,
		"INVITERREHOME",
		child.ID,
		"CHILDREHOME",
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.SetParent(s.ctx, child.ID, &root.ID))

	reloadedChild, err := s.client.User.Get(s.ctx, child.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedChild.ParentUserID)
	s.Require().Equal(root.ID, *reloadedChild.ParentUserID)

	var inviterID int64
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `
SELECT inviter_id FROM user_affiliates WHERE user_id = $1`,
		child.ID,
	).Scan(&inviterID))
	s.Require().Equal(inviter.ID, inviterID)
}

func (s *AgentManagementRepoSuite) TestDeleteLevel1AgentDeletesAccountMovesDirectUsersAndEnterprises() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise@test.com", service.RoleEnterprise, &level1.ID, 20, 200)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delete-level1-account", true, 0.3)
	enterpriseGroup := s.mustCreateAgentGroup("exclusive-delete-level1-enterprise", true, 0.4)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID, 1.8, true))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directEnterprise.ID, enterpriseGroup.ID, 2.4, false))
	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, level1.ID, 100, 1000))
	_, err := s.client.UserAllowedGroup.Create().
		SetUserID(directUser.ID).
		SetGroupID(exclusiveGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = s.client.UserAllowedGroup.Create().
		SetUserID(directEnterprise.ID).
		SetGroupID(enterpriseGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
	INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
	VALUES ($1, $2, 1.8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	       ($3, $4, 2.4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		directUser.ID,
		exclusiveGroup.ID,
		directEnterprise.ID,
		enterpriseGroup.ID,
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.DeleteLevel1AgentAndMoveChildren(s.ctx, level1.ID, root.ID))

	agentExists, err := s.client.User.Query().
		Where(user.IDEQ(level1.ID)).
		Exist(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	s.Require().False(agentExists)

	reloadedUser, err := s.client.User.Get(s.ctx, directUser.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedUser.ParentUserID)
	s.Require().Equal(root.ID, *reloadedUser.ParentUserID)
	s.Require().Equal(service.RoleUser, reloadedUser.Role)

	reloadedEnterprise, err := s.client.User.Get(s.ctx, directEnterprise.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedEnterprise.ParentUserID)
	s.Require().Equal(root.ID, *reloadedEnterprise.ParentUserID)
	s.Require().Equal(service.RoleEnterprise, reloadedEnterprise.Role)

	delegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(delegation)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, root.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(1.8, delegation.RateMultiplier)
	s.Require().True(delegation.CanDelegate)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, root.ID, directEnterprise.ID, enterpriseGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.4, delegation.RateMultiplier)
	s.Require().False(delegation.CanDelegate)
	allowedCount, err := s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(directUser.ID), userallowedgroup.GroupIDEQ(exclusiveGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, allowedCount)
	allowedCount, err = s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(directEnterprise.ID), userallowedgroup.GroupIDEQ(enterpriseGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, allowedCount)
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directUser.ID, exclusiveGroup.ID, 1.8))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directEnterprise.ID, enterpriseGroup.ID, 2.4))

	profile, err := s.repo.GetAgentProfile(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Nil(profile)
}

func (s *AgentManagementRepoSuite) TestDeleteLevel1AgentClearsOwnUsageReferencesBeforeHardDelete() {
	root := s.mustCreateAgentUser("root-admin-usage-delete@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-usage-delete@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "agent-delete-usage-account"})
	apiKey := mustCreateApiKey(s.T(), s.client, &service.APIKey{
		UserID: level1.ID,
		Key:    "sk-agent-delete-usage",
		Name:   "agent-delete-usage",
	})

	_, err := integrationDB.ExecContext(s.ctx, `
INSERT INTO usage_logs (request_id, model, input_tokens, output_tokens, total_cost, actual_cost, created_at, api_key_id, account_id, user_id)
VALUES ($1, $2, 1, 1, 0.1, 0.1, CURRENT_TIMESTAMP, $3, $4, $5)`,
		"agent-delete-usage-log",
		"claude-3",
		apiKey.ID,
		account.ID,
		level1.ID,
	)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
VALUES ($1, $2, $3)`,
		"agent-delete-usage-log",
		apiKey.ID,
		"agent-delete-usage-fingerprint",
	)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
INSERT INTO usage_cleanup_tasks (status, filters, created_by, deleted_rows, created_at, updated_at)
VALUES ($1, '{}'::jsonb, $2, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"succeeded",
		level1.ID,
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.DeleteLevel1AgentAndMoveChildren(s.ctx, level1.ID, root.ID))

	var count int
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM usage_logs WHERE user_id = $1 OR api_key_id = $2`, level1.ID, apiKey.ID).Scan(&count))
	s.Require().Zero(count)
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM usage_billing_dedup WHERE api_key_id = $1`, apiKey.ID).Scan(&count))
	s.Require().Zero(count)
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM api_keys WHERE id = $1`, apiKey.ID).Scan(&count))
	s.Require().Zero(count)
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM usage_cleanup_tasks WHERE created_by = $1`, level1.ID).Scan(&count))
	s.Require().Zero(count)
	s.Require().NoError(integrationDB.QueryRowContext(s.ctx, `SELECT COUNT(*) FROM usage_cleanup_tasks WHERE created_by = $1`, root.ID).Scan(&count))
	s.Require().Equal(1, count)
	agentExists, err := s.client.User.Query().
		Where(user.IDEQ(level1.ID)).
		Exist(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	s.Require().False(agentExists)
}

func (s *AgentManagementRepoSuite) TestGroupDelegationRoundTripAndSoftDelete() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delegated", true, 0.3)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID, 1.8, true))

	got, err := s.repo.GetGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().Equal(root.ID, got.ManagerUserID)
	s.Require().Equal(level1.ID, got.ChildUserID)
	s.Require().Equal(exclusiveGroup.ID, got.GroupID)
	s.Require().Equal(1.8, got.RateMultiplier)
	s.Require().True(got.CanDelegate)
	s.Require().NotNil(got.Group)
	s.Require().Contains(got.Group.Name, "exclusive-delegated")

	list, err := s.repo.ListGroupDelegationsForChild(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Len(list, 1)
	s.Require().Equal(1.8, list[0].RateMultiplier)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID, 2.1, false))
	got, err = s.repo.GetGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Require().Equal(2.1, got.RateMultiplier)
	s.Require().False(got.CanDelegate)

	s.Require().NoError(s.repo.DeleteGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID))
	got, err = s.repo.GetGroupDelegation(s.ctx, root.ID, level1.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(got)

	list, err = s.repo.ListGroupDelegationsForChild(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Empty(list)
}

func (s *AgentManagementRepoSuite) TestRaiseManagedGroupRateFloorUpdatesDelegationsDefaultsAndManagedUserRates() {
	root := s.mustCreateAgentUser("root-admin-rate-floor@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-rate-floor@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUserLow := s.mustCreateAgentUser("direct-user-low-rate-floor@test.com", service.RoleUser, &level1.ID, 10, 100)
	directUserHigh := s.mustCreateAgentUser("direct-user-high-rate-floor@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise-rate-floor@test.com", service.RoleEnterprise, &level1.ID, 10, 100)
	employee := s.mustCreateAgentUser("employee-rate-floor@test.com", service.RoleEmployee, &directEnterprise.ID, 10, 100)
	outsideUser := s.mustCreateAgentUser("outside-user-rate-floor@test.com", service.RoleUser, &root.ID, 10, 100)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delegated-rate-floor", true, 0.3)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUserLow.ID, exclusiveGroup.ID, 1.7, false))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUserHigh.ID, exclusiveGroup.ID, 2.3, false))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, directEnterprise.ID, employee.ID, exclusiveGroup.ID, 1.4, false))
	_, err := integrationDB.ExecContext(s.ctx, `
INSERT INTO agent_invite_group_defaults (agent_user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, 1.6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		level1.ID,
		exclusiveGroup.ID,
	)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $7, 1.5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($2, $7, 1.7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($3, $7, 2.3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($4, $7, 1.8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($5, $7, 1.4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($6, $7, 1.2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		level1.ID,
		directUserLow.ID,
		directUserHigh.ID,
		directEnterprise.ID,
		employee.ID,
		outsideUser.ID,
		exclusiveGroup.ID,
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.RaiseManagedGroupRateFloor(s.ctx, level1.ID, exclusiveGroup.ID, 2.0))

	delegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, directUserLow.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.0, delegation.RateMultiplier)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, level1.ID, directUserHigh.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.3, delegation.RateMultiplier)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, directEnterprise.ID, employee.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.0, delegation.RateMultiplier)
	s.Require().Equal(1, s.countAgentManagementRows("agent_invite_group_defaults", "agent_user_id = $1 AND group_id = $2 AND rate_multiplier = $3", level1.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", level1.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directUserLow.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directUserHigh.ID, exclusiveGroup.ID, 2.3))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directEnterprise.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", employee.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", outsideUser.ID, exclusiveGroup.ID, 1.2))
}

func (s *AgentManagementRepoSuite) TestRaiseManagedGroupRateFloorUpdatesDirectEnterpriseEmployees() {
	root := s.mustCreateAgentUser("root-admin-enterprise-rate-floor@test.com", service.RoleAdmin, nil, 1000, 10000)
	enterprise := s.mustCreateAgentUser("enterprise-rate-floor@test.com", service.RoleEnterprise, &root.ID, 100, 1000)
	employee := s.mustCreateAgentUser("employee-direct-enterprise-rate-floor@test.com", service.RoleEmployee, &enterprise.ID, 10, 100)
	outsideEmployee := s.mustCreateAgentUser("outside-employee-rate-floor@test.com", service.RoleEmployee, &root.ID, 10, 100)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delegated-enterprise-rate-floor", true, 0.3)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, enterprise.ID, employee.ID, exclusiveGroup.ID, 1.4, false))
	_, err := integrationDB.ExecContext(s.ctx, `
INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $3, 1.4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ($2, $3, 1.2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		employee.ID,
		outsideEmployee.ID,
		exclusiveGroup.ID,
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.RaiseManagedGroupRateFloor(s.ctx, enterprise.ID, exclusiveGroup.ID, 2.0))

	delegation, err := s.repo.GetGroupDelegation(s.ctx, enterprise.ID, employee.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.0, delegation.RateMultiplier)
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", employee.ID, exclusiveGroup.ID, 2.0))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", outsideEmployee.ID, exclusiveGroup.ID, 1.2))
}

func (s *AgentManagementRepoSuite) TestDeleteLevel1AgentForAdminUserDeletionDeletesAccountMovesChildrenAndPreservesDelegatedGroups() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-delete@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user-delete@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise-delete@test.com", service.RoleEnterprise, &level1.ID, 30, 300)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delete-level1", true, 0.3)
	enterpriseGroup := s.mustCreateAgentGroup("exclusive-delete-level1-admin-enterprise", true, 0.5)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, level1.ID, 100, 1000))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID, 1.8, true))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directEnterprise.ID, enterpriseGroup.ID, 2.7, true))
	_, err := s.client.UserAllowedGroup.Create().
		SetUserID(directUser.ID).
		SetGroupID(exclusiveGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = s.client.UserAllowedGroup.Create().
		SetUserID(directEnterprise.ID).
		SetGroupID(enterpriseGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
	INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
	VALUES ($1, $2, 1.8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	       ($3, $4, 2.7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		directUser.ID,
		exclusiveGroup.ID,
		directEnterprise.ID,
		enterpriseGroup.ID,
	)
	s.Require().NoError(err)

	affected, err := s.repo.DeleteAgentForAdminUserDeletion(s.ctx, level1)
	s.Require().NoError(err)
	s.Require().Contains(affected, level1.ID)
	s.Require().Contains(affected, directUser.ID)
	s.Require().Contains(affected, directEnterprise.ID)

	level1Exists, err := s.client.User.Query().
		Where(user.IDEQ(level1.ID)).
		Exist(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	s.Require().False(level1Exists)

	reloadedUser, err := s.client.User.Get(s.ctx, directUser.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedUser.ParentUserID)
	s.Require().Equal(root.ID, *reloadedUser.ParentUserID)

	reloadedEnterprise, err := s.client.User.Get(s.ctx, directEnterprise.ID)
	s.Require().NoError(err)
	s.Require().Equal(service.RoleEnterprise, reloadedEnterprise.Role)
	s.Require().NotNil(reloadedEnterprise.ParentUserID)
	s.Require().Equal(root.ID, *reloadedEnterprise.ParentUserID)

	profile, err := s.repo.GetAgentProfile(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Nil(profile)
	delegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(delegation)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, root.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(1.8, delegation.RateMultiplier)
	s.Require().True(delegation.CanDelegate)
	delegation, err = s.repo.GetGroupDelegation(s.ctx, root.ID, directEnterprise.ID, enterpriseGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(delegation)
	s.Require().Equal(2.7, delegation.RateMultiplier)
	s.Require().True(delegation.CanDelegate)

	allowedCount, err := s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(directUser.ID), userallowedgroup.GroupIDEQ(exclusiveGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, allowedCount)
	allowedCount, err = s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(directEnterprise.ID), userallowedgroup.GroupIDEQ(enterpriseGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, allowedCount)
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directUser.ID, exclusiveGroup.ID, 1.8))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directEnterprise.ID, enterpriseGroup.ID, 2.7))
}

func (s *AgentManagementRepoSuite) TestRehomeChildGroupDelegationsPreservesDirectUserAuthorizationWhenMovedToAdmin() {
	root := s.mustCreateAgentUser("root-admin-rehome-user@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-rehome-user@test.com", service.RoleAgentLevel1, &root.ID, 70, 700)
	directUser := s.mustCreateAgentUser("direct-user-rehome@test.com", service.RoleUser, &level1.ID, 10, 100)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delete-user-rehome", true, 0.3)

	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID, 2.4, false))
	_, err := s.client.UserAllowedGroup.Create().
		SetUserID(directUser.ID).
		SetGroupID(exclusiveGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = integrationDB.ExecContext(s.ctx, `
INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, 2.4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		directUser.ID,
		exclusiveGroup.ID,
	)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.RehomeChildGroupDelegations(s.ctx, level1.ID, root.ID, directUser.ID))
	s.Require().NoError(s.repo.SetParent(s.ctx, directUser.ID, &root.ID))

	oldDelegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(oldDelegation)
	newDelegation, err := s.repo.GetGroupDelegation(s.ctx, root.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().NotNil(newDelegation)
	s.Require().Equal(2.4, newDelegation.RateMultiplier)
	s.Require().False(newDelegation.CanDelegate)

	reloadedUser, err := s.client.User.Get(s.ctx, directUser.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedUser.ParentUserID)
	s.Require().Equal(root.ID, *reloadedUser.ParentUserID)
	s.Require().Equal(1, s.countAgentManagementRows("user_allowed_groups", "user_id = $1 AND group_id = $2", directUser.ID, exclusiveGroup.ID))
	s.Require().Equal(1, s.countAgentManagementRows("user_group_rate_multipliers", "user_id = $1 AND group_id = $2 AND rate_multiplier = $3", directUser.ID, exclusiveGroup.ID, 2.4))
}
