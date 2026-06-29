package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
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
	ListDirectChildrenWithGroupDelegation(ctx context.Context, actorID int64, kind service.DirectChildKind, query service.DirectChildrenGroupQuery) (*service.DirectChildrenResult, error)
	ListDirectChildrenWithoutGroupDelegation(ctx context.Context, actorID int64, kind service.DirectChildKind, query service.DirectChildrenGroupQuery) (*service.DirectChildrenResult, error)
	GetAdminAgentTree(ctx context.Context, actorID int64) (*service.AdminAgentTreeResult, error)
	GetSubordinateStructure(ctx context.Context, actorID int64, ownerID *int64) (*service.SubordinateStructureResult, error)
	ListAgentUsage(ctx context.Context, actorID int64, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error)
	GetAgentUsageStats(ctx context.Context, actorID int64, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error)
	ListAgentUsageUsers(ctx context.Context, actorID int64) ([]service.User, error)
	SearchAgentUsageUsers(ctx context.Context, actorID int64, keyword string, limit int) ([]service.User, error)
	SearchAgentUsageAPIKeys(ctx context.Context, actorID int64, userID int64, keyword string, limit int) ([]service.AgentUsageAPIKeySummary, error)
	SearchAgentUsageAccounts(ctx context.Context, actorID int64, keyword string, limit int) ([]service.AgentUsageAccountSummary, error)
	GetSummary(ctx context.Context, actorID int64) (*service.AgentManagementSummary, error)
	CreateDirectUser(ctx context.Context, actorID int64, input service.CreateDirectUserInput) (*service.User, error)
	UpdateAllocation(ctx context.Context, actorID int64, childID int64, req service.AllocationUpdate) (*service.AllocationSummary, error)
	UpdateChildNotes(ctx context.Context, actorID int64, childID int64, input service.AgentChildNotesUpdate) (*service.User, error)
	UpdateInviteDefaults(ctx context.Context, actorID int64, input service.AgentInviteDefaultsUpdate) (*service.AgentProfile, error)
	UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, input service.AgentUpgradeInput) (*service.User, error)
	DeleteDirectChild(ctx context.Context, actorID int64, childID int64) error
	ListMyGroups(ctx context.Context, actorID int64) ([]service.AgentGroupRate, error)
	ListChildGroupDelegationOptions(ctx context.Context, actorID int64, childID int64) ([]service.ChildGroupDelegationOption, error)
	SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, input service.ChildGroupDelegationInput) error
	SetChildGroupDelegationsBatch(ctx context.Context, actorID int64, childID int64, input service.ChildGroupDelegationBatchInput) error
	SetDirectChildrenGroupDelegationsBatch(ctx context.Context, actorID int64, kind service.DirectChildKind, input service.DirectChildrenGroupDelegationBatchInput) (int, error)
	UpdateDirectChildrenExistingGroupDelegations(ctx context.Context, actorID int64, kind service.DirectChildKind, input service.DirectChildrenGroupDelegationUpdateInput) (*service.DirectChildrenGroupDelegationUpdateResult, error)
	RemoveDirectChildrenGroupDelegationsBatch(ctx context.Context, actorID int64, kind service.DirectChildKind, input service.DirectChildrenGroupDelegationReclaimInput) (*service.DirectChildrenGroupDelegationReclaimResult, error)
	SetAgentIncome(ctx context.Context, actorID int64, childID int64, input service.AgentIncomeSetInput) (*service.User, error)
	RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error
	ListInviteGroupDefaultOptions(ctx context.Context, actorID int64) ([]service.ChildGroupDelegationOption, error)
	SetInviteGroupDefault(ctx context.Context, actorID int64, groupID int64, input service.AgentInviteGroupDefaultInput) error
	SetInviteGroupDefaultsBatch(ctx context.Context, actorID int64, input service.AgentInviteGroupDefaultBatchInput) error
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

