package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeAgentManagementService struct {
	updateAllocationCalls         int
	upgradeCalls                  int
	createDirectUserCalls         int
	listDirectUsersCalls          int
	listSearch                    string
	listQuery                     service.DirectChildrenQuery
	createActorID                 int64
	createInput                   service.CreateDirectUserInput
	updateActorID                 int64
	updateChildID                 int64
	updateAllocation              service.AllocationUpdate
	updateAllocationErr           error
	upgradeActorID                int64
	upgradeChildID                int64
	upgradeInput                  service.AgentUpgradeInput
	listChildGroupsCalls          int
	listChildGroupsActor          int64
	listChildGroupsChild          int64
	listInviteDefaultGroupsCalls  int
	listInviteDefaultGroupsActor  int64
	setInviteDefaultGroupCalls    int
	setInviteDefaultGroupActor    int64
	setInviteDefaultGroupID       int64
	setInviteDefaultGroupInput    service.AgentInviteGroupDefaultInput
	removeInviteDefaultGroupCalls int
	removeInviteDefaultGroupActor int64
	removeInviteDefaultGroupID    int64
}

func (s *fakeAgentManagementService) ListDirectUsers(context.Context, int64) (*service.DirectChildrenResult, error) {
	s.listDirectUsersCalls++
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectUsersWithQuery(_ context.Context, _ int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error) {
	s.listDirectUsersCalls++
	s.listSearch = query.Search
	s.listQuery = query
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectAgents(context.Context, int64) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectAgentsWithQuery(context.Context, int64, service.DirectChildrenQuery) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectEnterprises(context.Context, int64) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectEnterprisesWithQuery(context.Context, int64, service.DirectChildrenQuery) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) GetSummary(context.Context, int64) (*service.AgentManagementSummary, error) {
	return &service.AgentManagementSummary{}, nil
}

func (s *fakeAgentManagementService) CreateDirectUser(_ context.Context, actorID int64, input service.CreateDirectUserInput) (*service.User, error) {
	s.createDirectUserCalls++
	s.createActorID = actorID
	s.createInput = input
	return &service.User{
		ID:                   99,
		Email:                input.Email,
		Username:             input.Username,
		Role:                 service.RoleUser,
		ParentUserID:         &actorID,
		Concurrency:          input.AllocatedConcurrency,
		RPMLimit:             input.AllocatedRPM,
		AllocatedConcurrency: input.AllocatedConcurrency,
		AllocatedRPM:         input.AllocatedRPM,
		Status:               service.StatusActive,
	}, nil
}

func (s *fakeAgentManagementService) UpdateAllocation(_ context.Context, actorID int64, childID int64, req service.AllocationUpdate) (*service.AllocationSummary, error) {
	concurrency := req.RequestedConcurrency()
	rpm := req.RequestedRPM()
	s.updateAllocationCalls++
	s.updateActorID = actorID
	s.updateChildID = childID
	s.updateAllocation = req
	if s.updateAllocationErr != nil {
		return nil, s.updateAllocationErr
	}
	return &service.AllocationSummary{
		TotalConcurrency:     20,
		AllocatedConcurrency: concurrency,
		RemainingConcurrency: 20 - concurrency,
		TotalRPM:             200,
		AllocatedRPM:         rpm,
		RemainingRPM:         200 - rpm,
	}, nil
}

func (s *fakeAgentManagementService) UpdateInviteDefaults(context.Context, int64, service.AgentInviteDefaultsUpdate) (*service.AgentProfile, error) {
	return &service.AgentProfile{InviteDefaultConcurrency: 1, InviteDefaultRPM: 1}, nil
}

func (s *fakeAgentManagementService) UpgradeDirectUser(_ context.Context, actorID int64, childID int64, input service.AgentUpgradeInput) (*service.User, error) {
	s.upgradeCalls++
	s.upgradeActorID = actorID
	s.upgradeChildID = childID
	s.upgradeInput = input
	return &service.User{ID: childID, Role: input.TargetRole}, nil
}

func (s *fakeAgentManagementService) DeleteDirectChild(context.Context, int64, int64) error {
	return nil
}

func (s *fakeAgentManagementService) ListMyGroups(context.Context, int64) ([]service.AgentGroupRate, error) {
	return []service.AgentGroupRate{}, nil
}

func (s *fakeAgentManagementService) ListChildGroupDelegationOptions(_ context.Context, actorID int64, childID int64) ([]service.ChildGroupDelegationOption, error) {
	s.listChildGroupsCalls++
	s.listChildGroupsActor = actorID
	s.listChildGroupsChild = childID
	return []service.ChildGroupDelegationOption{
		{
			Group:               service.Group{ID: 20, Name: "exclusive", RateMultiplier: 1.5, IsExclusive: true, Status: service.StatusActive},
			EffectiveRate:       1.5,
			CanDelegate:         true,
			Source:              "delegated",
			Assigned:            true,
			ChildRateMultiplier: 2.4,
			ChildCanDelegate:    false,
		},
	}, nil
}

func (s *fakeAgentManagementService) SetChildGroupDelegation(context.Context, int64, int64, int64, service.ChildGroupDelegationInput) error {
	return nil
}

func (s *fakeAgentManagementService) RemoveChildGroupDelegation(context.Context, int64, int64, int64) error {
	return nil
}

func (s *fakeAgentManagementService) ListInviteGroupDefaultOptions(_ context.Context, actorID int64) ([]service.ChildGroupDelegationOption, error) {
	s.listInviteDefaultGroupsCalls++
	s.listInviteDefaultGroupsActor = actorID
	return []service.ChildGroupDelegationOption{
		{
			Group:               service.Group{ID: 20, Name: "exclusive", RateMultiplier: 1.5, IsExclusive: true, Status: service.StatusActive},
			EffectiveRate:       1.5,
			CanDelegate:         true,
			Source:              "delegated",
			Assigned:            true,
			ChildRateMultiplier: 2.4,
			ChildCanDelegate:    false,
		},
	}, nil
}

func (s *fakeAgentManagementService) SetInviteGroupDefault(_ context.Context, actorID int64, groupID int64, input service.AgentInviteGroupDefaultInput) error {
	s.setInviteDefaultGroupCalls++
	s.setInviteDefaultGroupActor = actorID
	s.setInviteDefaultGroupID = groupID
	s.setInviteDefaultGroupInput = input
	return nil
}

func (s *fakeAgentManagementService) RemoveInviteGroupDefault(_ context.Context, actorID int64, groupID int64) error {
	s.removeInviteDefaultGroupCalls++
	s.removeInviteDefaultGroupActor = actorID
	s.removeInviteDefaultGroupID = groupID
	return nil
}

func newAgentManagementHandlerTestRouter(svc *fakeAgentManagementService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := newAgentManagementHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	r.GET("/direct-users", h.ListDirectUsers)
	r.PUT("/children/:id/allocation", h.UpdateAllocation)
	r.POST("/children/:id/upgrade", h.UpgradeDirectUser)
	r.POST("/direct-users", h.CreateDirectUser)
	r.GET("/children/:id/groups", h.ListChildGroupDelegationOptions)
	r.GET("/invite-default-groups", h.ListInviteGroupDefaultOptions)
	r.PUT("/invite-default-groups/:group_id", h.SetInviteGroupDefault)
	r.DELETE("/invite-default-groups/:group_id", h.RemoveInviteGroupDefault)
	return r
}

func TestAgentManagementHandlerPassesSearchToDirectUsers(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/direct-users?search=alice", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.listDirectUsersCalls)
	require.Equal(t, "alice", svc.listSearch)
}

func TestAgentManagementHandlerPassesPaginationToDirectUsers(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/direct-users?page=3&page_size=50&search=alice", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.listDirectUsersCalls)
	require.Equal(t, "alice", svc.listQuery.Search)
	require.Equal(t, 3, svc.listQuery.Pagination.Page)
	require.Equal(t, 50, svc.listQuery.Pagination.PageSize)
}

func TestAgentManagementHandlerIncludesReadOnlyBalance(t *testing.T) {
	got := agentManagedUserFromService(&service.User{
		ID:      7,
		Email:   "direct@example.com",
		Role:    service.RoleUser,
		Balance: 12.5,
		Status:  service.StatusActive,
	})

	require.Equal(t, 12.5, got.Balance)
}

func TestAgentManagementHandlerRejectsBalancePayload(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/children/7/allocation", strings.NewReader(`{
		"allocated_concurrency": 5,
		"allocated_rpm": 60,
		"balance": 100
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, 0, svc.updateAllocationCalls)
}

