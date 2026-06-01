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
	s.repo = NewAgentManagementRepository(s.client)

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
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM user_allowed_groups WHERE group_id IN (SELECT id FROM groups WHERE name LIKE 'exclusive-delegated-%')")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM groups WHERE name LIKE 'exclusive-delegated-%'")
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

func (s *AgentManagementRepoSuite) TestDeleteLevel1AgentMovesChildrenAndPromotesLevel2() {
	root := s.mustCreateAgentUser("root-admin@test.com", service.RoleAdmin, nil, 1000, 10000)
	level1 := s.mustCreateAgentUser("level1@test.com", service.RoleAgentLevel1, &root.ID, 100, 1000)
	directUser := s.mustCreateAgentUser("direct-user@test.com", service.RoleUser, &level1.ID, 10, 100)
	directEnterprise := s.mustCreateAgentUser("direct-enterprise@test.com", service.RoleEnterprise, &level1.ID, 20, 200)
	directLevel2 := s.mustCreateAgentUser("direct-level2@test.com", service.RoleAgentLevel2, &level1.ID, 30, 300)
	nestedUser := s.mustCreateAgentUser("nested-user@test.com", service.RoleUser, &directLevel2.ID, 5, 50)

	s.Require().NoError(s.repo.DeleteLevel1AgentAndMoveChildren(s.ctx, level1.ID, root.ID))

	deletedAgent, err := s.client.User.Query().
		Where(user.IDEQ(level1.ID)).
		Only(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	s.Require().NotNil(deletedAgent.DeletedAt)

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
