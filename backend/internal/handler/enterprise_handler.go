package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type EnterpriseHandler struct {
	enterpriseService *service.EnterpriseService
}

func NewEnterpriseHandler(enterpriseService *service.EnterpriseService) *EnterpriseHandler {
	return &EnterpriseHandler{enterpriseService: enterpriseService}
}

type enterpriseCreateMemberRequest struct {
	Email          string  `json:"email" binding:"required,email"`
	Password       string  `json:"password" binding:"required,min=6"`
	Username       string  `json:"username"`
	Notes          string  `json:"notes"`
	Concurrency    int     `json:"concurrency"`
	RPMLimit       *int    `json:"rpm_limit"`
	AllowedGroups  []int64            `json:"allowed_groups"`
	MemberNote     string             `json:"member_note"`
	PricingFactor  float64            `json:"pricing_factor"`
	PricingScope   string             `json:"pricing_scope"`
	GroupRates     map[int64]*float64 `json:"group_rates"`
	InitialBalance float64            `json:"initial_balance"`
}

type enterpriseUpdateMemberRequest struct {
	MemberRole    *string            `json:"member_role"`
	MemberNote    *string            `json:"member_note"`
	PricingFactor *float64           `json:"pricing_factor"`
	PricingScope  *string            `json:"pricing_scope"`
	Concurrency   *int               `json:"concurrency"`
	Status        *string            `json:"status"`
	AllowedGroups *[]int64           `json:"allowed_groups"`
	GroupRates    map[int64]*float64 `json:"group_rates"`
}

type enterpriseUpdatePricingDefaultsRequest struct {
	MemberDefaultPricingFactor *float64           `json:"member_default_pricing_factor"`
	MemberGroupRates           map[int64]*float64 `json:"member_group_rates"`
}

type enterpriseAdjustBalanceRequest struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Operation string  `json:"operation"`
	Notes     string  `json:"notes"`
}

type enterpriseBindInviteCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

type enterpriseCreateInviteCodeRequest struct {
	Code      string  `json:"code"`
	MaxUses   int     `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"`
	Notes     string  `json:"notes"`
}

type enterpriseUpdateInviteCodeRequest struct {
	Status    *string `json:"status"`
	MaxUses   *int    `json:"max_uses"`
	ExpiresAt *string `json:"expires_at"`
	Notes     *string `json:"notes"`
}

func (h *EnterpriseHandler) GetMe(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	enterprise, err := h.enterpriseService.GetUserEnterpriseContext(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if enterprise == nil {
		response.Success(c, gin.H{"enterprise": nil})
		return
	}
	tenant, err := h.enterpriseService.GetTenant(c.Request.Context(), enterprise.TenantID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"enterprise": enterprise,
		"tenant":     tenant,
	})
}

func (h *EnterpriseHandler) BindInviteCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req enterpriseBindInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.enterpriseService.BindCurrentUserByInviteCode(c.Request.Context(), subject.UserID, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) ListMembers(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, _, err := h.enterpriseService.ListMyMembers(c.Request.Context(), subject.UserID, page, pageSize, service.EnterpriseMemberListFilters{
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

func (h *EnterpriseHandler) ListGroups(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	items, _, err := h.enterpriseService.ListMyGroupSummaries(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *EnterpriseHandler) CreateMember(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req enterpriseCreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	member, user, err := h.enterpriseService.CreateMemberByManager(c.Request.Context(), subject.UserID, service.CreateEnterpriseMemberUserInput{
		Email:          req.Email,
		Password:       req.Password,
		Username:       req.Username,
		Notes:          req.Notes,
		Concurrency:    req.Concurrency,
		RPMLimit:       req.RPMLimit,
		AllowedGroups:  req.AllowedGroups,
		MemberNote:     req.MemberNote,
		PricingFactor:  req.PricingFactor,
		PricingScope:   req.PricingScope,
		GroupRates:     req.GroupRates,
		InitialBalance: req.InitialBalance,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"membership": member,
		"user":       dto.UserFromService(user),
	})
}

func (h *EnterpriseHandler) UpdateMember(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID, ok := parseEnterpriseUserID(c)
	if !ok {
		return
	}
	var req enterpriseUpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.enterpriseService.UpdateMemberByManager(c.Request.Context(), subject.UserID, userID, service.UpdateEnterpriseMemberInput{
		MemberRole:    req.MemberRole,
		MemberNote:    req.MemberNote,
		PricingFactor: req.PricingFactor,
		PricingScope:  req.PricingScope,
		Concurrency:   req.Concurrency,
		Status:        req.Status,
		AllowedGroups: req.AllowedGroups,
		GroupRates:    req.GroupRates,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *EnterpriseHandler) UpdatePricingDefaults(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req enterpriseUpdatePricingDefaultsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tenant, err := h.enterpriseService.UpdateMyPricingDefaults(c.Request.Context(), subject.UserID, service.UpdateEnterpriseManagerPricingDefaultsInput{
		MemberDefaultPricingFactor: req.MemberDefaultPricingFactor,
		MemberGroupRates:           req.MemberGroupRates,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tenant)
}

func (h *EnterpriseHandler) AdjustMemberBalance(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	userID, ok := parseEnterpriseUserID(c)
	if !ok {
		return
	}
	var req enterpriseAdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	member, user, err := h.enterpriseService.AdjustMemberBalanceByManager(c.Request.Context(), subject.UserID, userID, service.AdjustEnterpriseMemberBalanceInput{
		Amount:    req.Amount,
		Operation: req.Operation,
		Notes:     req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"membership": member,
		"user":       dto.UserFromService(user),
	})
}

func (h *EnterpriseHandler) ListInviteCodes(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, _, err := h.enterpriseService.ListMyInviteCodes(c.Request.Context(), subject.UserID, page, pageSize, service.EnterpriseInviteCodeListFilters{
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req enterpriseCreateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	expiresAt, err := parseEnterpriseOptionalTime(req.ExpiresAt)
	if err != nil {
		response.BadRequest(c, "Invalid expires_at")
		return
	}
	item, _, err := h.enterpriseService.CreateMyInviteCode(c.Request.Context(), subject.UserID, service.CreateEnterpriseInviteCodeInput{
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	inviteID, ok := parseEnterpriseUserIDParam(c, "invite_id")
	if !ok {
		return
	}
	var req enterpriseUpdateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	expiresAt, err := parseEnterpriseOptionalTime(req.ExpiresAt)
	if err != nil {
		response.BadRequest(c, "Invalid expires_at")
		return
	}
	item, _, err := h.enterpriseService.UpdateMyInviteCode(c.Request.Context(), subject.UserID, inviteID, service.UpdateEnterpriseInviteCodeInput{
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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, _, err := h.enterpriseService.ListMyLedger(c.Request.Context(), subject.UserID, page, pageSize, c.DefaultQuery("sort_by", "created_at"), c.DefaultQuery("sort_order", "desc"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func parseEnterpriseUserID(c *gin.Context) (int64, bool) {
	return parseEnterpriseUserIDParam(c, "user_id")
}

func parseEnterpriseUserIDParam(c *gin.Context, key string) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(c.Param(key)), 10, 64)
	if err != nil || v <= 0 {
		response.BadRequest(c, "Invalid "+key)
		return 0, false
	}
	return v, true
}

func parseEnterpriseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, err
	}
	return &t, nil
}