func (h *AgentManagementHandler) ListDirectUsersWithGroup(c *gin.Context) {
	h.listDirectChildrenWithGroup(c, service.DirectChildKindUsers)
}

func (h *AgentManagementHandler) ListDirectUsersWithoutGroup(c *gin.Context) {
	h.listDirectChildrenWithoutGroup(c, service.DirectChildKindUsers)
}

func (h *AgentManagementHandler) ListDirectAgentsWithGroup(c *gin.Context) {
	h.listDirectChildrenWithGroup(c, service.DirectChildKindAgents)
}

func (h *AgentManagementHandler) ListDirectAgentsWithoutGroup(c *gin.Context) {
	h.listDirectChildrenWithoutGroup(c, service.DirectChildKindAgents)
}

func (h *AgentManagementHandler) ListDirectEnterprisesWithGroup(c *gin.Context) {
	h.listDirectChildrenWithGroup(c, service.DirectChildKindEnterprises)
}

func (h *AgentManagementHandler) ListDirectEnterprisesWithoutGroup(c *gin.Context) {
	h.listDirectChildrenWithoutGroup(c, service.DirectChildKindEnterprises)
}

func (h *AgentManagementHandler) listDirectChildrenWithGroup(c *gin.Context, kind service.DirectChildKind) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	groupID, err := strconv.ParseInt(strings.TrimSpace(c.Query("group_id")), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group_id")
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListDirectChildrenWithGroupDelegation(c.Request.Context(), actorID, kind, service.DirectChildrenGroupQuery{
		GroupID: groupID,
		Search:  normalizedQuerySearch(c.Query("search")),
		Pagination: pagination.PaginationParams{
			Page:     page,
			PageSize: pageSize,
		},
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

func (h *AgentManagementHandler) listDirectChildrenWithoutGroup(c *gin.Context, kind service.DirectChildKind) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	groupIDs, ok := parseGroupIDsQuery(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.service.ListDirectChildrenWithoutGroupDelegation(c.Request.Context(), actorID, kind, service.DirectChildrenGroupQuery{
		GroupIDs: groupIDs,
		Search:   normalizedQuerySearch(c.Query("search")),
		Pagination: pagination.PaginationParams{
			Page:     page,
			PageSize: pageSize,
		},
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

func (h *AgentManagementHandler) SubordinateStructure(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var ownerID *int64
	if raw := strings.TrimSpace(c.Query("owner_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid owner_id")
			return
		}
		ownerID = &id
	}
	result, err := h.service.GetSubordinateStructure(c.Request.Context(), actorID, ownerID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, subordinateStructureFromService(result))
}

func (h *AgentManagementHandler) ListUsage(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}
	filters, ok := parseAgentUsageFilters(c, false)
	if !ok {
		return
	}
	records, result, err := h.service.ListAgentUsage(c.Request.Context(), actorID, params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.UsageLog, 0, len(records))
	for i := range records {
		item := dto.UsageLogFromService(&records[i])
		if item == nil {
			continue
		}
		item.UpstreamEndpoint = nil
		item.AccountID = 0
		item.APIKeyID = 0
		item.APIKey = nil
		out = append(out, *item)
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

func (h *AgentManagementHandler) UsageStats(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	filters, ok := parseAgentUsageFilters(c, true)
	if !ok {
		return
	}
	stats, err := h.service.GetAgentUsageStats(c.Request.Context(), actorID, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *AgentManagementHandler) ListUsageUsers(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	users, err := h.service.ListAgentUsageUsers(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]agentManagedUserResponse, 0, len(users))
	for i := range users {
		out = append(out, agentManagedUserFromService(&users[i]))
	}
	response.Success(c, out)
}

func (h *AgentManagementHandler) SearchUsageUsers(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	users, err := h.service.SearchAgentUsageUsers(c.Request.Context(), actorID, c.Query("q"), 30)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	type simpleUser struct {
		ID      int64  `json:"id"`
		Email   string `json:"email"`
		Deleted bool   `json:"deleted"`
	}
	out := make([]simpleUser, 0, len(users))
	for i := range users {
		out = append(out, simpleUser{
			ID:      users[i].ID,
			Email:   users[i].Email,
			Deleted: users[i].DeletedAt != nil,
		})
	}
	response.Success(c, out)
}

func (h *AgentManagementHandler) SearchUsageAPIKeys(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var userID int64
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid user_id")
			return
		}
		userID = id
	}
	keys, err := h.service.SearchAgentUsageAPIKeys(c.Request.Context(), actorID, userID, c.Query("q"), 30)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, keys)
}

func (h *AgentManagementHandler) SearchUsageAccounts(c *gin.Context) {
	if _, ok := currentActorID(c); !ok {
		return
	}
	response.Success(c, []service.AgentUsageAccountSummary{})
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

func (h *AgentManagementHandler) UpdateChildNotes(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	var req service.AgentChildNotesUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	user, err := h.service.UpdateChildNotes(c.Request.Context(), actorID, childID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agentManagedUserFromService(user))
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

func (h *AgentManagementHandler) SetChildGroupDelegationsBatch(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	var req service.ChildGroupDelegationBatchInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.SetChildGroupDelegationsBatch(c.Request.Context(), actorID, childID, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"child_id": childID, "group_ids": req.GroupIDs, "all": req.All})
}

func (h *AgentManagementHandler) SetDirectUsersGroupDelegationsBatch(c *gin.Context) {
	h.setDirectChildrenGroupDelegationsBatch(c, service.DirectChildKindUsers)
}

func (h *AgentManagementHandler) SetDirectAgentsGroupDelegationsBatch(c *gin.Context) {
	h.setDirectChildrenGroupDelegationsBatch(c, service.DirectChildKindAgents)
}

func (h *AgentManagementHandler) SetDirectEnterprisesGroupDelegationsBatch(c *gin.Context) {
	h.setDirectChildrenGroupDelegationsBatch(c, service.DirectChildKindEnterprises)
}

func (h *AgentManagementHandler) UpdateDirectUsersExistingGroupDelegations(c *gin.Context) {
	h.updateDirectChildrenExistingGroupDelegations(c, service.DirectChildKindUsers)
}

func (h *AgentManagementHandler) UpdateDirectAgentsExistingGroupDelegations(c *gin.Context) {
	h.updateDirectChildrenExistingGroupDelegations(c, service.DirectChildKindAgents)
}

func (h *AgentManagementHandler) UpdateDirectEnterprisesExistingGroupDelegations(c *gin.Context) {
	h.updateDirectChildrenExistingGroupDelegations(c, service.DirectChildKindEnterprises)
}

func (h *AgentManagementHandler) ReclaimDirectUsersGroupDelegations(c *gin.Context) {
	h.reclaimDirectChildrenGroupDelegations(c, service.DirectChildKindUsers)
}

func (h *AgentManagementHandler) ReclaimDirectAgentsGroupDelegations(c *gin.Context) {
	h.reclaimDirectChildrenGroupDelegations(c, service.DirectChildKindAgents)
}

func (h *AgentManagementHandler) ReclaimDirectEnterprisesGroupDelegations(c *gin.Context) {
	h.reclaimDirectChildrenGroupDelegations(c, service.DirectChildKindEnterprises)
}

func (h *AgentManagementHandler) setDirectChildrenGroupDelegationsBatch(c *gin.Context, kind service.DirectChildKind) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.DirectChildrenGroupDelegationBatchInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	updated, err := h.service.SetDirectChildrenGroupDelegationsBatch(c.Request.Context(), actorID, kind, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	allChildren := true
	if req.AllChildren != nil {
		allChildren = *req.AllChildren
	}
	response.Success(c, gin.H{"kind": kind, "group_ids": req.GroupIDs, "all": req.All, "child_ids": req.ChildIDs, "all_children": allChildren, "updated_children": updated})
}

func (h *AgentManagementHandler) updateDirectChildrenExistingGroupDelegations(c *gin.Context, kind service.DirectChildKind) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.DirectChildrenGroupDelegationUpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.UpdateDirectChildrenExistingGroupDelegations(c.Request.Context(), actorID, kind, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AgentManagementHandler) reclaimDirectChildrenGroupDelegations(c *gin.Context, kind service.DirectChildKind) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.DirectChildrenGroupDelegationReclaimInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.RemoveDirectChildrenGroupDelegationsBatch(c.Request.Context(), actorID, kind, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AgentManagementHandler) SetAgentIncome(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	childID, ok := parsePositiveID(c, "id", "Invalid child ID")
	if !ok {
		return
	}
	var req service.AgentIncomeSetInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	user, err := h.service.SetAgentIncome(c.Request.Context(), actorID, childID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, agentManagedUserFromService(user))
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

func (h *AgentManagementHandler) SetInviteGroupDefaultsBatch(c *gin.Context) {
	actorID, ok := currentActorID(c)
	if !ok {
		return
	}
	var req service.AgentInviteGroupDefaultBatchInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.SetInviteGroupDefaultsBatch(c.Request.Context(), actorID, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"group_ids": req.GroupIDs, "all": req.All})
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

func parseGroupIDsQuery(c *gin.Context) ([]int64, bool) {
	rawValues := make([]string, 0, 2)
	if raw := strings.TrimSpace(c.Query("group_ids")); raw != "" {
		rawValues = append(rawValues, strings.Split(raw, ",")...)
	}
	if raw := strings.TrimSpace(c.Query("group_id")); raw != "" {
		rawValues = append(rawValues, raw)
	}
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(rawValues))
	for _, raw := range rawValues {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_ids")
			return nil, false
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		response.BadRequest(c, "Invalid group_ids")
		return nil, false
	}
	return out, true
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

func parseAgentUsageFilters(c *gin.Context, withDefaultPeriod bool) (usagestats.UsageLogFilters, bool) {
	var userID, apiKeyID, accountID, groupID int64
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid user_id")
			return usagestats.UsageLogFilters{}, false
		}
		userID = id
	}
	if raw := strings.TrimSpace(c.Query("api_key_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid api_key_id")
			return usagestats.UsageLogFilters{}, false
		}
		apiKeyID = id
	}
	if raw := strings.TrimSpace(c.Query("account_id")); raw != "" {
		response.BadRequest(c, "account_id filter is not available")
		return usagestats.UsageLogFilters{}, false
	}
	if raw := strings.TrimSpace(c.Query("group_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return usagestats.UsageLogFilters{}, false
		}
		groupID = id
	}

	var requestType *int16
	var stream *bool
	if raw := strings.TrimSpace(c.Query("request_type")); raw != "" {
		parsed, err := service.ParseUsageRequestType(raw)
		if err != nil {
			response.BadRequest(c, err.Error())
			return usagestats.UsageLogFilters{}, false
		}
		value := int16(parsed)
		requestType = &value
	} else if raw := strings.TrimSpace(c.Query("stream")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "Invalid stream value, use true or false")
			return usagestats.UsageLogFilters{}, false
		}
		stream = &parsed
	}

	var billingType *int8
	if raw := strings.TrimSpace(c.Query("billing_type")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			response.BadRequest(c, "Invalid billing_type")
			return usagestats.UsageLogFilters{}, false
		}
		value := int8(parsed)
		billingType = &value
	}

	startTime, endTime, ok := parseAgentUsageTimeRange(c, withDefaultPeriod)
	if !ok {
		return usagestats.UsageLogFilters{}, false
	}

	return usagestats.UsageLogFilters{
		UserID:      userID,
		APIKeyID:    apiKeyID,
		AccountID:   accountID,
		GroupID:     groupID,
		Model:       c.Query("model"),
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
		BillingMode: strings.TrimSpace(c.Query("billing_mode")),
		StartTime:   startTime,
		EndTime:     endTime,
	}, true
}

