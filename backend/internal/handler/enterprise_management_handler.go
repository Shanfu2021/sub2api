package handler

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type enterpriseManagementService interface {
	ListEmployeesWithQuery(ctx context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error)
	CreateEmployee(ctx context.Context, actorID int64, input service.EmployeeCreateInput) (*service.User, error)
	ImportEmployees(ctx context.Context, actorID int64, input service.EmployeeImportInput) (*service.EmployeeImportResult, error)
	UpdateEmployeeAllocation(ctx context.Context, actorID int64, employeeID int64, input service.EmployeeAllocationUpdate) (*service.AllocationSummary, error)
	InitializeEmployeeBalances(ctx context.Context, actorID int64, input service.EmployeeBalanceInitializationInput) (*service.EmployeeBalanceInitializationResult, error)
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
	params := pagination.DefaultPagination()
	if page, err := strconv.Atoi(strings.TrimSpace(c.Query("page"))); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(strings.TrimSpace(c.Query("page_size"))); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}
	result, err := h.service.ListEmployeesWithQuery(c.Request.Context(), actorID, service.DirectChildrenQuery{
		Pagination: params,
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

func (h *EnterpriseManagementHandler) ImportEmployees(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	req, ok := bindEmployeeImport(c)
	if !ok {
		return
	}
	result, err := h.service.ImportEmployees(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, enterpriseEmployeeImportResultFromService(result))
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

func (h *EnterpriseManagementHandler) InitializeEmployeeBalances(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.EmployeeBalanceInitializationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.InitializeEmployeeBalances(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
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

func bindEmployeeImport(c *gin.Context) (service.EmployeeImportInput, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return service.EmployeeImportInput{}, false
	}
	var records []json.RawMessage
	if err := json.Unmarshal(body, &records); err != nil {
		var wrapper struct {
			Employees []json.RawMessage `json:"employees"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return service.EmployeeImportInput{}, false
		}
		records = wrapper.Employees
	}
	input := service.EmployeeImportInput{Employees: make([]service.EmployeeImportRecord, 0, len(records))}
	for _, raw := range records {
		input.Employees = append(input.Employees, decodeEmployeeImportRecord(raw))
	}
	return input, true
}

func decodeEmployeeImportRecord(raw json.RawMessage) service.EmployeeImportRecord {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return service.EmployeeImportRecord{InvalidFields: []string{"record"}}
	}
	record := service.EmployeeImportRecord{}
	if rawEmail, ok := fields["email"]; ok {
		var value string
		if err := json.Unmarshal(rawEmail, &value); err != nil {
			record.InvalidFields = append(record.InvalidFields, "email")
		} else {
			record.Email = &value
		}
	}
	if rawUsername, ok := fields["username"]; ok {
		var value string
		if err := json.Unmarshal(rawUsername, &value); err != nil {
			record.InvalidFields = append(record.InvalidFields, "username")
		} else {
			record.Username = &value
		}
	}
	if rawPassword, ok := fields["password"]; ok {
		var value string
		if err := json.Unmarshal(rawPassword, &value); err != nil {
			record.InvalidFields = append(record.InvalidFields, "password")
		} else {
			record.Password = &value
		}
	}
	if rawConcurrency, ok := fields["concurrency"]; ok {
		var value int
		if err := json.Unmarshal(rawConcurrency, &value); err != nil {
			record.InvalidFields = append(record.InvalidFields, "concurrency")
		} else {
			record.Concurrency = &value
		}
	}
	if rawRPM, ok := fields["rpm"]; ok {
		var value int
		if err := json.Unmarshal(rawRPM, &value); err != nil {
			record.InvalidFields = append(record.InvalidFields, "rpm")
		} else {
			record.RPM = &value
		}
	}
	return record
}

type enterpriseEmployeeImportResultResponse struct {
	Created      []agentManagedUserResponse   `json:"created"`
	CreatedCount int                          `json:"created_count"`
	Skipped      []service.EmployeeImportSkip `json:"skipped"`
	SkippedCount int                          `json:"skipped_count"`
	Allocation   service.AllocationSummary    `json:"allocation"`
}

func enterpriseEmployeeImportResultFromService(result *service.EmployeeImportResult) enterpriseEmployeeImportResultResponse {
	if result == nil {
		return enterpriseEmployeeImportResultResponse{}
	}
	created := make([]agentManagedUserResponse, 0, len(result.Created))
	for i := range result.Created {
		created = append(created, agentManagedUserFromService(&result.Created[i]))
	}
	return enterpriseEmployeeImportResultResponse{
		Created:      created,
		CreatedCount: result.CreatedCount,
		Skipped:      result.Skipped,
		SkippedCount: result.SkippedCount,
		Allocation:   result.Allocation,
	}
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
