package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type agentManagementService interface {
	ListDirectUsers(ctx context.Context, actorID int64) (*service.DirectChildrenResult, error)
	ListDirectUsersWithQuery(ctx context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error)
	ListDirectAgents(ctx context.Context, actorID int64) (*service.DirectChildrenResult, error)
	ListDirectAgentsWithQuery(ctx context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error)
	ListDirectEnterprises(ctx context.Context, actorID int64) (*service.DirectChildrenResult, error)
	ListDirectEnterprisesWithQuery(ctx context.Context, actorID int64, query service.DirectChildrenQuery) (*service.DirectChildrenResult, error)
	GetAdminAgentTree(ctx context.Context, actorID int64) (*service.AdminAgentTreeResult, error)
	GetSummary(ctx context.Context, actorID int64) (*service.AgentManagementSummary, error)
	CreateDirectUser(ctx context.Context, actorID int64, input service.CreateDirectUserInput) (*service.User, error)
	UpdateAllocation(ctx context.Context, actorID int64, childID int64, req service.AllocationUpdate) (*service.AllocationSummary, error)
	UpdateInviteDefaults(ctx context.Context, actorID int64, input service.AgentInviteDefaultsUpdate) (*service.AgentProfile, error)
	UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, input service.AgentUpgradeInput) (*service.User, error)
	DeleteDirectChild(ctx context.Context, actorID int64, childID int64) error
	ListMyGroups(ctx context.Context, actorID int64) ([]service.AgentGroupRate, error)
	ListChildGroupDelegationOptions(ctx context.Context, actorID int64, childID int64) ([]service.ChildGroupDelegationOption, error)
	SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, input service.ChildGroupDelegationInput) error
	RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error
	ListInviteGroupDefaultOptions(ctx context.Context, actorID int64) ([]service.ChildGroupDelegationOption, error)
	SetInviteGroupDefault(ctx context.Context, actorID int64, groupID int64, input service.AgentInviteGroupDefaultInput) error
	RemoveInviteGroupDefault(ctx context.Context, actorID int64, groupID int64) error
}

type AgentManagementHandler struct {
	service agentManagementService
}

func NewAgentManagementHandler(svc *service.AgentManagementService) *AgentManagementHandler {
	return newAgentManagementHandler(svc)
}

func newAgentManagementHandler(svc agentManagementService) *AgentManagementHandler {
	return &AgentManagementHandler{service: svc}
}

func (h *AgentManagementHandler) Summary(c *gin.Context) {
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

func (h *AgentManagementHandler) ListDirectUsers(c *gin.Context) {
	h.listDirectChildren(c, h.service.ListDirectUsersWithQuery)
}

func (h *AgentManagementHandler) ListDirectAgents(c *gin.Context) {
	h.listDirectChildren(c, h.service.ListDirectAgentsWithQuery)
}

func (h *AgentManagementHandler) ListDirectEnterprises(c *gin.Context) {
	h.listDirectChildren(c, h.service.ListDirectEnterprisesWithQuery)
}

func (h *AgentManagementHandler) AdminAgentTree(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	result, err := h.service.GetAdminAgentTree(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]adminAgentTreeAgentResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, adminAgentTreeAgentFromService(result.Items[i]))
	}
	response.Success(c, gin.H{"items": items})
}

func (h *AgentManagementHandler) CreateDirectUser(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	req, ok := bindCreateDirectUser(c)
	if !ok {
		return
	}
	user, err := h.service.CreateDirectUser(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agentManagedUserFromService(user))
}