func parseAgentUsageTimeRange(c *gin.Context, withDefaultPeriod bool) (*time.Time, *time.Time, bool) {
	userTZ := c.Query("timezone")
	startRaw := strings.TrimSpace(c.Query("start_date"))
	endRaw := strings.TrimSpace(c.Query("end_date"))
	var startTime, endTime *time.Time
	if startRaw != "" {
		parsed, err := timezone.ParseInUserLocation("2006-01-02", startRaw, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return nil, nil, false
		}
		startTime = &parsed
	}
	if endRaw != "" {
		parsed, err := timezone.ParseInUserLocation("2006-01-02", endRaw, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return nil, nil, false
		}
		parsed = parsed.AddDate(0, 0, 1)
		endTime = &parsed
	}
	if !withDefaultPeriod || startTime != nil || endTime != nil {
		return startTime, endTime, true
	}

	now := timezone.NowInUserLocation(userTZ)
	var start time.Time
	switch c.DefaultQuery("period", "today") {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	default:
		start = timezone.StartOfDayInUserLocation(now, userTZ)
	}
	return &start, &now, true
}

type agentManagedUserResponse struct {
	ID                       int64             `json:"id"`
	Email                    string            `json:"email"`
	Username                 string            `json:"username"`
	Role                     string            `json:"role"`
	ParentUserID             *int64            `json:"parent_user_id,omitempty"`
	Balance                  float64           `json:"balance"`
	Concurrency              int               `json:"concurrency"`
	RPMLimit                 int               `json:"rpm_limit"`
	AllocatedConcurrency     int               `json:"allocated_concurrency"`
	AllocatedRPM             int               `json:"allocated_rpm"`
	PoolConcurrency          int               `json:"pool_concurrency"`
	PoolRPM                  int               `json:"pool_rpm"`
	InviteDefaultConcurrency int               `json:"invite_default_concurrency"`
	InviteDefaultRPM         int               `json:"invite_default_rpm"`
	AgentIncome              float64           `json:"agent_income"`
	Notes                    string            `json:"notes"`
	Status                   string            `json:"status"`
	GroupRates               map[int64]float64 `json:"group_rates,omitempty"`
	CreatedAt                string            `json:"created_at"`
	UpdatedAt                string            `json:"updated_at"`
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

type subordinateStructureResponse struct {
	OwnerOptions  []agentManagedUserResponse         `json:"owner_options"`
	SelectedOwner agentManagedUserResponse           `json:"selected_owner"`
	Users         []agentManagedUserResponse         `json:"users"`
	Enterprises   []adminAgentTreeEnterpriseResponse `json:"enterprises"`
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
		Notes:                    u.Notes,
		Status:                   u.Status,
		GroupRates:               u.GroupRates,
		CreatedAt:                u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:                u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func subordinateStructureFromService(in *service.SubordinateStructureResult) subordinateStructureResponse {
	if in == nil {
		return subordinateStructureResponse{}
	}
	ownerOptions := make([]agentManagedUserResponse, 0, len(in.OwnerOptions))
	for i := range in.OwnerOptions {
		ownerOptions = append(ownerOptions, agentManagedUserFromService(&in.OwnerOptions[i]))
	}
	users := make([]agentManagedUserResponse, 0, len(in.Users))
	for i := range in.Users {
		users = append(users, agentManagedUserFromService(&in.Users[i]))
	}
	enterprises := make([]adminAgentTreeEnterpriseResponse, 0, len(in.Enterprises))
	for i := range in.Enterprises {
		enterprises = append(enterprises, adminAgentTreeEnterpriseFromService(in.Enterprises[i]))
	}
	return subordinateStructureResponse{
		OwnerOptions:  ownerOptions,
		SelectedOwner: agentManagedUserFromService(&in.SelectedOwner),
		Users:         users,
		Enterprises:   enterprises,
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
