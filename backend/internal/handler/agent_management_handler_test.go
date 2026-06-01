package handler

import (
	"context"
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
	updateActorID         int64
	updateChildID         int64
	updateAllocation      service.AllocationUpdate
	upgradeActorID        int64
	upgradeChildID        int64
	upgradeTargetRole     string
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

func (s *fakeAgentManagementService) UpdateAllocation(_ context.Context, actorID int64, childID int64, req service.AllocationUpdate) (*service.AllocationSummary, error) {
	s.updateAllocationCalls++
	s.updateActorID = actorID
	s.updateChildID = childID
	s.updateAllocation = req
	return &service.AllocationSummary{
		TotalConcurrency:     20,
		AllocatedConcurrency: req.AllocatedConcurrency,
		RemainingConcurrency: 20 - req.AllocatedConcurrency,
		TotalRPM:             200,
		AllocatedRPM:         req.AllocatedRPM,
		RemainingRPM:         200 - req.AllocatedRPM,
	}, nil
}

func (s *fakeAgentManagementService) UpgradeDirectUser(_ context.Context, actorID int64, childID int64, targetRole string) (*service.User, error) {
	s.upgradeCalls++
	s.upgradeActorID = actorID
	s.upgradeChildID = childID
	s.upgradeTargetRole = targetRole
	return &service.User{ID: childID, Role: targetRole}, nil
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
		"allocated_concurrency": 5,
		"allocated_rpm": 60
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.updateAllocationCalls)
	require.Equal(t, int64(42), svc.updateActorID)
	require.Equal(t, int64(7), svc.updateChildID)
	require.Equal(t, service.AllocationUpdate{AllocatedConcurrency: 5, AllocatedRPM: 60}, svc.updateAllocation)
}

func TestAgentManagementHandlerUpgradePayload(t *testing.T) {
	svc := &fakeAgentManagementService{}
	router := newAgentManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/children/7/upgrade", strings.NewReader(`{"target_role":"agent_level1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.upgradeCalls)
	require.Equal(t, int64(42), svc.upgradeActorID)
	require.Equal(t, int64(7), svc.upgradeChildID)
	require.Equal(t, service.RoleAgentLevel1, svc.upgradeTargetRole)
}