func (h *AgentManagementHandler) UpdateAllocation(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	req, ok := bindAllocationUpdate(c)
	if !ok {
		return
	}
	summary, err := h.service.UpdateAllocation(c.Request.Context(), actorID, childID, req)
	if err != nil {
		var reclaimErr *service.AgentPoolReclaimExceededError
		if errors.As(err, &reclaimErr) {
			response.ErrorFrom(c, service.ErrAgentManagementPoolReclaimExceeded.WithMetadata(map[string]string{
				"allocated_concurrency": strconv.Itoa(reclaimErr.AllocatedConcurrency),
				"requested_concurrency": strconv.Itoa(reclaimErr.RequestedConcurrency),
				"allocated_rpm":         strconv.Itoa(reclaimErr.AllocatedRPM),
				"requested_rpm":         strconv.Itoa(reclaimErr.RequestedRPM),
			}))
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

func (h *AgentManagementHandler) UpdateInviteDefaults(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.AgentInviteDefaultsUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	profile, err := h.service.UpdateInviteDefaults(c.Request.Context(), actorID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *AgentManagementHandler) UpgradeDirectUser(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	var req service.AgentUpgradeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	user, err := h.service.UpgradeDirectUser(c.Request.Context(), actorID, childID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agentManagedUserFromService(user))
}

func (h *AgentManagementHandler) DeleteDirectChild(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	if err := h.service.DeleteDirectChild(c.Request.Context(), actorID, childID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"id": childID})
}

func (h *AgentManagementHandler) ListMyGroups(c *gin.Context) {
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

func (h *AgentManagementHandler) ListChildGroupDelegationOptions(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	options, err := h.service.ListChildGroupDelegationOptions(c.Request.Context(), actorID, childID)
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

func (h *AgentManagementHandler) SetChildGroupDelegation(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	var req service.ChildGroupDelegationInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.SetChildGroupDelegation(c.Request.Context(), actorID, childID, groupID, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"child_id": childID, "group_id": groupID})
}

func (h *AgentManagementHandler) RemoveChildGroupDelegation(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	if err := h.service.RemoveChildGroupDelegation(c.Request.Context(), actorID, childID, groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"child_id": childID, "group_id": groupID})
}

func (h *AgentManagementHandler) ListInviteGroupDefaultOptions(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	options, err := h.service.ListInviteGroupDefaultOptions(c.Request.Context(), actorID)
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

func (h *AgentManagementHandler) SetInviteGroupDefault(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	var req service.AgentInviteGroupDefaultInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.SetInviteGroupDefault(c.Request.Context(), actorID, groupID, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID})
}

func (h *AgentManagementHandler) RemoveInviteGroupDefault(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	if err := h.service.RemoveInviteGroupDefault(c.Request.Context(), actorID, groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_id": groupID})
}

func (h *AgentManagementHandler) listDirectChildren(c *gin.Context, list func(context.Context, int64, service.DirectChildrenQuery) (*service.DirectChildrenResult, error)) {
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
	result, err := list(c.Request.Context(), actorID, service.DirectChildrenQuery{
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

func normalizedQuerySearch(search string) string {
	search = strings.TrimSpace(search)
	runes := []rune(search)
	if len(runes) > 100 {
		return string(runes[:100])
	}
	return search
}

func currentActorID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func parsePositiveID(c *gin.Context, name string, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, message)
		return 0, false
	}
	return id, true
}

func bindAllocationUpdate(c *gin.Context) (service.AllocationUpdate, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return service.AllocationUpdate{}, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.AllocationUpdate{}, false
	}
	if _, exists := fields["balance"]; exists {
		response.BadRequest(c, "balance is not managed by agent management")
		return service.AllocationUpdate{}, false
	}
	var req service.AllocationUpdate
	if err := json.Unmarshal(body, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.AllocationUpdate{}, false
	}
	return req, true
}

func bindCreateDirectUser(c *gin.Context) (service.CreateDirectUserInput, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return service.CreateDirectUserInput{}, false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.CreateDirectUserInput{}, false
	}
	if _, exists := fields["balance"]; exists {
		response.BadRequest(c, "balance is not managed by agent management")
		return service.CreateDirectUserInput{}, false
	}
	var req service.CreateDirectUserInput
	if err := json.Unmarshal(body, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return service.CreateDirectUserInput{}, false
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)
	if req.Email == "" || req.Password == "" {
		response.BadRequest(c, "email and password are required")
		return service.CreateDirectUserInput{}, false
	}
	return req, true
}

type agentManagedUserResponse struct {
	ID                       int64   `json:"id"`
	Email                    string  `json:"email"`
	Username                 string  `json:"username"`
	Role                     string  `json:"role"`
	ParentUserID             *int64  `json:"parent_user_id,omitempty"`
	Balance                  float64 `json:"balance"`
	Concurrency              int     `json:"concurrency"`
	RPMLimit                 int     `json:"rpm_limit"`
	AllocatedConcurrency     int     `json:"allocated_concurrency"`
	AllocatedRPM             int     `json:"allocated_rpm"`
	PoolConcurrency          int     `json:"pool_concurrency"`
	PoolRPM                  int     `json:"pool_rpm"`
	InviteDefaultConcurrency int     `json:"invite_default_concurrency"`
	InviteDefaultRPM         int     `json:"invite_default_rpm"`
	AgentIncome              float64 `json:"agent_income"`
	Status                   string  `json:"status"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

type agentGroupRateResponse struct {
	Group         *dto.Group `json:"group"`
	EffectiveRate float64    `json:"effective_rate"`
	CanDelegate   bool       `json:"can_delegate"`
	Source        string     `json:"source"`
}

type childGroupDelegationOptionResponse struct {
	Group               *dto.Group `json:"group"`
	EffectiveRate       float64    `json:"effective_rate"`
	CanDelegate         bool       `json:"can_delegate"`
	Source              string     `json:"source"`
	Assigned            bool       `json:"assigned"`
	ChildRateMultiplier float64    `json:"child_rate_multiplier"`
	ChildCanDelegate    bool       `json:"child_can_delegate"`
}

type adminAgentTreeAgentResponse struct {
	Agent       agentManagedUserResponse           `json:"agent"`
	Users       []agentManagedUserResponse         `json:"users"`
	Enterprises []adminAgentTreeEnterpriseResponse `json:"enterprises"`
}

type adminAgentTreeEnterpriseResponse struct {
	Enterprise agentManagedUserResponse   `json:"enterprise"`
	Employees  []agentManagedUserResponse `json:"employees"`
}

func agentManagedUserFromService(u *service.User) agentManagedUserResponse {
	if u == nil {
		return agentManagedUserResponse{}
	}
	poolConcurrency := 0
	poolRPM := 0
	inviteDefaultConcurrency := 0
	inviteDefaultRPM := 0
	if u.AgentProfile != nil {
		poolConcurrency = u.AgentProfile.PoolConcurrency
		poolRPM = u.AgentProfile.PoolRPM
		inviteDefaultConcurrency = u.AgentProfile.InviteDefaultConcurrency
		inviteDefaultRPM = u.AgentProfile.InviteDefaultRPM
	} else if u.EnterpriseProfile != nil {
		poolConcurrency = u.EnterpriseProfile.PoolConcurrency
		poolRPM = u.EnterpriseProfile.PoolRPM
	}
	return agentManagedUserResponse{
		ID:                       u.ID,
		Email:                    u.Email,
		Username:                 u.Username,
		Role:                     u.Role,
		ParentUserID:             u.ParentUserID,
		Balance:                  u.Balance,
		Concurrency:              u.Concurrency,
		RPMLimit:                 u.RPMLimit,
		AllocatedConcurrency:     u.AllocatedConcurrency,
		AllocatedRPM:             u.AllocatedRPM,
		PoolConcurrency:          poolConcurrency,
		PoolRPM:                  poolRPM,
		InviteDefaultConcurrency: inviteDefaultConcurrency,
		InviteDefaultRPM:         inviteDefaultRPM,
		AgentIncome:              u.AgentIncome,
		Status:                   u.Status,
		CreatedAt:                u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:                u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func adminAgentTreeAgentFromService(in service.AdminAgentTreeAgent) adminAgentTreeAgentResponse {
	users := make([]agentManagedUserResponse, 0, len(in.Users))
	for i := range in.Users {
		users = append(users, agentManagedUserFromService(&in.Users[i]))
	}
	enterprises := make([]adminAgentTreeEnterpriseResponse, 0, len(in.Enterprises))
	for i := range in.Enterprises {
		enterprises = append(enterprises, adminAgentTreeEnterpriseFromService(in.Enterprises[i]))
	}
	return adminAgentTreeAgentResponse{
		Agent:       agentManagedUserFromService(&in.Agent),
		Users:       users,
		Enterprises: enterprises,
	}
}

func adminAgentTreeEnterpriseFromService(in service.AdminAgentTreeEnterprise) adminAgentTreeEnterpriseResponse {
	employees := make([]agentManagedUserResponse, 0, len(in.Employees))
	for i := range in.Employees {
		employees = append(employees, agentManagedUserFromService(&in.Employees[i]))
	}
	return adminAgentTreeEnterpriseResponse{
		Enterprise: agentManagedUserFromService(&in.Enterprise),
		Employees:  employees,
	}
}

func agentGroupRateFromService(in service.AgentGroupRate) agentGroupRateResponse {
	return agentGroupRateResponse{
		Group:         dto.GroupFromServiceShallow(&in.Group),
		EffectiveRate: in.EffectiveRate,
		CanDelegate:   in.CanDelegate,
		Source:        in.Source,
	}
}

func childGroupDelegationOptionFromService(in service.ChildGroupDelegationOption) childGroupDelegationOptionResponse {
	return childGroupDelegationOptionResponse{
		Group:               dto.GroupFromServiceShallow(&in.Group),
		EffectiveRate:       in.EffectiveRate,
		CanDelegate:         in.CanDelegate,
		Source:              in.Source,
		Assigned:            in.Assigned,
		ChildRateMultiplier: in.ChildRateMultiplier,
		ChildCanDelegate:    in.ChildCanDelegate,
	}
}
