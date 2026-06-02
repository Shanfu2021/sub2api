//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
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
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM api_keys")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM users")
}

func (s *AgentManagementRepoSuite) TearDownTest() {
	s.cleanupAgentManagementGroups()
}

func (s *AgentManagementRepoSuite) cleanupAgentManagementGroups() {
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM agent_group_delegations")
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

func (s *AgentManagementRepoSuite) TestListDirectChildrenByRole() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user@test.com", service.RoleUser, &level1.ID, 10, 100)
	directLevel2 := s.mustCreateAgentUser("direct-level2@test.com", service.RoleAgentLevel2, &level1.ID, 20, 200)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise@test.com", service.RoleEnterprise, &level1.ID, 30, 300)
	s.mustCreateAgentUser("nested-user@test.com", service.RoleUser, &directLevel2.ID, 5, 50)
	s.mustCreateAgentUser("root-user@test.com", service.RoleUser, &root.ID, 5, 50)

	users, page, err := s.repo.ListDirectChildren(s.ctx, level1.ID, []string{service.RoleUser}, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(users, 1)
	s.Require().Equal(directUser.ID, users[0].ID)
	s.Require().Equal(int64(1), page.Total)

	agents, _, err := s.repo.ListDirectChildren(s.ctx, level1.ID, []string{service.RoleAgentLevel2}, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(agents, 1)
	s.Require().Equal(directLevel2.ID, agents[0].ID)

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
	childAgent := s.mustCreateAgentUser("child-agent-profile@test.com", service.RoleAgentLevel2, &manager.ID, 0, 0)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, manager.ID, 100, 1000))
	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, childAgent.ID, 30, 300))

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

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, childAgent.ID, 0, 0))
	usage, err = s.repo.GetDirectChildQuotaUsage(s.ctx, manager.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(10, usage.Concurrency)
	s.Require().Equal(100, usage.RPM)
	s.Require().True(usage.UnlimitedConcurrency)
	s.Require().True(usage.UnlimitedRPM)

	s.Require().NoError(s.repo.SetEffectiveQuota(s.ctx, ordinary.ID, 25, 250))
	updated, err := s.client.User.Get(s.ctx, ordinary.ID)
	s.Require().NoError(err)
	s.Require().Equal(25, updated.Concurrency)
	s.Require().Equal(250, updated.RpmLimit)
}