func TestAgentManagementHandlerRoutesUseAuthenticatedUser(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/children/7/allocation", strings.NewReader(`{
		"actor_id": 999,
		"concurrency": 5,
		"rpm": 60
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.updateAllocationCalls)
	require.Equal(t, int64(42), svc.updateActorID)
	require.Equal(t, int64(7), svc.updateChildID)
	require.NotNil(t, svc.updateAllocation.Concurrency)
	require.NotNil(t, svc.updateAllocation.RPM)
	require.Equal(t, 5, *svc.updateAllocation.Concurrency)
	require.Equal(t, 60, *svc.updateAllocation.RPM)
}

func TestAgentManagementHandlerAcceptsLegacyAllocationPayload(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/children/7/allocation", strings.NewReader(`{
		"allocated_concurrency": 5,
		"allocated_rpm": 60
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.updateAllocationCalls)
	require.Equal(t, service.AllocationUpdate{AllocatedConcurrency: 5, AllocatedRPM: 60}, svc.updateAllocation)
}

func TestAgentManagementHandlerReturnsPoolReclaimDetails(t *testing.T) {
	svc := &fakeAgentManagementService{
		updateAllocationErr: fmt.Errorf("wrapped: %w", &service.AgentPoolReclaimExceededError{
			AllocatedConcurrency: 20,
			RequestedConcurrency: 10,
			AllocatedRPM:         200,
			RequestedRPM:         100,
		}),
	}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/children/7/allocation", strings.NewReader(`{
		"concurrency": 10,
		"rpm": 100
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "AGENT_MANAGEMENT_POOL_RECLAIM_EXCEEDED", body["reason"])
	metadata, ok := body["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "20", metadata["allocated_concurrency"])
	require.Equal(t, "10", metadata["requested_concurrency"])
	require.Equal(t, "200", metadata["allocated_rpm"])
	require.Equal(t, "100", metadata["requested_rpm"])
}

