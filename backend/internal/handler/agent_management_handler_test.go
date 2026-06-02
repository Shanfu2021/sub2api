package handler

import (
	"context"
	"encoding/json"
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
	updateAllocationCalls int
	upgradeCalls          int
	createDirectUserCalls int
	createActorID         int64
	createInput           service.CreateDirectUserInput
	updateActorID         int64
	updateChildID         int64
	updateAllocation      service.AllocationUpdate
	upgradeActorID        int64
	upgradeChildID        int64
	upgradeInput          service.AgentUpgradeInput
}

func (s *fakeAgentManagementService) ListDirectUsers(context.Context, int64) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectAgents(context.Context, int64) (*service.DirectChildrenResult, error) {
	return &service.DirectChildrenResult{}, nil
}

func (s *fakeAgentManagementService) ListDirectEnterprises(context.Context, int64) (*service.DirectChildrenResult, error) {
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
	return &service.AllocationSummary{
		TotalConcurrency:     20,
		AllocatedConcurrency: concurrency,
		RemainingConcurrency: 20 - concurrency,
		TotalRPM:             200,
		AllocatedRPM:         rpm,
		RemainingRPM:         200 - rpm,
	}, nil
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

func (s *fakeAgentManagementService) SetChildGroupDelegation(context.Context, int64, int64, int64, service.ChildGroupDelegationInput) error {
	return nil
}

func (s *fakeAgentManagementService) RemoveChildGroupDelegation(context.Context, int64, int64, int64) error {
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
	r.PUT("/children/:id/allocation", h.UpdateAllocation)
	r.POST("/children/:id/upgrade", h.UpgradeDirectUser)
	r.POST("/direct-users", h.CreateDirectUser)
	return r
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
		AgentProfile: &service.AgentProfile{UserID: userID, PoolConcurrency: 30, PoolRPM: 300},
	})

	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"pool_concurrency":30`)
	require.Contains(t, string(payload), `"pool_rpm":300`)
}