func (s *AgentManagementRepoSuite) TestDetachChildToRootAdmin() {
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

func (s *AgentManagementRepoSuite) TestDetachLevel1AgentKeepsAccountMovesChildrenAndPromotesLevel2() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise@test.com", service.RoleEnterprise, &level1.ID, 20, 200)
	directLevel2 := s.mustCreateAgentUser("direct-level2@test.com", service.RoleAgentLevel2, &level1.ID, 30, 300)
	nestedUser := s.mustCreateAgentUser("nested-user@test.com", service.RoleUser, &directLevel2.ID, 5, 50)

	s.Require().NoError(s.repo.DetachLevel1AgentAndMoveChildren(s.ctx, level1.ID, root.ID))

	detachedAgent, err := s.client.User.Query().
		Where(user.IDEQ(level1.ID)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Nil(detachedAgent.DeletedAt)
	s.Require().NotNil(detachedAgent.ParentUserID)
	s.Require().Equal(root.ID, *detachedAgent.ParentUserID)
	s.Require().Equal(service.RoleUser, detachedAgent.Role)

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

	reloadedAgent, err := s.client.User.Get(s.ctx, directLevel2.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedAgent.ParentUserID)
	s.Require().Equal(root.ID, *reloadedAgent.ParentUserID)
	s.Require().Equal(service.RoleAgentLevel1, reloadedAgent.Role)

	reloadedNested, err := s.client.User.Get(s.ctx, nestedUser.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedNested.ParentUserID)
	s.Require().Equal(directLevel2.ID, *reloadedNested.ParentUserID)
	s.Require().Equal(service.RoleUser, reloadedNested.Role)
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

func (s *AgentManagementRepoSuite) TestRehomeLevel1AgentForAdminUserDeletionKeepsAccountMovesChildrenAndClearsAgentData() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-delete@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user-delete@test.com", service.RoleUser, &level1.ID, 10, 100)
	directLevel2 := s.mustCreateAgentUser("direct-level2-delete@test.com", service.RoleAgentLevel2, &level1.ID, 30, 300)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delete-level1", true, 0.3)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, level1.ID, 100, 1000))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID, 1.8, true))
	_, err := s.client.UserAllowedGroup.Create().
		SetUserID(directUser.ID).
		SetGroupID(exclusiveGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	affected, err := s.repo.RehomeAgentForAdminUserDeletion(s.ctx, level1)
	s.Require().NoError(err)
	s.Require().Contains(affected, level1.ID)
	s.Require().Contains(affected, directUser.ID)
	s.Require().Contains(affected, directLevel2.ID)

	reloadedLevel1, err := s.client.User.Get(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Equal(service.RoleUser, reloadedLevel1.Role)
	s.Require().NotNil(reloadedLevel1.ParentUserID)
	s.Require().Equal(root.ID, *reloadedLevel1.ParentUserID)

	reloadedUser, err := s.client.User.Get(s.ctx, directUser.ID)
	s.Require().NoError(err)
	s.Require().NotNil(reloadedUser.ParentUserID)
	s.Require().Equal(root.ID, *reloadedUser.ParentUserID)

	reloadedLevel2, err := s.client.User.Get(s.ctx, directLevel2.ID)
	s.Require().NoError(err)
	s.Require().Equal(service.RoleAgentLevel1, reloadedLevel2.Role)
	s.Require().NotNil(reloadedLevel2.ParentUserID)
	s.Require().Equal(root.ID, *reloadedLevel2.ParentUserID)

	profile, err := s.repo.GetAgentProfile(s.ctx, level1.ID)
	s.Require().NoError(err)
	s.Require().Nil(profile)
	delegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, directUser.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(delegation)
	allowedCount, err := s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(directUser.ID), userallowedgroup.GroupIDEQ(exclusiveGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Zero(allowedCount)
}

func (s *AgentManagementRepoSuite) TestRehomeLevel2AgentForAdminUserDeletionPromotesToLevel1AndPreservesProfile() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1-parent-delete@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	level2 := s.mustCreateAgentUser("level2-delete@test.com", service.RoleAgentLevel2, &level1.ID, 30, 300)
	exclusiveGroup := s.mustCreateAgentGroup("exclusive-delete-level2", true, 0.3)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, level2.ID, 30, 300))
	s.Require().NoError(s.repo.UpsertGroupDelegation(s.ctx, level1.ID, level2.ID, exclusiveGroup.ID, 1.8, false))
	_, err := s.client.UserAllowedGroup.Create().
		SetUserID(level2.ID).
		SetGroupID(exclusiveGroup.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	affected, err := s.repo.RehomeAgentForAdminUserDeletion(s.ctx, level2)
	s.Require().NoError(err)
	s.Require().Contains(affected, level1.ID)
	s.Require().Contains(affected, level2.ID)

	reloadedLevel2, err := s.client.User.Get(s.ctx, level2.ID)
	s.Require().NoError(err)
	s.Require().Equal(service.RoleAgentLevel1, reloadedLevel2.Role)
	s.Require().NotNil(reloadedLevel2.ParentUserID)
	s.Require().Equal(root.ID, *reloadedLevel2.ParentUserID)
	s.Require().Equal(30, reloadedLevel2.AllocatedConcurrency)
	s.Require().Equal(300, reloadedLevel2.AllocatedRpm)

	profile, err := s.repo.GetAgentProfile(s.ctx, level2.ID)
	s.Require().NoError(err)
	s.Require().NotNil(profile)
	s.Require().Equal(30, profile.PoolConcurrency)
	s.Require().Equal(300, profile.PoolRPM)

	delegation, err := s.repo.GetGroupDelegation(s.ctx, level1.ID, level2.ID, exclusiveGroup.ID)
	s.Require().NoError(err)
	s.Require().Nil(delegation)
	allowedCount, err := s.client.UserAllowedGroup.Query().
		Where(userallowedgroup.UserIDEQ(level2.ID), userallowedgroup.GroupIDEQ(exclusiveGroup.ID)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Require().Zero(allowedCount)
}