func TestAgentManagementHandlerCreatesDirectUserWithAuthenticatedUser(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/direct-users", strings.NewReader(`{
		"actor_id": 999,
		"email": "direct@example.com",
		"password": "secret123",
		"username": "direct",
		"allocated_concurrency": 5,
		"allocated_rpm": 60
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.createDirectUserCalls)
	require.Equal(t, int64(42), svc.createActorID)
	require.Equal(t, service.CreateDirectUserInput{
		Email:                "direct@example.com",
		Password:             "secret123",
		Username:             "direct",
		AllocatedConcurrency: 5,
		AllocatedRPM:         60,
	}, svc.createInput)
}

func TestAgentManagementHandlerListsChildGroupDelegationOptions(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/children/7/groups", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.listChildGroupsCalls)
	require.Equal(t, int64(42), svc.listChildGroupsActor)
	require.Equal(t, int64(7), svc.listChildGroupsChild)
	require.Contains(t, rec.Body.String(), `"assigned":true`)
	require.Contains(t, rec.Body.String(), `"child_rate_multiplier":2.4`)
	require.Contains(t, rec.Body.String(), `"child_can_delegate":false`)
	require.NotContains(t, rec.Body.String(), `"rate_multiplier":0.3`)
}

func TestAgentManagementHandlerInviteDefaultGroupEndpointsUseAuthenticatedUser(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	listReq := httptest.NewRequest(http.MethodGet, "/invite-default-groups", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	require.Equal(t, http.StatusOK, listRec.Code)
	require.Equal(t, 1, svc.listInviteDefaultGroupsCalls)
	require.Equal(t, int64(42), svc.listInviteDefaultGroupsActor)
	require.Contains(t, listRec.Body.String(), `"assigned":true`)

	putReq := httptest.NewRequest(http.MethodPut, "/invite-default-groups/20", strings.NewReader(`{"rate_multiplier":2.4}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)

	require.Equal(t, http.StatusOK, putRec.Code)
	require.Equal(t, 1, svc.setInviteDefaultGroupCalls)
	require.Equal(t, int64(42), svc.setInviteDefaultGroupActor)
	require.Equal(t, int64(20), svc.setInviteDefaultGroupID)
	require.Equal(t, service.AgentInviteGroupDefaultInput{RateMultiplier: 2.4}, svc.setInviteDefaultGroupInput)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/invite-default-groups/20", nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	require.Equal(t, http.StatusOK, deleteRec.Code)
	require.Equal(t, 1, svc.removeInviteDefaultGroupCalls)
	require.Equal(t, int64(42), svc.removeInviteDefaultGroupActor)
	require.Equal(t, int64(20), svc.removeInviteDefaultGroupID)
}

func TestAgentManagementHandlerRejectsBalanceOnDirectUserCreate(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/direct-users", strings.NewReader(`{
		"email": "direct@example.com",
		"password": "secret123",
		"balance": 100
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, 0, svc.createDirectUserCalls)
}

func TestAgentManagementHandlerUpgradePayload(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/children/7/upgrade", strings.NewReader(`{
		"target_role": "agent_level1",
		"pool_concurrency": 100,
		"pool_rpm": 1000
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.upgradeCalls)
	require.Equal(t, int64(42), svc.upgradeActorID)
	require.Equal(t, int64(7), svc.upgradeChildID)
	require.Equal(t, service.AgentUpgradeInput{
		TargetRole:      service.RoleAgentLevel1,
		PoolConcurrency: 100,
		PoolRPM:         1000,
	}, svc.upgradeInput)
}

func TestAgentManagedUserResponseIncludesAgentProfilePool(t *testing.T) {
	userID := int64(12)
	got := agentManagedUserFromService(&service.User{
		ID:           userID,
		Email:        "agent@example.com",
		Role:         service.RoleAgentLevel2,
		Concurrency:  5,
		RPMLimit:     50,
		AgentProfile: &service.AgentProfile{UserID: userID, PoolConcurrency: 30, PoolRPM: 300, InviteDefaultConcurrency: 2, InviteDefaultRPM: 20},
	})

	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"pool_concurrency":30`)
	require.Contains(t, string(payload), `"pool_rpm":300`)
	require.Contains(t, string(payload), `"invite_default_concurrency":2`)
	require.Contains(t, string(payload), `"invite_default_rpm":20`)
}

func TestAgentManagedUserResponseIncludesEnterpriseProfilePool(t *testing.T) {
	userID := int64(13)
	got := agentManagedUserFromService(&service.User{
		ID:                userID,
		Email:             "enterprise@example.com",
		Role:              service.RoleEnterprise,
		Concurrency:       5,
		RPMLimit:          50,
		EnterpriseProfile: &service.EnterpriseProfile{UserID: userID, PoolConcurrency: 40, PoolRPM: 400},
	})

	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"pool_concurrency":40`)
	require.Contains(t, string(payload), `"pool_rpm":400`)
	require.Contains(t, string(payload), `"invite_default_concurrency":0`)
	require.Contains(t, string(payload), `"invite_default_rpm":0`)
}
