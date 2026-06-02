package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeEnterpriseManagementService struct {
	createCalls int
	updateCalls int
	setCalls    int
	removeCalls int

	listActorID   int64
	listQuery     service.DirectChildrenQuery
	createActorID int64
	createInput   service.EmployeeCreateInput
	updateActorID int64
	updateChildID int64
	updateInput   service.EmployeeAllocationUpdate
	setActorID    int64
	setChildID    int64
	setGroupID    int64
	removeActorID int64
	removeChildID int64
	removeGroupID int64
}

func (s *fakeEnterpriseManagementService) ListEmployeesWithQuery(_ context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error) {
	s.listActorID = actorID
	s.listQuery = query
	return &service.DirectChildrenResult{
		Users: []service.User{},
		Pagination: &pagination.PaginationResult{
			Total:    0,
			Page:     query.Pagination.Page,
			PageSize: query.Pagination.Limit(),
			Pages:    0,
		},
	}, nil
}

func (s *fakeEnterpriseManagementService) CreateEmployee(_ context.Context, actorID int64, input service.EmployeeCreateInput) (*service.User, error) {
	s.createCalls++
	s.createActorID = actorID
	s.createInput = input
	return &service.User{
		ID:           99,
		Email:        input.Email,
		Username:     input.Username,
		Role:         service.RoleEmployee,
		ParentUserID: &actorID,
		Balance:      input.Balance,
		Concurrency:  input.Concurrency,
		RPMLimit:     input.RPM,
		Status:       service.StatusActive,
	}, nil
}

func (s *fakeEnterpriseManagementService) UpdateEmployeeAllocation(_ context.Context, actorID int64, employeeID int64, input service.EmployeeAllocationUpdate) (*service.AllocationSummary, error) {
	s.updateCalls++
	s.updateActorID = actorID
	s.updateChildID = employeeID
	s.updateInput = input
	return &service.AllocationSummary{
		TotalConcurrency:     20,
		AllocatedConcurrency: input.Concurrency,
		RemainingConcurrency: 20 - input.Concurrency,
		TotalRPM:             200,
		AllocatedRPM:         input.RPM,
		RemainingRPM:         200 - input.RPM,
	}, nil
}

func (s *fakeEnterpriseManagementService) DeleteEmployee(context.Context, int64, int64) error {
	return nil
}

func (s *fakeEnterpriseManagementService) GetSummary(context.Context, int64) (*service.AgentManagementSummary, error) {
	return &service.AgentManagementSummary{}, nil
}

func (s *fakeEnterpriseManagementService) ListMyGroups(context.Context, int64) ([]service.AgentGroupRate, error) {
	return []service.AgentGroupRate{}, nil
}

func (s *fakeEnterpriseManagementService) ListEmployeeGroupOptions(context.Context, int64, int64) ([]service.ChildGroupDelegationOption, error) {
	return []service.ChildGroupDelegationOption{}, nil
}

func (s *fakeEnterpriseManagementService) SetEmployeeGroup(_ context.Context, actorID int64, employeeID int64, groupID int64, assigned bool) error {
	if assigned {
		s.setCalls++
		s.setActorID = actorID
		s.setChildID = employeeID
		s.setGroupID = groupID
		return nil
	}
	s.removeCalls++
	s.removeActorID = actorID
	s.removeChildID = employeeID
	s.removeGroupID = groupID
	return nil
}

func newEnterpriseManagementHandlerTestRouter(svc *fakeEnterpriseManagementService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := newEnterpriseManagementHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	r.GET("/employees", h.ListEmployees)
	r.POST("/employees", h.CreateEmployee)
	r.PUT("/employees/:id/allocation", h.UpdateEmployeeAllocation)
	r.PUT("/employees/:id/groups/:group_id", h.SetEmployeeGroup)
	r.DELETE("/employees/:id/groups/:group_id", h.RemoveEmployeeGroup)
	return r
}

func TestEnterpriseManagementHandlerListEmployeesUsesPaginationQuery(t *testing.T) {
	svc := &fakeEnterpriseManagementService{}
	router := newEnterpriseManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/employees?page=3&page_size=50&search=employee", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), svc.listActorID)
	require.Equal(t, 3, svc.listQuery.Pagination.Page)
	require.Equal(t, 50, svc.listQuery.Pagination.PageSize)
	require.Equal(t, "employee", svc.listQuery.Search)
	require.Contains(t, rec.Body.String(), `"Page":3`)
}

func TestEnterpriseManagementHandlerCreatesEmployeeWithBalance(t *testing.T) {
	svc := &fakeEnterpriseManagementService{}
	router := newEnterpriseManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/employees", strings.NewReader(`{
		"actor_id": 999,
		"email": "employee@example.com",
		"password": "secret123",
		"username": "employee",
		"balance": 12.5,
		"concurrency": 5,
		"rpm": 60
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.createCalls)
	require.Equal(t, int64(42), svc.createActorID)
	require.Equal(t, service.EmployeeCreateInput{
		Email:       "employee@example.com",
		Password:    "secret123",
		Username:    "employee",
		Balance:     12.5,
		Concurrency: 5,
		RPM:         60,
	}, svc.createInput)
	require.Contains(t, rec.Body.String(), `"balance":12.5`)
}

func TestEnterpriseManagementHandlerUpdatesEmployeeAllocationWithBalance(t *testing.T) {
	svc := &fakeEnterpriseManagementService{}
	router := newEnterpriseManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/employees/7/allocation", strings.NewReader(`{
		"balance": 9.5,
		"concurrency": 3,
		"rpm": 30
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.updateCalls)
	require.Equal(t, int64(42), svc.updateActorID)
	require.Equal(t, int64(7), svc.updateChildID)
	require.Equal(t, service.EmployeeAllocationUpdate{Balance: 9.5, Concurrency: 3, RPM: 30}, svc.updateInput)
}

func TestEnterpriseManagementHandlerRejectsGroupRatePayload(t *testing.T) {
	svc := &fakeEnterpriseManagementService{}
	router := newEnterpriseManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/employees/7/groups/11", strings.NewReader(`{
		"rate_multiplier": 2.0,
		"assigned": true
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, 0, svc.setCalls)
}

func TestEnterpriseManagementHandlerSetsAndRemovesEmployeeGroup(t *testing.T) {
	svc := &fakeEnterpriseManagementService{}
	router := newEnterpriseManagementHandlerTestRouter(svc)

	req := httptest.NewRequest(http.MethodPut, "/employees/7/groups/11", strings.NewReader(`{"assigned": true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.setCalls)
	require.Equal(t, int64(42), svc.setActorID)
	require.Equal(t, int64(7), svc.setChildID)
	require.Equal(t, int64(11), svc.setGroupID)

	req = httptest.NewRequest(http.MethodDelete, "/employees/7/groups/11", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, svc.removeCalls)
	require.Equal(t, int64(42), svc.removeActorID)
	require.Equal(t, int64(7), svc.removeChildID)
	require.Equal(t, int64(11), svc.removeGroupID)
}
