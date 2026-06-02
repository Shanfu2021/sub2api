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
	GetSummary(ctx context.Context, actorID int64) (*service.AgentManagementSummary, error)
	CreateDirectUser(ctx context.Context, actorID int64, input service.CreateDirectUserInput) (*service.User, error)
	UpdateAllocation(ctx context.Context, actorID int64, childID int64, req service.AllocationUpdate) (*service.AllocationSummary, error)
	UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, input service.AgentUpgradeInput) (*service.User, error)
	DeleteDirectChild(ctx context.Context, actorID int64, childID int64) error
	ListMyGroups(ctx context.Context, actorID int64) ([]service.AgentGroupRate, error)
	SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, input service.ChildGroupDelegationInput) error
	RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error
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

func (h *AgentManagementHandler) listDirectChildren(c *gin.Context, list func(context.Context, int64, service.DirectChildrenQuery) (*service.DirectChildrenResult, error)) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	result, err := list(c.Request.Context(), actorID, service.DirectChildrenQuery{
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
	ID                   int64   `json:"id"`
	Email                string  `json:"email"`
	Username             string  `json:"username"`
	Role                 string  `json:"role"`
	ParentUserID         *int64  `json:"parent_user_id,omitempty"`
	Balance              float64 `json:"balance"`
	Concurrency          int     `json:"concurrency"`
	RPMLimit             int     `json:"rpm_limit"`
	AllocatedConcurrency int     `json:"allocated_concurrency"`
	AllocatedRPM         int     `json:"allocated_rpm"`
	PoolConcurrency      int     `json:"pool_concurrency"`
	PoolRPM              int     `json:"pool_rpm"`
	Status               string  `json:"status"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type agentGroupRateResponse struct {
	Group         *dto.Group `json:"group"`
	EffectiveRate float64    `json:"effective_rate"`
	CanDelegate   bool       `json:"can_delegate"`
	Source        string     `json:"source"`
}

func agentManagedUserFromService(u *service.User) agentManagedUserResponse {
	if u == nil {
		return agentManagedUserResponse{}
	}
	poolConcurrency := 0
	poolRPM := 0
	if u.AgentProfile != nil {
		poolConcurrency = u.AgentProfile.PoolConcurrency
		poolRPM = u.AgentProfile.PoolRPM
	}
	return agentManagedUserResponse{
		ID:                   u.ID,
		Email:                u.Email,
		Username:             u.Username,
		Role:                 u.Role,
		ParentUserID:         u.ParentUserID,
		Balance:              u.Balance,
		Concurrency:          u.Concurrency,
		RPMLimit:             u.RPMLimit,
		AllocatedConcurrency: u.AllocatedConcurrency,
		AllocatedRPM:         u.AllocatedRPM,
		PoolConcurrency:      poolConcurrency,
		PoolRPM:              poolRPM,
		Status:               u.Status,
		CreatedAt:            u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
