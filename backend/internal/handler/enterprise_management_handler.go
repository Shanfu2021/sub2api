package handler

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type enterpriseManagementService interface {
	ListEmployeesWithQuery(ctx context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error)
	CreateEmployee(ctx context.Context, actorID int64, input service.EmployeeCreateInput) (*service.User, error)
	UpdateEmployeeAllocation(ctx context.Context, actorID int64, employeeID int64, input service.EmployeeAllocationUpdate) (*service.AllocationSummary, error)
	DeleteEmployee(ctx context.Context, actorID int64, employeeID int64) error
	GetSummary(ctx context.Context, actorID int64) (*service.AgentManagementSummary, error)
	ListMyGroups(ctx context.Context, actorID int64) ([]service.AgentGroupRate, error)
	ListEmployeeGroupOptions(ctx context.Context, actorID int64, employeeID int64) ([]service.ChildGroupDelegationOption, error)
	SetEmployeeGroup(ctx context.Context, actorID int64, employeeID int64, groupID int64, assigned bool) error
}

type EnterpriseManagementHandler struct {
	service enterpriseManagementService
}

func NewEnterpriseManagementHandler(svc *service.EnterpriseManagementService) *EnterpriseManagementHandler {
	return newEnterpriseManagementHandler(svc)
}

func newEnterpriseManagementHandler(svc enterpriseManagementService) *EnterpriseManagementHandler {
	return &EnterpriseManagementHandler{service: svc}
}

func (h *EnterpriseManagementHandler) Summary(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	summary, err := h.service.GetSummary(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *EnterpriseManagementHandler) ListEmployees(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	result, err := h.service.ListEmployeesWithQuery(c.Request.Context(), actorID, service.DirectChildrenQuery{
		Pagination: pagination.DefaultPagination(),
		Search:     normalizedQuerySearch(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	users := make([]agentManagedUserResponse, 0, len(result.Users))
	for i := range result.Users {
		users = append(users, agentManagedUserFromService(&result.Users[i]))
	}
	response.Success(c, gin.H{
		"items":      users,
		"pagination": result.Pagination,
	})
}

func (h *EnterpriseManagementHandler) CreateEmployee(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	req, ok := bindEmployeeCreate(c)
	if !ok {
		return
	}
	user, err := h.service.CreateEmployee(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agentManagedUserFromService(user))
}

func (h *EnterpriseManagementHandler) UpdateEmployeeAllocation(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	employeeID, ok := parsePositiveID(c, "id", "Invalid employee ID")
	if !ok {
		return
	}
	req, ok := bindEmployeeAllocationUpdate(c)
	if !ok {
		return
	}
	summary, err := h.service.UpdateEmployeeAllocation(c.Request.Context(), actorID, employeeID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *EnterpriseManagementHandler) DeleteEmployee(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	employeeID, ok := parsePositiveID(c, "id", "Invalid employee ID")
	if !ok {
		return
	}
	if err := h.service.DeleteEmployee(c.Request.Context(), actorID, employeeID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": employeeID})
}

func (h *EnterpriseManagementHandler) ListMyGroups(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	groups, err := h.service.ListMyGroups(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]agentGroupRateResponse, 0, len(groups))
	for i := range groups {
		out = append(out, agentGroupRateFromService(groups[i]))
	}
	response.Success(c, out)
}

func (h *EnterpriseManagementHandler) ListEmployeeGroupOptions(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	employeeID, ok := parsePositiveID(c, "id", "Invalid employee ID")
	if !ok {
		return
	}
	options, err := h.service.ListEmployeeGroupOptions(c.Request.Context(), actorID, employeeID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]childGroupDelegationOptionResponse, 0, len(options))
	for i := range options {
		out = append(out, childGroupDelegationOptionFromService(options[i]))
	}
	response.Success(c, out)
}

func (h *EnterpriseManagementHandler) SetEmployeeGroup(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	employeeID, ok := parsePositiveID(c, "id", "Invalid employee ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	assigned, ok := bindEmployeeGroupAssignment(c)
	if !ok {
		return
	}
	if err := h.service.SetEmployeeGroup(c.Request.Context(), actorID, employeeID, groupID, assigned); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"employee_id": employeeID, "group_id": groupID})
}

func (h *EnterpriseManagementHandler) RemoveEmployeeGroup(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	employeeID, ok := parsePositiveID(c, "id", "Invalid employee ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	if err := h.service.SetEmployeeGroup(c.Request.Context(), actorID, employeeID, groupID, false); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"employee_id": employeeID, "group_id": groupID})
}

func bindEmployeeCreate(c *gin.Context) (service.EmployeeCreateInput, bool) {
	var req service.EmployeeCreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.EmployeeCreateInput{}, false
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)
	if req.Email == "" || req.Password == "" {
		response.BadRequest(c, "email and password are required")
		return service.EmployeeCreateInput{}, false
	}
	return req, true
}

func bindEmployeeAllocationUpdate(c *gin.Context) (service.EmployeeAllocationUpdate, bool) {
	var req service.EmployeeAllocationUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.EmployeeAllocationUpdate{}, false
	}
	return req, true
}

func bindEmployeeGroupAssignment(c *gin.Context) (bool, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return false, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return false, false
	}
	if _, exists := fields["rate_multiplier"]; exists {
		response.BadRequest(c, "enterprise cannot reprice employee groups")
		return false, false
	}
	if _, exists := fields["can_delegate"]; exists {
		response.BadRequest(c, "enterprise cannot delegate group propagation")
		return false, false
	}
	var req struct {
		Assigned bool `json:"assigned"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return false, false
	}
	return req.Assigned, true
}
