package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type EnterpriseHandler struct {
	enterpriseService *service.EnterpriseService
}

func NewEnterpriseHandler(enterpriseService *service.EnterpriseService) *EnterpriseHandler {
	return &EnterpriseHandler{enterpriseService: enterpriseService}
}

type createEnterpriseTenantRequest struct {
	Name               string  `json:"name" binding:"required"`
	Code               string  `json:"code"`
	Status             string  `json:"status"`
	Notes              string  `json:"notes"`
	PortalHost         string  `json:"portal_host"`
	PricingFloorFactor float64 `json:"pricing_floor_factor"`
	PricingScope       string  `json:"pricing_scope"`
	AllowedGroupIDs    []int64 `json:"allowed_group_ids"`
}

type updateEnterpriseTenantRequest struct {
	Name               *string  `json:"name"`
	Status             *string  `json:"status"`
	Notes              *string  `json:"notes"`
	PortalHost         *string  `json:"portal_host"`
	PricingFloorFactor *float64 `json:"pricing_floor_factor"`
	PricingScope       *string  `json:"pricing_scope"`
	AllowedGroupIDs    *[]int64 `json:"allowed_group_ids"`
}

type adjustEnterpriseQuotaRequest struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Direction string  `json:"direction" binding:"required"`
	Notes     string  `json:"notes"`
}

type bindEnterpriseMemberRequest struct {
	UserID        int64   `json:"user_id" binding:"required,gt=0"`
	MemberRole    string  `json:"member_role"`
	MemberNote    string  `json:"member_note"`
	PricingFactor float64 `json:"pricing_factor"`
	PricingScope  string  `json:"pricing_scope"`
	JoinedVia     string  `json:"joined_via"`
	JoinedSource  string  `json:"joined_source"`
}

type updateEnterpriseMemberRequest struct {
	MemberRole    *string  `json:"member_role"`
	MemberNote    *string  `json:"member_note"`
	PricingFactor *float64 `json:"pricing_factor"`
	PricingScope  *string  `json:"pricing_scope"`
	Status        *string  `json:"status"`
	AllowedGroups *[]int64 `json:"allowed_groups"`
}

type createEnterpriseInviteCodeRequest struct {
	Code      string  `json:"code"`
	MaxUses   int     `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"`
	Notes     string  `json:"notes"`
}

type updateEnterpriseInviteCodeRequest struct {
	Status    *string `json:"status"`
	MaxUses   *int    `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"`
	Notes     *string `json:"notes"`
}

func (h *EnterpriseHandler) ListTenants(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.enterpriseService.ListTenants(c.Request.Context(), page, pageSize, service.EnterpriseTenantListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		Search: strings.TrimSpace(c.Query("search")),
	}, c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *EnterpriseHandler) CreateTenant(c *gin.Context) {
	var req createEnterpriseTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.CreateTenant(c.Request.Context(), actor.UserID, service.CreateEnterpriseTenantInput{
		Name:               req.Name,
		Code:               req.Code,
		Status:             req.Status,
		Notes:              req.Notes,
		PortalHost:         req.PortalHost,
		PricingFloorFactor: req.PricingFloorFactor,
		PricingScope:       req.PricingScope,
		AllowedGroupIDs:    req.AllowedGroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) GetTenant(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	item, err := h.enterpriseService.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) UpdateTenant(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req updateEnterpriseTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.UpdateTenant(c.Request.Context(), actor.UserID, tenantID, service.UpdateEnterpriseTenantInput{
		Name:               req.Name,
		Status:             req.Status,
		Notes:              req.Notes,
		PortalHost:         req.PortalHost,
		PricingFloorFactor: req.PricingFloorFactor,
		PricingScope:       req.PricingScope,
		AllowedGroupIDs:    req.AllowedGroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) AdjustQuota(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req adjustEnterpriseQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.AdjustTenantQuota(c.Request.Context(), actor.UserID, tenantID, service.AdjustEnterpriseQuotaInput{
		Amount:    req.Amount,
		Direction: req.Direction,
		Notes:     req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) ListMembers(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.enterpriseService.ListTenantMembers(c.Request.Context(), tenantID, page, pageSize, service.EnterpriseMemberListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		Role:   strings.TrimSpace(c.Query("role")),
		Search: strings.TrimSpace(c.Query("search")),
	}, c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *EnterpriseHandler) BindMember(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req bindEnterpriseMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.BindUserToTenant(c.Request.Context(), actor.UserID, tenantID, service.BindEnterpriseMemberInput{
		UserID:        req.UserID,
		MemberRole:    req.MemberRole,
		MemberNote:    req.MemberNote,
		PricingFactor: req.PricingFactor,
		PricingScope:  req.PricingScope,
		JoinedVia:     req.JoinedVia,
		JoinedSource:  req.JoinedSource,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) UpdateMember(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	userID, ok := parseInt64Param(c, "user_id")
	if !ok {
		return
	}
	var req updateEnterpriseMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.UpdateTenantMember(c.Request.Context(), actor.UserID, tenantID, userID, service.UpdateEnterpriseMemberInput{
		MemberRole:    req.MemberRole,
		MemberNote:    req.MemberNote,
		PricingFactor: req.PricingFactor,
		PricingScope:  req.PricingScope,
		Status:        req.Status,
		AllowedGroups: req.AllowedGroups,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) DeleteMember(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	userID, ok := parseInt64Param(c, "user_id")
	if !ok {
		return
	}
	if err := h.enterpriseService.RemoveTenantMember(c.Request.Context(), tenantID, userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *EnterpriseHandler) ListInviteCodes(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.enterpriseService.ListTenantInviteCodes(c.Request.Context(), tenantID, page, pageSize, service.EnterpriseInviteCodeListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		Search: strings.TrimSpace(c.Query("search")),
	}, c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *EnterpriseHandler) CreateInviteCode(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req createEnterpriseInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		response.BadRequest(c, "Invalid expires_at")
		return
	}
	actor, _ := middleware.GetAuthSubjectFromContext(c)
	item, err := h.enterpriseService.CreateTenantInviteCode(c.Request.Context(), actor.UserID, tenantID, service.CreateEnterpriseInviteCodeInput{
		Code:      req.Code,
		MaxUses:   req.MaxUses,
		ExpiresAt: expiresAt,
		Notes:     req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) UpdateInviteCode(c *gin.Context) {
	inviteID, ok := parseInt64Param(c, "invite_id")
	if !ok {
		return
	}
	var req updateEnterpriseInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		response.BadRequest(c, "Invalid expires_at")
		return
	}
	item, err := h.enterpriseService.UpdateTenantInviteCode(c.Request.Context(), inviteID, service.UpdateEnterpriseInviteCodeInput{
		Status:    req.Status,
		MaxUses:   req.MaxUses,
		ExpiresAt: expiresAt,
		Notes:     req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) ListLedger(c *gin.Context) {
	tenantID, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.enterpriseService.ListLedger(c.Request.Context(), tenantID, page, pageSize, c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func parseInt64Param(c *gin.Context, name string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || v <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return v, true
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, err
	}
	return &t, nil
}
