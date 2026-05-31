package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	EnterpriseTenantStatusActive   = "active"
	EnterpriseTenantStatusDisabled = "disabled"

	EnterpriseMemberRoleManager = "manager"
	EnterpriseMemberRoleMember  = "member"

	EnterpriseJoinViaInviteCode  = "invite_code"
	EnterpriseJoinViaAdminCreate = "admin_create"
	EnterpriseJoinViaManualBind  = "manual_bind"

	EnterpriseInviteStatusActive   = "active"
	EnterpriseInviteStatusDisabled = "disabled"

	EnterpriseLedgerDirectionPlatformGrant   = "platform_grant"
	EnterpriseLedgerDirectionPlatformReclaim = "platform_reclaim"
	EnterpriseLedgerDirectionManagerGrant    = "manager_grant"
	EnterpriseLedgerDirectionManagerReclaim  = "manager_reclaim"
	EnterpriseLedgerDirectionAdjustment      = "adjustment"
)

var (
	ErrEnterpriseTenantNotFound        = errors.NotFound("ENTERPRISE_TENANT_NOT_FOUND", "enterprise tenant not found")
	ErrEnterpriseMembershipNotFound    = errors.NotFound("ENTERPRISE_MEMBERSHIP_NOT_FOUND", "enterprise membership not found")
	ErrEnterpriseInviteCodeNotFound    = errors.NotFound("ENTERPRISE_INVITE_CODE_NOT_FOUND", "enterprise invite code not found")
	ErrEnterpriseInviteCodeInvalid     = errors.BadRequest("ENTERPRISE_INVITE_CODE_INVALID", "enterprise invite code is invalid")
	ErrEnterpriseInviteCodeDisabled    = errors.BadRequest("ENTERPRISE_INVITE_CODE_DISABLED", "enterprise invite code is disabled")
	ErrEnterpriseInviteCodeExpired     = errors.BadRequest("ENTERPRISE_INVITE_CODE_EXPIRED", "enterprise invite code has expired")
	ErrEnterpriseInviteCodeUsedOut     = errors.BadRequest("ENTERPRISE_INVITE_CODE_USED_OUT", "enterprise invite code has reached max uses")
	ErrEnterpriseUserAlreadyBound      = errors.Conflict("ENTERPRISE_USER_ALREADY_BOUND", "user already belongs to an enterprise")
	ErrEnterpriseManagerRequired       = errors.Forbidden("ENTERPRISE_MANAGER_REQUIRED", "enterprise manager permission required")
	ErrEnterpriseForbidden             = errors.Forbidden("ENTERPRISE_FORBIDDEN", "enterprise access denied")
	ErrEnterpriseTenantDisabled        = errors.Forbidden("ENTERPRISE_TENANT_DISABLED", "enterprise tenant is disabled")
	ErrEnterpriseQuotaExceeded         = errors.BadRequest("ENTERPRISE_QUOTA_EXCEEDED", "enterprise quota exceeded")
	ErrEnterpriseMemberBalanceNegative = errors.BadRequest("ENTERPRISE_MEMBER_BALANCE_NEGATIVE", "member balance cannot become negative")
	ErrEnterprisePricingTooLow         = errors.BadRequest("ENTERPRISE_PRICING_TOO_LOW", "pricing factor must be at least 0.01")
	ErrEnterpriseSelfRechargeForbidden = errors.Forbidden("ENTERPRISE_SELF_RECHARGE_FORBIDDEN", "enterprise users cannot self recharge")
	ErrEnterpriseSelfRedeemForbidden   = errors.Forbidden("ENTERPRISE_SELF_REDEEM_FORBIDDEN", "enterprise users cannot redeem balance codes")
	ErrEnterpriseScopeNotSupported     = errors.BadRequest("ENTERPRISE_SCOPE_NOT_SUPPORTED", "enterprise pricing scope is not supported")
	ErrEnterpriseLastManagerRequired   = errors.Forbidden("ENTERPRISE_LAST_MANAGER_REQUIRED", "cannot remove the last enterprise manager")
	ErrEnterpriseMemberStatusInvalid   = errors.BadRequest("ENTERPRISE_MEMBER_STATUS_INVALID", "enterprise member status is invalid")
	ErrEnterpriseInviteMaxUsesInvalid  = errors.BadRequest("ENTERPRISE_INVITE_MAX_USES_INVALID", "enterprise invite max uses cannot be negative")
)

type EnterpriseTenant struct {
	ID                    int64             `json:"id"`
	Name                       string            `json:"name"`
	Code                       string            `json:"code"`
	Status                     string            `json:"status"`
	Notes                      string            `json:"notes"`
	PortalHost                 string            `json:"portal_host,omitempty"`
	PricingFloorFactor         float64           `json:"pricing_floor_factor"`
	MemberDefaultPricingFactor float64           `json:"member_default_pricing_factor"`
	PricingScope               string            `json:"pricing_scope"`
	Concurrency                int               `json:"concurrency"`
	MemberDefaultConcurrency   int               `json:"member_default_concurrency"`
	BalanceQuotaTotal          float64           `json:"balance_quota_total"`
	BalanceQuotaUsed           float64           `json:"balance_quota_used"`
	BalanceQuotaSpent          float64           `json:"balance_quota_spent"`
	BalanceOverdraftLimit      float64           `json:"balance_overdraft_limit"`
	AllowedGroupIDs            []int64           `json:"allowed_group_ids,omitempty"`
	GroupRates                 map[int64]float64 `json:"group_rates,omitempty"`
	MemberGroupRates           map[int64]float64 `json:"member_group_rates,omitempty"`
	ManagerCount               int64             `json:"manager_count"`
	MemberCount                int64             `json:"member_count"`
	CreatedBy                  *int64            `json:"created_by,omitempty"`
	UpdatedBy                  *int64            `json:"updated_by,omitempty"`
	CreatedAt                  time.Time         `json:"created_at"`
	UpdatedAt                  time.Time         `json:"updated_at"`
}

func (t *EnterpriseTenant) AvailableBalanceQuota() float64 {
	if t == nil {
		return 0
	}
	return t.BalanceQuotaTotal + t.BalanceOverdraftLimit - t.BalanceQuotaSpent
}

func (t *EnterpriseTenant) NetBalanceQuota() float64 {
	if t == nil {
		return 0
	}
	return t.BalanceQuotaTotal - t.BalanceQuotaSpent
}

type EnterpriseMembership struct {
	ID            int64     `json:"id"`
	TenantID      int64     `json:"tenant_id"`
	UserID        int64     `json:"user_id"`
	MemberRole    string    `json:"member_role"`
	MemberNote    string    `json:"member_note"`
	JoinedVia     string    `json:"joined_via"`
	JoinedSource  string    `json:"joined_source"`
	PricingFactor float64   `json:"pricing_factor"`
	PricingScope  string    `json:"pricing_scope"`
	CreatedBy     *int64    `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	UserEmail       string  `json:"user_email"`
	UserUsername    string  `json:"user_username"`
	UserStatus      string  `json:"user_status"`
	UserBalance     float64           `json:"user_balance"`
	UserConcurrency int               `json:"user_concurrency"`
	AllowedGroups   []int64           `json:"allowed_groups,omitempty"`
	GroupRates      map[int64]float64 `json:"group_rates,omitempty"`
}

type EnterpriseGroupSummary struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Platform         string `json:"platform"`
	SubscriptionType string `json:"subscription_type"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	IsExclusive      bool   `json:"is_exclusive"`
	Status           string `json:"status"`
}

type EnterpriseInviteCode struct {
	ID        int64      `json:"id"`
	TenantID  int64      `json:"tenant_id"`
	Code      string     `json:"code"`
	Status    string     `json:"status"`
	MaxUses   int        `json:"max_uses"`
	UsedCount int        `json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Notes     string     `json:"notes"`
	CreatedBy *int64     `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (c *EnterpriseInviteCode) CanUse(now time.Time) error {
	if c == nil {
		return ErrEnterpriseInviteCodeInvalid
	}
	if c.Status != EnterpriseInviteStatusActive {
		return ErrEnterpriseInviteCodeDisabled
	}
	if c.ExpiresAt != nil && c.ExpiresAt.Before(now) {
		return ErrEnterpriseInviteCodeExpired
	}
	if c.MaxUses > 0 && c.UsedCount >= c.MaxUses {
		return ErrEnterpriseInviteCodeUsedOut
	}
	return nil
}

type EnterpriseWalletLedgerEntry struct {
	ID              int64     `json:"id"`
	TenantID        int64     `json:"tenant_id"`
	OperatorUserID  *int64    `json:"operator_user_id,omitempty"`
	TargetUserID    *int64    `json:"target_user_id,omitempty"`
	Direction       string    `json:"direction"`
	Amount          float64   `json:"amount"`
	BalanceBefore   float64   `json:"balance_before"`
	BalanceAfter    float64   `json:"balance_after"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	OperatorEmail   string    `json:"operator_email,omitempty"`
	TargetUserEmail string    `json:"target_user_email,omitempty"`
	TargetUserName  string    `json:"target_user_name,omitempty"`
	TenantName      string    `json:"tenant_name,omitempty"`
	TenantCode      string    `json:"tenant_code,omitempty"`
}

type EnterpriseContext struct {
	TenantID              int64   `json:"tenant_id"`
	TenantName            string  `json:"tenant_name"`
	TenantCode            string  `json:"tenant_code"`
	TenantStatus          string  `json:"tenant_status"`
	PortalHost            string  `json:"portal_host,omitempty"`
	MemberRole            string  `json:"member_role"`
	MemberNote            string  `json:"member_note,omitempty"`
	JoinedVia             string  `json:"joined_via,omitempty"`
	JoinedSource          string  `json:"joined_source,omitempty"`
	PricingFactor              float64 `json:"pricing_factor"`
	PricingScope               string  `json:"pricing_scope"`
	PricingFloorFactor         float64 `json:"pricing_floor_factor"`
	MemberDefaultPricingFactor float64 `json:"member_default_pricing_factor"`
	Concurrency                int     `json:"concurrency"`
	MemberDefaultConcurrency   int     `json:"member_default_concurrency"`
	BalanceQuotaTotal          float64 `json:"balance_quota_total"`
	BalanceQuotaUsed           float64 `json:"balance_quota_used"`
	BalanceQuotaSpent          float64 `json:"balance_quota_spent"`
	BalanceOverdraftLimit      float64 `json:"balance_overdraft_limit"`
	AllowedGroupIDs       []int64           `json:"allowed_group_ids,omitempty"`
	GroupRates            map[int64]float64 `json:"group_rates,omitempty"`
	MemberGroupRates      map[int64]float64 `json:"member_group_rates,omitempty"`
	SelfRechargeBlocked   bool              `json:"self_recharge_blocked"`
	SelfRedeemBlocked     bool              `json:"self_redeem_blocked"`
}

func (c *EnterpriseContext) IsManager() bool {
	return c != nil && c.MemberRole == EnterpriseMemberRoleManager
}

func (c *EnterpriseContext) AvailableBalanceQuota() float64 {
	if c == nil {
		return 0
	}
	return c.BalanceQuotaTotal + c.BalanceOverdraftLimit - c.BalanceQuotaSpent
}

func (c *EnterpriseContext) IsMember() bool {
	return c != nil && c.MemberRole != ""
}

type EnterpriseTenantListFilters struct {
	Status string
	Search string
}

type EnterpriseMemberListFilters struct {
	Status string
	Role   string
	Search string
}

type EnterpriseInviteCodeListFilters struct {
	Status string
	Search string
}

type EnterpriseTenantRepository interface {
	ListTenants(ctx context.Context, params pagination.PaginationParams, filters EnterpriseTenantListFilters) ([]EnterpriseTenant, int64, error)
	GetTenantByID(ctx context.Context, tenantID int64) (*EnterpriseTenant, error)
	GetTenantByCode(ctx context.Context, code string) (*EnterpriseTenant, error)
	NextTenantCode(ctx context.Context) (string, error)
	LockTenantByID(ctx context.Context, tenantID int64) (*EnterpriseTenant, error)
	CreateTenant(ctx context.Context, tenant *EnterpriseTenant) error
	UpdateTenant(ctx context.Context, tenant *EnterpriseTenant) error
	SetTenantAllowedGroups(ctx context.Context, tenantID int64, groupIDs []int64, groupRates map[int64]*float64, memberGroupRates map[int64]*float64) error
	GetTenantAllowedGroups(ctx context.Context, tenantIDs []int64) (map[int64][]int64, error)
	ListTenantGroupSummaries(ctx context.Context, tenantID int64) ([]EnterpriseGroupSummary, error)
	GetTenantGroupRates(ctx context.Context, tenantIDs []int64) (map[int64]map[int64]float64, error)
	GetTenantMemberGroupRates(ctx context.Context, tenantIDs []int64) (map[int64]map[int64]float64, error)
	GetMembershipByUserID(ctx context.Context, userID int64) (*EnterpriseMembership, error)
	GetMembershipByTenantAndUserID(ctx context.Context, tenantID, userID int64) (*EnterpriseMembership, error)
	GetMembershipByTenantAndUserIDForUpdate(ctx context.Context, tenantID, userID int64) (*EnterpriseMembership, error)
	ListMembershipUserIDs(ctx context.Context, tenantID int64) ([]int64, error)
	ListMemberships(ctx context.Context, tenantID int64, params pagination.PaginationParams, filters EnterpriseMemberListFilters) ([]EnterpriseMembership, int64, error)
	CreateMembership(ctx context.Context, membership *EnterpriseMembership) error
	UpdateMembership(ctx context.Context, membership *EnterpriseMembership) error
	DeleteMembership(ctx context.Context, tenantID, userID int64) error
	ListInviteCodes(ctx context.Context, tenantID int64, params pagination.PaginationParams, filters EnterpriseInviteCodeListFilters) ([]EnterpriseInviteCode, int64, error)
	GetInviteCodeByID(ctx context.Context, inviteID int64) (*EnterpriseInviteCode, error)
	GetInviteCodeByCode(ctx context.Context, code string) (*EnterpriseInviteCode, error)
	GetInviteCodeByCodeForUpdate(ctx context.Context, code string) (*EnterpriseInviteCode, error)
	CreateInviteCode(ctx context.Context, invite *EnterpriseInviteCode) error
	UpdateInviteCode(ctx context.Context, invite *EnterpriseInviteCode) error
	IncrementInviteCodeUsage(ctx context.Context, inviteID int64) error
	CreateLedgerEntry(ctx context.Context, entry *EnterpriseWalletLedgerEntry) error
	ListLedger(ctx context.Context, tenantID int64, params pagination.PaginationParams) ([]EnterpriseWalletLedgerEntry, int64, error)
	GetEnterpriseContextByUserID(ctx context.Context, userID int64) (*EnterpriseContext, error)
}

type EnterpriseInviteBinder interface {
	BindUserByInviteCode(ctx context.Context, userID int64, code string, joinedSource string) (*EnterpriseMembership, error)
	ValidateInviteCode(ctx context.Context, code string) error
}

type CreateEnterpriseTenantInput struct {
	Name               string
	Code               string
	Status             string
	Notes                      string
	PortalHost                 string
	PricingFloorFactor         float64
	MemberDefaultPricingFactor float64
	PricingScope               string
	Concurrency                int
	MemberDefaultConcurrency   int
	BalanceOverdraftLimit      float64
	AllowedGroupIDs            []int64
	GroupRates                 map[int64]*float64
	MemberGroupRates           map[int64]*float64
}

type UpdateEnterpriseTenantInput struct {
	Name                       *string
	Status                     *string
	Notes                      *string
	PortalHost                 *string
	PricingFloorFactor         *float64
	MemberDefaultPricingFactor *float64
	PricingScope               *string
	Concurrency                *int
	MemberDefaultConcurrency   *int
	BalanceOverdraftLimit      *float64
	AllowedGroupIDs            *[]int64
	GroupRates                 map[int64]*float64
	MemberGroupRates           map[int64]*float64
}

type AdjustEnterpriseQuotaInput struct {
	Amount    float64
	Direction string
	Notes     string
}

type CreateEnterpriseInviteCodeInput struct {
	Code      string
	MaxUses   int
	ExpiresAt *time.Time
	Notes     string
}

type UpdateEnterpriseInviteCodeInput struct {
	Status    *string
	MaxUses   *int
	ExpiresAt *time.Time
	Notes     *string
}

type BindEnterpriseMemberInput struct {
	UserID        int64
	MemberRole    string
	MemberNote    string
	PricingFactor float64
	PricingScope  string
	GroupRates    map[int64]*float64
	JoinedVia     string
	JoinedSource  string
}

type UpdateEnterpriseMemberInput struct {
	MemberRole    *string
	MemberNote    *string
	PricingFactor *float64
	PricingScope  *string
	Concurrency   *int
	Status        *string
	AllowedGroups *[]int64
	GroupRates    map[int64]*float64
}

type UpdateEnterpriseManagerPricingDefaultsInput struct {
	MemberDefaultPricingFactor *float64
	MemberDefaultConcurrency   *int
	MemberGroupRates           map[int64]*float64
}

type CreateEnterpriseMemberUserInput struct {
	Email          string
	Password       string
	Username       string
	Notes          string
	Concurrency    int
	RPMLimit       *int
	AllowedGroups  []int64
	MemberNote     string
	PricingFactor  float64
	PricingScope   string
	GroupRates     map[int64]*float64
	InitialBalance float64
}

type AdjustEnterpriseMemberBalanceInput struct {
	Amount    float64
	Operation string
	Notes     string
}

type EnterpriseService struct {
	repo                 EnterpriseTenantRepository
	userRepo             UserRepository
	userGroupRateRepo    UserGroupRateRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
	entClient            *dbent.Client
	settingService       *SettingService
}

func NewEnterpriseService(
	repo EnterpriseTenantRepository,
	userRepo UserRepository,
	userGroupRateRepo UserGroupRateRepository,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
	entClient *dbent.Client,
	settingService *SettingService,
) *EnterpriseService {
	return &EnterpriseService{
		repo:                 repo,
		userRepo:             userRepo,
		userGroupRateRepo:    userGroupRateRepo,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
		entClient:            entClient,
		settingService:       settingService,
	}
}

func normalizeEnterpriseTenantStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case EnterpriseTenantStatusDisabled:
		return EnterpriseTenantStatusDisabled
	default:
		return EnterpriseTenantStatusActive
	}
}

func normalizeEnterpriseMemberRole(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case EnterpriseMemberRoleManager:
		return EnterpriseMemberRoleManager
	default:
		return EnterpriseMemberRoleMember
	}
}

func normalizeEnterpriseInviteStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case EnterpriseInviteStatusDisabled:
		return EnterpriseInviteStatusDisabled
	default:
		return EnterpriseInviteStatusActive
	}
}

func normalizeEnterpriseMemberStatus(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case StatusActive:
		return StatusActive, nil
	case StatusDisabled:
		return StatusDisabled, nil
	default:
		return "", ErrEnterpriseMemberStatusInvalid
	}
}

func normalizeEnterprisePricingScope(v string) string {
	return NormalizeEnterprisePricingScopeForRepo(v)
}

func NormalizeEnterprisePricingScopeForRepo(_ string) string {
	// 企业第一版只允许余额/按量分组享受企业倍率，避免订阅也被打折。
	return PromoDiscountScopeBalance
}

func normalizeEnterprisePricingFactor(v float64) float64 {
	return NormalizePricingDiscountFactorForRepo(v)
}

func normalizeEnterpriseMemberPricingFactor(v float64) float64 {
	return NormalizeEnterpriseMemberPricingFactorForRepo(v)
}

func NormalizeEnterpriseMemberPricingFactorForRepo(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return NormalizePricingDiscountFactorForRepo(v)
}

func NormalizeEnterpriseMemberDefaultPricingFactor(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return NormalizePricingDiscountFactorForRepo(v)
}

func normalizeEnterpriseMemberDefaultPricingFactor(v float64) float64 {
	return NormalizeEnterpriseMemberDefaultPricingFactor(v)
}

func normalizeEnterpriseConcurrency(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func (s *EnterpriseService) ListTenants(ctx context.Context, page, pageSize int, filters EnterpriseTenantListFilters, sortBy, sortOrder string) ([]EnterpriseTenant, int64, error) {
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	return s.repo.ListTenants(ctx, params, filters)
}

func (s *EnterpriseService) GetTenant(ctx context.Context, tenantID int64) (*EnterpriseTenant, error) {
	return s.repo.GetTenantByID(ctx, tenantID)
}

func (s *EnterpriseService) CreateTenant(ctx context.Context, actorUserID int64, input CreateEnterpriseTenantInput) (*EnterpriseTenant, error) {
	tenant := &EnterpriseTenant{
		Name:                       strings.TrimSpace(input.Name),
		Code:                       strings.ToUpper(strings.TrimSpace(input.Code)),
		Status:                     normalizeEnterpriseTenantStatus(input.Status),
		Notes:                      strings.TrimSpace(input.Notes),
		PortalHost:                 strings.TrimSpace(input.PortalHost),
		PricingFloorFactor:         normalizeEnterprisePricingFactor(input.PricingFloorFactor),
		MemberDefaultPricingFactor: normalizeEnterpriseMemberDefaultPricingFactor(input.MemberDefaultPricingFactor),
		PricingScope:               normalizeEnterprisePricingScope(input.PricingScope),
		Concurrency:                normalizeEnterpriseConcurrency(input.Concurrency),
		MemberDefaultConcurrency:   normalizeEnterpriseConcurrency(input.MemberDefaultConcurrency),
		BalanceOverdraftLimit:      input.BalanceOverdraftLimit,
	}
	if tenant.Name == "" {
		return nil, errors.BadRequest("ENTERPRISE_TENANT_INVALID", "tenant name is required")
	}
	if tenant.BalanceOverdraftLimit < 0 {
		return nil, errors.BadRequest("ENTERPRISE_OVERDRAFT_LIMIT_INVALID", "enterprise overdraft limit cannot be negative")
	}
	if tenant.PricingScope != PromoDiscountScopeBalance && tenant.PricingScope != PromoDiscountScopeAll && tenant.PricingScope != PromoDiscountScopeSubscription {
		return nil, ErrEnterpriseScopeNotSupported
	}
	if actorUserID > 0 {
		tenant.CreatedBy = &actorUserID
		tenant.UpdatedBy = &actorUserID
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	if tenant.Code == "" {
		nextCode, err := s.repo.NextTenantCode(txCtx)
		if err != nil {
			return nil, err
		}
		tenant.Code = nextCode
	}
	if err := s.repo.CreateTenant(txCtx, tenant); err != nil {
		return nil, err
	}
	if err := validateEnterpriseTenantGroupRates(input.GroupRates, input.AllowedGroupIDs); err != nil {
		return nil, err
	}
	if err := validateEnterpriseTenantGroupRates(input.MemberGroupRates, input.AllowedGroupIDs); err != nil {
		return nil, err
	}
	if err := s.repo.SetTenantAllowedGroups(txCtx, tenant.ID, input.AllowedGroupIDs, input.GroupRates, input.MemberGroupRates); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return s.repo.GetTenantByID(ctx, tenant.ID)
}

func (s *EnterpriseService) UpdateTenant(ctx context.Context, actorUserID, tenantID int64, input UpdateEnterpriseTenantInput) (*EnterpriseTenant, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	tenant, err := s.repo.LockTenantByID(txCtx, tenantID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		tenant.Name = strings.TrimSpace(*input.Name)
	}
	if input.Status != nil {
		tenant.Status = normalizeEnterpriseTenantStatus(*input.Status)
	}
	if input.Notes != nil {
		tenant.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.PortalHost != nil {
		tenant.PortalHost = strings.TrimSpace(*input.PortalHost)
	}
	if input.PricingFloorFactor != nil {
		tenant.PricingFloorFactor = normalizeEnterprisePricingFactor(*input.PricingFloorFactor)
	}
	if input.MemberDefaultPricingFactor != nil {
		tenant.MemberDefaultPricingFactor = normalizeEnterpriseMemberDefaultPricingFactor(*input.MemberDefaultPricingFactor)
	}
	if input.PricingScope != nil {
		tenant.PricingScope = normalizeEnterprisePricingScope(*input.PricingScope)
	}
	if input.Concurrency != nil {
		tenant.Concurrency = normalizeEnterpriseConcurrency(*input.Concurrency)
	}
	if input.MemberDefaultConcurrency != nil {
		tenant.MemberDefaultConcurrency = normalizeEnterpriseConcurrency(*input.MemberDefaultConcurrency)
	}
	if input.BalanceOverdraftLimit != nil {
		if *input.BalanceOverdraftLimit < 0 {
			return nil, errors.BadRequest("ENTERPRISE_OVERDRAFT_LIMIT_INVALID", "enterprise overdraft limit cannot be negative")
		}
		tenant.BalanceOverdraftLimit = *input.BalanceOverdraftLimit
	}
	if tenant.Name == "" {
		return nil, errors.BadRequest("ENTERPRISE_TENANT_INVALID", "tenant name is required")
	}
	if actorUserID > 0 {
		tenant.UpdatedBy = &actorUserID
	}
	if err := s.repo.UpdateTenant(txCtx, tenant); err != nil {
		return nil, err
	}
	if input.AllowedGroupIDs != nil || input.GroupRates != nil || input.MemberGroupRates != nil {
		allowedGroupIDs := tenant.AllowedGroupIDs
		if input.AllowedGroupIDs != nil {
			allowedGroupIDs = *input.AllowedGroupIDs
		}
		if err := validateEnterpriseTenantGroupRates(input.GroupRates, allowedGroupIDs); err != nil {
			return nil, err
		}
		if err := validateEnterpriseTenantGroupRates(input.MemberGroupRates, allowedGroupIDs); err != nil {
			return nil, err
		}
		groupRates := tenant.GroupRates
		if input.GroupRates != nil {
			groupRates = normalizeFloatRateMapPointers(input.GroupRates)
		}
		memberGroupRates := tenant.MemberGroupRates
		if input.MemberGroupRates != nil {
			memberGroupRates = normalizeFloatRateMapPointers(input.MemberGroupRates)
		}
		if err := s.repo.SetTenantAllowedGroups(txCtx, tenantID, allowedGroupIDs, floatRateMapToPointers(groupRates), floatRateMapToPointers(memberGroupRates)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseTenantCaches(ctx, tenantID)
	return s.repo.GetTenantByID(ctx, tenantID)
}

func (s *EnterpriseService) AdjustTenantQuota(ctx context.Context, actorUserID, tenantID int64, input AdjustEnterpriseQuotaInput) (*EnterpriseTenant, error) {
	amount := input.Amount
	if amount <= 0 {
		return nil, errors.BadRequest("ENTERPRISE_QUOTA_AMOUNT_INVALID", "quota amount must be greater than 0")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	tenant, err := s.repo.LockTenantByID(txCtx, tenantID)
	if err != nil {
		return nil, err
	}

	before := tenant.BalanceQuotaTotal
	switch input.Direction {
	case EnterpriseLedgerDirectionPlatformGrant:
		tenant.BalanceQuotaTotal += amount
	case EnterpriseLedgerDirectionPlatformReclaim:
		if tenant.BalanceQuotaTotal < amount || tenant.AvailableBalanceQuota() < amount {
			return nil, ErrEnterpriseQuotaExceeded
		}
		tenant.BalanceQuotaTotal -= amount
	default:
		return nil, errors.BadRequest("ENTERPRISE_QUOTA_DIRECTION_INVALID", "invalid quota direction")
	}
	if tenant.AvailableBalanceQuota() < 0 {
		return nil, ErrEnterpriseQuotaExceeded
	}
	if actorUserID > 0 {
		tenant.UpdatedBy = &actorUserID
	}
	if err := s.repo.UpdateTenant(txCtx, tenant); err != nil {
		return nil, err
	}
	var operator *int64
	if actorUserID > 0 {
		operator = &actorUserID
	}
	if err := s.repo.CreateLedgerEntry(txCtx, &EnterpriseWalletLedgerEntry{
		TenantID:       tenantID,
		OperatorUserID: operator,
		Direction:      input.Direction,
		Amount:         amount,
		BalanceBefore:  before,
		BalanceAfter:   tenant.BalanceQuotaTotal,
		Notes:          strings.TrimSpace(input.Notes),
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseTenantCaches(ctx, tenantID)
	return s.repo.GetTenantByID(ctx, tenantID)
}

func (s *EnterpriseService) ListTenantMembers(ctx context.Context, tenantID int64, page, pageSize int, filters EnterpriseMemberListFilters, sortBy, sortOrder string) ([]EnterpriseMembership, int64, error) {
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	return s.repo.ListMemberships(ctx, tenantID, params, filters)
}

func (s *EnterpriseService) ListMyGroupSummaries(ctx context.Context, managerUserID int64) ([]EnterpriseGroupSummary, *EnterpriseContext, error) {
	enterprise, err := s.GetUserEnterpriseContext(ctx, managerUserID)
	if err != nil {
		return nil, nil, err
	}
	if enterprise == nil || enterprise.TenantID <= 0 {
		return nil, nil, ErrEnterpriseForbidden
	}
	if enterprise.TenantStatus != EnterpriseTenantStatusActive {
		return nil, nil, ErrEnterpriseTenantDisabled
	}
	items, err := s.repo.ListTenantGroupSummaries(ctx, enterprise.TenantID)
	return items, enterprise, err
}

func (s *EnterpriseService) BindUserToTenant(ctx context.Context, actorUserID, tenantID int64, input BindEnterpriseMemberInput) (*EnterpriseMembership, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	tenant, err := s.repo.LockTenantByID(txCtx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status != EnterpriseTenantStatusActive {
		return nil, ErrEnterpriseTenantDisabled
	}
	if existing, err := s.repo.GetMembershipByUserID(txCtx, input.UserID); err == nil && existing != nil {
		return nil, ErrEnterpriseUserAlreadyBound
	} else if err != nil && err != ErrEnterpriseMembershipNotFound {
		return nil, err
	}
	user, err := s.userRepo.GetByID(txCtx, input.UserID)
	if err != nil {
		return nil, err
	}
	membership := &EnterpriseMembership{
		TenantID:      tenantID,
		UserID:        user.ID,
		MemberRole:    normalizeEnterpriseMemberRole(input.MemberRole),
		MemberNote:    strings.TrimSpace(input.MemberNote),
		JoinedVia:     firstNonEmptyString(strings.TrimSpace(input.JoinedVia), EnterpriseJoinViaManualBind),
		JoinedSource:  strings.TrimSpace(input.JoinedSource),
		PricingFactor: normalizeEnterpriseMemberPricingFactor(input.PricingFactor),
		PricingScope:  normalizeEnterprisePricingScope(input.PricingScope),
	}
	if err := validateEnterpriseMemberGroupRates(input.GroupRates, tenant.AllowedGroupIDs); err != nil {
		return nil, err
	}
	if actorUserID > 0 {
		membership.CreatedBy = &actorUserID
	}
	if err := s.repo.CreateMembership(txCtx, membership); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	if err := s.syncEnterpriseMemberGroupRates(ctx, membership.UserID, input.GroupRates); err != nil {
		return nil, err
	}
	s.invalidateEnterpriseUserCaches(ctx, membership.UserID)
	return s.repo.GetMembershipByTenantAndUserID(ctx, tenantID, membership.UserID)
}

func (s *EnterpriseService) UpdateTenantMember(ctx context.Context, actorUserID, tenantID, userID int64, input UpdateEnterpriseMemberInput) (*EnterpriseMembership, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	tenant, err := s.repo.LockTenantByID(txCtx, tenantID)
	if err != nil {
		return nil, err
	}
	membership, err := s.repo.GetMembershipByTenantAndUserIDForUpdate(txCtx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(txCtx, userID)
	if err != nil {
		return nil, err
	}
	if input.MemberRole != nil {
		nextRole := normalizeEnterpriseMemberRole(*input.MemberRole)
		if membership.MemberRole == EnterpriseMemberRoleManager && nextRole != EnterpriseMemberRoleManager && tenant.ManagerCount <= 1 {
			return nil, ErrEnterpriseLastManagerRequired
		}
		membership.MemberRole = nextRole
	}
	if input.MemberNote != nil {
		membership.MemberNote = strings.TrimSpace(*input.MemberNote)
	}
	if input.PricingFactor != nil {
		next := normalizeEnterpriseMemberPricingFactor(*input.PricingFactor)
		membership.PricingFactor = next
	}
	if input.PricingScope != nil {
		membership.PricingScope = normalizeEnterprisePricingScope(*input.PricingScope)
	}
	if err := validateEnterpriseMemberGroupRates(input.GroupRates, tenant.AllowedGroupIDs); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateMembership(txCtx, membership); err != nil {
		return nil, err
	}
	if input.Status != nil {
		nextStatus, err := normalizeEnterpriseMemberStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		if membership.MemberRole == EnterpriseMemberRoleManager && nextStatus != StatusActive && tenant.ManagerCount <= 1 {
			return nil, ErrEnterpriseLastManagerRequired
		}
		user.Status = nextStatus
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return nil, err
		}
	}
	if input.Concurrency != nil {
		user.Concurrency = normalizeEnterpriseConcurrency(*input.Concurrency)
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return nil, err
		}
	}
	if input.AllowedGroups != nil {
		if err := validateEnterpriseAllowedGroups(*input.AllowedGroups, tenant.AllowedGroupIDs); err != nil {
			return nil, err
		}
		user.AllowedGroups = append([]int64(nil), (*input.AllowedGroups)...)
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	if err := s.syncEnterpriseMemberGroupRates(ctx, userID, input.GroupRates); err != nil {
		return nil, err
	}
	s.invalidateEnterpriseUserCaches(ctx, userID)
	return s.repo.GetMembershipByTenantAndUserID(ctx, tenantID, userID)
}

func (s *EnterpriseService) RemoveTenantMember(ctx context.Context, tenantID, userID int64) error {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	tenant, err := s.repo.LockTenantByID(txCtx, tenantID)
	if err != nil {
		return err
	}
	membership, err := s.repo.GetMembershipByTenantAndUserIDForUpdate(txCtx, tenantID, userID)
	if err != nil {
		return err
	}
	if membership.MemberRole == EnterpriseMemberRoleManager && tenant.ManagerCount <= 1 {
		return ErrEnterpriseLastManagerRequired
	}
	if err := s.repo.DeleteMembership(txCtx, tenantID, userID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseUserCaches(ctx, userID)
	return nil
}

func (s *EnterpriseService) ListTenantInviteCodes(ctx context.Context, tenantID int64, page, pageSize int, filters EnterpriseInviteCodeListFilters, sortBy, sortOrder string) ([]EnterpriseInviteCode, int64, error) {
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	return s.repo.ListInviteCodes(ctx, tenantID, params, filters)
}

func (s *EnterpriseService) CreateTenantInviteCode(ctx context.Context, actorUserID, tenantID int64, input CreateEnterpriseInviteCodeInput) (*EnterpriseInviteCode, error) {
	if input.MaxUses < 0 {
		return nil, ErrEnterpriseInviteMaxUsesInvalid
	}
	invite := &EnterpriseInviteCode{
		TenantID:  tenantID,
		Code:      strings.ToUpper(strings.TrimSpace(input.Code)),
		Status:    EnterpriseInviteStatusActive,
		MaxUses:   input.MaxUses,
		ExpiresAt: input.ExpiresAt,
		Notes:     strings.TrimSpace(input.Notes),
	}
	if invite.Code == "" {
		code, err := generateRandomUpperCode(10)
		if err != nil {
			return nil, err
		}
		invite.Code = code
	}
	if actorUserID > 0 {
		invite.CreatedBy = &actorUserID
	}
	if err := s.repo.CreateInviteCode(ctx, invite); err != nil {
		return nil, err
	}
	return s.repo.GetInviteCodeByID(ctx, invite.ID)
}

func (s *EnterpriseService) UpdateTenantInviteCode(ctx context.Context, inviteID int64, input UpdateEnterpriseInviteCodeInput) (*EnterpriseInviteCode, error) {
	invite, err := s.repo.GetInviteCodeByID(ctx, inviteID)
	if err != nil {
		return nil, err
	}
	if input.Status != nil {
		invite.Status = normalizeEnterpriseInviteStatus(*input.Status)
	}
	if input.MaxUses != nil {
		if *input.MaxUses < 0 {
			return nil, ErrEnterpriseInviteMaxUsesInvalid
		}
		invite.MaxUses = *input.MaxUses
	}
	if input.ExpiresAt != nil {
		invite.ExpiresAt = input.ExpiresAt
	}
	if input.Notes != nil {
		invite.Notes = strings.TrimSpace(*input.Notes)
	}
	if err := s.repo.UpdateInviteCode(ctx, invite); err != nil {
		return nil, err
	}
	return s.repo.GetInviteCodeByID(ctx, inviteID)
}

func (s *EnterpriseService) BindUserByInviteCode(ctx context.Context, userID int64, code string, joinedSource string) (*EnterpriseMembership, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if existing, err := s.repo.GetMembershipByUserID(txCtx, userID); err == nil && existing != nil {
		return nil, ErrEnterpriseUserAlreadyBound
	} else if err != nil && err != ErrEnterpriseMembershipNotFound {
		return nil, err
	}
	invite, err := s.repo.GetInviteCodeByCodeForUpdate(txCtx, code)
	if err != nil {
		return nil, err
	}
	if err := invite.CanUse(time.Now()); err != nil {
		return nil, err
	}
	tenant, err := s.repo.LockTenantByID(txCtx, invite.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status != EnterpriseTenantStatusActive {
		return nil, ErrEnterpriseTenantDisabled
	}
	if tenant.MemberDefaultConcurrency > 0 {
		user, err := s.userRepo.GetByID(txCtx, userID)
		if err != nil {
			return nil, err
		}
		user.Concurrency = normalizeEnterpriseConcurrency(tenant.MemberDefaultConcurrency)
		if err := s.userRepo.Update(txCtx, user); err != nil {
			return nil, err
		}
	}
	membership := &EnterpriseMembership{
		TenantID:      invite.TenantID,
		UserID:        userID,
		MemberRole:    EnterpriseMemberRoleMember,
		JoinedVia:     EnterpriseJoinViaInviteCode,
		JoinedSource:  strings.TrimSpace(joinedSource),
		PricingFactor: 0,
		PricingScope:  normalizeEnterprisePricingScope(tenant.PricingScope),
	}
	if err := s.repo.CreateMembership(txCtx, membership); err != nil {
		return nil, err
	}
	if err := s.repo.IncrementInviteCodeUsage(txCtx, invite.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseUserCaches(ctx, userID)
	return s.repo.GetMembershipByTenantAndUserID(ctx, invite.TenantID, userID)
}

func (s *EnterpriseService) ValidateInviteCode(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrEnterpriseInviteCodeInvalid
	}
	if s == nil || s.repo == nil {
		return ErrEnterpriseInviteCodeInvalid
	}
	invite, err := s.repo.GetInviteCodeByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := invite.CanUse(time.Now()); err != nil {
		return err
	}
	tenant, err := s.repo.GetTenantByID(ctx, invite.TenantID)
	if err != nil {
		return err
	}
	if tenant.Status != EnterpriseTenantStatusActive {
		return ErrEnterpriseTenantDisabled
	}
	return nil
}

func (s *EnterpriseService) GetUserEnterpriseContext(ctx context.Context, userID int64) (*EnterpriseContext, error) {
	return s.repo.GetEnterpriseContextByUserID(ctx, userID)
}

func (s *EnterpriseService) GetManagerTenant(ctx context.Context, managerUserID int64) (*EnterpriseContext, error) {
	enterprise, err := s.repo.GetEnterpriseContextByUserID(ctx, managerUserID)
	if err != nil {
		return nil, err
	}
	if enterprise == nil || !enterprise.IsManager() {
		return nil, ErrEnterpriseManagerRequired
	}
	if enterprise.TenantStatus != EnterpriseTenantStatusActive {
		return nil, ErrEnterpriseTenantDisabled
	}
	return enterprise, nil
}

func (s *EnterpriseService) CreateMemberByManager(ctx context.Context, managerUserID int64, input CreateEnterpriseMemberUserInput) (*EnterpriseMembership, *User, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, nil, err
	}
	if len(input.AllowedGroups) > 0 {
		return nil, nil, ErrEnterpriseForbidden
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if existing, err := s.userRepo.ExistsByEmail(txCtx, input.Email); err != nil {
		return nil, nil, err
	} else if existing {
		return nil, nil, ErrEmailExists
	}
	passwordHash, err := hashPasswordForEnterpriseCreate(input.Password)
	if err != nil {
		return nil, nil, err
	}
	defaultRPMLimit := 0
	if input.RPMLimit != nil {
		defaultRPMLimit = *input.RPMLimit
	} else if s.settingService != nil {
		defaultRPMLimit = s.settingService.GetDefaultUserRPMLimit(txCtx)
	}
	memberConcurrency := normalizeEnterpriseConcurrency(input.Concurrency)
	if memberConcurrency <= 0 && managerCtx.MemberDefaultConcurrency > 0 {
		memberConcurrency = managerCtx.MemberDefaultConcurrency
	} else if memberConcurrency <= 0 && s.settingService != nil {
		memberConcurrency = s.settingService.GetDefaultConcurrency(txCtx)
	}
	user := &User{
		Email:        strings.TrimSpace(input.Email),
		PasswordHash: passwordHash,
		Username:     strings.TrimSpace(input.Username),
		Notes:        strings.TrimSpace(input.Notes),
		Role:         RoleUser,
		Balance:      0,
		Concurrency:  memberConcurrency,
		Status:       StatusActive,
		RPMLimit:     defaultRPMLimit,
	}
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return nil, nil, err
	}
	membership := &EnterpriseMembership{
		TenantID:      managerCtx.TenantID,
		UserID:        user.ID,
		MemberRole:    EnterpriseMemberRoleMember,
		MemberNote:    strings.TrimSpace(input.MemberNote),
		JoinedVia:     EnterpriseJoinViaAdminCreate,
		JoinedSource:  "manager_create",
		PricingFactor: normalizeEnterpriseMemberPricingFactor(input.PricingFactor),
		PricingScope:  normalizeEnterprisePricingScope(input.PricingScope),
	}
	if err := validateEnterpriseMemberGroupRates(input.GroupRates, managerCtx.AllowedGroupIDs); err != nil {
		return nil, nil, err
	}
	createdBy := managerUserID
	membership.CreatedBy = &createdBy
	if err := s.repo.CreateMembership(txCtx, membership); err != nil {
		return nil, nil, err
	}
	if input.InitialBalance > 0 {
		if _, err := s.adjustMemberBalanceTx(txCtx, managerCtx, managerUserID, user, input.InitialBalance, EnterpriseLedgerDirectionManagerGrant, "initial balance"); err != nil {
			return nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}
	if err := s.syncEnterpriseMemberGroupRates(ctx, user.ID, input.GroupRates); err != nil {
		return nil, nil, err
	}
	s.invalidateEnterpriseUserCaches(ctx, user.ID)
	createdMembership, err := s.repo.GetMembershipByTenantAndUserID(ctx, managerCtx.TenantID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return createdMembership, user, nil
}

func (s *EnterpriseService) UpdateMemberByManager(ctx context.Context, managerUserID, memberUserID int64, input UpdateEnterpriseMemberInput) (*EnterpriseMembership, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, err
	}
	if input.AllowedGroups != nil {
		return nil, ErrEnterpriseForbidden
	}
	member, err := s.repo.GetMembershipByTenantAndUserID(ctx, managerCtx.TenantID, memberUserID)
	if err != nil {
		return nil, err
	}
	if member.MemberRole == EnterpriseMemberRoleManager && managerUserID == memberUserID && input.MemberRole != nil && normalizeEnterpriseMemberRole(*input.MemberRole) != EnterpriseMemberRoleManager {
		return nil, ErrEnterpriseForbidden
	}
	return s.UpdateTenantMember(ctx, managerUserID, managerCtx.TenantID, memberUserID, input)
}

func (s *EnterpriseService) UpdateMyPricingDefaults(ctx context.Context, managerUserID int64, input UpdateEnterpriseManagerPricingDefaultsInput) (*EnterpriseTenant, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, err
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	tenant, err := s.repo.LockTenantByID(txCtx, managerCtx.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status != EnterpriseTenantStatusActive {
		return nil, ErrEnterpriseTenantDisabled
	}
	if input.MemberDefaultPricingFactor != nil {
		tenant.MemberDefaultPricingFactor = normalizeEnterpriseMemberDefaultPricingFactor(*input.MemberDefaultPricingFactor)
	}
	if input.MemberDefaultConcurrency != nil {
		tenant.MemberDefaultConcurrency = normalizeEnterpriseConcurrency(*input.MemberDefaultConcurrency)
	}
	updatedBy := managerUserID
	tenant.UpdatedBy = &updatedBy
	if err := s.repo.UpdateTenant(txCtx, tenant); err != nil {
		return nil, err
	}
	if input.MemberGroupRates != nil {
		if err := validateEnterpriseTenantGroupRates(input.MemberGroupRates, tenant.AllowedGroupIDs); err != nil {
			return nil, err
		}
		groupRates := floatRateMapToPointers(tenant.GroupRates)
		memberGroupRates := floatRateMapToPointers(tenant.MemberGroupRates)
		if memberGroupRates == nil {
			memberGroupRates = make(map[int64]*float64, len(input.MemberGroupRates))
		}
		for groupID, rate := range input.MemberGroupRates {
			if groupID <= 0 {
				continue
			}
			if rate == nil {
				memberGroupRates[groupID] = nil
				continue
			}
			v := normalizeEnterprisePricingFactor(*rate)
			memberGroupRates[groupID] = &v
		}
		if err := s.repo.SetTenantAllowedGroups(txCtx, tenant.ID, tenant.AllowedGroupIDs, groupRates, memberGroupRates); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseTenantCaches(ctx, managerCtx.TenantID)
	return s.repo.GetTenantByID(ctx, managerCtx.TenantID)
}

func (s *EnterpriseService) AdjustMemberBalanceByManager(ctx context.Context, managerUserID, memberUserID int64, input AdjustEnterpriseMemberBalanceInput) (*EnterpriseMembership, *User, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	membership, err := s.repo.GetMembershipByTenantAndUserIDForUpdate(txCtx, managerCtx.TenantID, memberUserID)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.userRepo.GetByID(txCtx, memberUserID)
	if err != nil {
		return nil, nil, err
	}
	var direction string
	switch strings.ToLower(strings.TrimSpace(input.Operation)) {
	case "grant", "add":
		direction = EnterpriseLedgerDirectionManagerGrant
	case "reclaim", "subtract":
		direction = EnterpriseLedgerDirectionManagerReclaim
	default:
		return nil, nil, errors.BadRequest("ENTERPRISE_BALANCE_OPERATION_INVALID", "invalid enterprise balance operation")
	}
	if _, err := s.adjustMemberBalanceTx(txCtx, managerCtx, managerUserID, user, input.Amount, direction, input.Notes); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.invalidateEnterpriseUserCaches(ctx, memberUserID)
	return membership, user, nil
}

func (s *EnterpriseService) adjustMemberBalanceTx(ctx context.Context, managerCtx *EnterpriseContext, operatorUserID int64, user *User, amount float64, direction string, notes string) (*EnterpriseTenant, error) {
	if amount <= 0 {
		return nil, errors.BadRequest("ENTERPRISE_BALANCE_AMOUNT_INVALID", "balance amount must be greater than 0")
	}
	tenant, err := s.repo.LockTenantByID(ctx, managerCtx.TenantID)
	if err != nil {
		return nil, err
	}
	beforeQuota := tenant.BalanceQuotaUsed
	switch direction {
	case EnterpriseLedgerDirectionManagerGrant:
		tenant.BalanceQuotaUsed += amount
		if err := s.userRepo.UpdateBalance(ctx, user.ID, amount); err != nil {
			return nil, err
		}
		user.Balance += amount
	case EnterpriseLedgerDirectionManagerReclaim:
		if user.Balance < amount {
			return nil, ErrEnterpriseMemberBalanceNegative
		}
		if tenant.BalanceQuotaUsed < amount {
			return nil, ErrEnterpriseQuotaExceeded
		}
		tenant.BalanceQuotaUsed -= amount
		if err := s.userRepo.DeductBalance(ctx, user.ID, amount); err != nil {
			return nil, err
		}
		user.Balance -= amount
	default:
		return nil, errors.BadRequest("ENTERPRISE_BALANCE_OPERATION_INVALID", "invalid enterprise balance operation")
	}
	if err := s.repo.UpdateTenant(ctx, tenant); err != nil {
		return nil, err
	}
	operator := operatorUserID
	target := user.ID
	if err := s.repo.CreateLedgerEntry(ctx, &EnterpriseWalletLedgerEntry{
		TenantID:       tenant.ID,
		OperatorUserID: &operator,
		TargetUserID:   &target,
		Direction:      direction,
		Amount:         amount,
		BalanceBefore:  beforeQuota,
		BalanceAfter:   tenant.BalanceQuotaUsed,
		Notes:          strings.TrimSpace(notes),
	}); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *EnterpriseService) ListLedger(ctx context.Context, tenantID int64, page, pageSize int, sortBy, sortOrder string) ([]EnterpriseWalletLedgerEntry, int64, error) {
	params := pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}
	return s.repo.ListLedger(ctx, tenantID, params)
}

func (s *EnterpriseService) ListMyMembers(ctx context.Context, managerUserID int64, page, pageSize int, filters EnterpriseMemberListFilters, sortBy, sortOrder string) ([]EnterpriseMembership, int64, *EnterpriseContext, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, 0, nil, err
	}
	items, total, err := s.ListTenantMembers(ctx, managerCtx.TenantID, page, pageSize, filters, sortBy, sortOrder)
	return items, total, managerCtx, err
}

func (s *EnterpriseService) ListMyInviteCodes(ctx context.Context, managerUserID int64, page, pageSize int, filters EnterpriseInviteCodeListFilters, sortBy, sortOrder string) ([]EnterpriseInviteCode, int64, *EnterpriseContext, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, 0, nil, err
	}
	items, total, err := s.ListTenantInviteCodes(ctx, managerCtx.TenantID, page, pageSize, filters, sortBy, sortOrder)
	return items, total, managerCtx, err
}

func (s *EnterpriseService) ListMyLedger(ctx context.Context, managerUserID int64, page, pageSize int, sortBy, sortOrder string) ([]EnterpriseWalletLedgerEntry, int64, *EnterpriseContext, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, 0, nil, err
	}
	items, total, err := s.ListLedger(ctx, managerCtx.TenantID, page, pageSize, sortBy, sortOrder)
	return items, total, managerCtx, err
}

func (s *EnterpriseService) CreateMyInviteCode(ctx context.Context, managerUserID int64, input CreateEnterpriseInviteCodeInput) (*EnterpriseInviteCode, *EnterpriseContext, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, nil, err
	}
	item, err := s.CreateTenantInviteCode(ctx, managerUserID, managerCtx.TenantID, input)
	return item, managerCtx, err
}

func (s *EnterpriseService) UpdateMyInviteCode(ctx context.Context, managerUserID, inviteID int64, input UpdateEnterpriseInviteCodeInput) (*EnterpriseInviteCode, *EnterpriseContext, error) {
	managerCtx, err := s.GetManagerTenant(ctx, managerUserID)
	if err != nil {
		return nil, nil, err
	}
	invite, err := s.repo.GetInviteCodeByID(ctx, inviteID)
	if err != nil {
		return nil, nil, err
	}
	if invite.TenantID != managerCtx.TenantID {
		return nil, nil, ErrEnterpriseForbidden
	}
	item, err := s.UpdateTenantInviteCode(ctx, inviteID, input)
	return item, managerCtx, err
}

func (s *EnterpriseService) BindCurrentUserByInviteCode(ctx context.Context, userID int64, code string) (*EnterpriseMembership, error) {
	return s.BindUserByInviteCode(ctx, userID, code, "self_bind")
}

func (s *EnterpriseService) invalidateEnterpriseUserCaches(ctx context.Context, userID int64) {
	if userID <= 0 {
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
	}
}

func (s *EnterpriseService) invalidateEnterpriseTenantCaches(ctx context.Context, tenantID int64) {
	if tenantID <= 0 || s.authCacheInvalidator == nil {
		return
	}
	userIDs, err := s.repo.ListMembershipUserIDs(ctx, tenantID)
	if err != nil {
		return
	}
	for _, userID := range userIDs {
		if userID > 0 {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
	}
}

func (s *EnterpriseService) syncEnterpriseMemberGroupRates(ctx context.Context, userID int64, rates map[int64]*float64) error {
	if rates == nil {
		return nil
	}
	if s.userGroupRateRepo == nil {
		return errors.InternalServer("ENTERPRISE_GROUP_RATES_UNAVAILABLE", "enterprise group rate repository unavailable")
	}
	return s.userGroupRateRepo.SyncUserGroupRates(ctx, userID, normalizeEnterpriseMemberGroupRatesForRepo(rates))
}

func hashPasswordForEnterpriseCreate(password string) (string, error) {
	auth := &AuthService{}
	return auth.HashPassword(password)
}

func generateRandomUpperCode(length int) (string, error) {
	if length <= 0 {
		length = 10
	}
	code, err := GenerateRedeemCode()
	if err != nil {
		return "", err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) > length {
		code = code[:length]
	}
	return code, nil
}

func validateEnterpriseAllowedGroups(candidate, allowed []int64) error {
	if len(candidate) == 0 {
		return nil
	}
	allowedSet := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		if id > 0 {
			allowedSet[id] = struct{}{}
		}
	}
	for _, id := range candidate {
		if id <= 0 {
			continue
		}
		if len(allowedSet) == 0 {
			return ErrEnterpriseForbidden
		}
		if _, ok := allowedSet[id]; !ok {
			return ErrEnterpriseForbidden
		}
	}
	return nil
}

func validateEnterpriseTenantGroupRates(rates map[int64]*float64, allowed []int64) error {
	if len(rates) == 0 {
		return nil
	}
	allowedSet := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		if id > 0 {
			allowedSet[id] = struct{}{}
		}
	}
	if len(allowedSet) == 0 {
		return ErrEnterpriseForbidden
	}
	for groupID, rate := range rates {
		if groupID <= 0 {
			return ErrEnterpriseForbidden
		}
		if _, ok := allowedSet[groupID]; !ok {
			return ErrEnterpriseForbidden
		}
		if rate != nil && *rate < MinPricingDiscountFactor {
			return ErrEnterprisePricingTooLow
		}
	}
	return nil
}

func validateEnterpriseMemberGroupRates(rates map[int64]*float64, allowed []int64) error {
	if len(rates) == 0 {
		return nil
	}
	allowedSet := make(map[int64]struct{}, len(allowed))
	for _, id := range allowed {
		if id > 0 {
			allowedSet[id] = struct{}{}
		}
	}
	for groupID, rate := range rates {
		if groupID <= 0 {
			return ErrEnterpriseForbidden
		}
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[groupID]; !ok {
				return ErrEnterpriseForbidden
			}
		} else {
			return ErrEnterpriseForbidden
		}
		if rate == nil {
			continue
		}
		if *rate < MinPricingDiscountFactor {
			return ErrEnterprisePricingTooLow
		}
	}
	return nil
}

func normalizeEnterpriseMemberGroupRatesForRepo(rates map[int64]*float64) map[int64]*float64 {
	if rates == nil {
		return nil
	}
	out := make(map[int64]*float64, len(rates))
	for groupID, rate := range rates {
		if groupID <= 0 {
			continue
		}
		if rate == nil {
			out[groupID] = nil
			continue
		}
		v := normalizeEnterprisePricingFactor(*rate)
		out[groupID] = &v
	}
	return out
}

func normalizeFloatRateMapPointers(rates map[int64]*float64) map[int64]float64 {
	if rates == nil {
		return nil
	}
	out := make(map[int64]float64, len(rates))
	for groupID, rate := range rates {
		if groupID <= 0 || rate == nil {
			continue
		}
		out[groupID] = normalizeEnterprisePricingFactor(*rate)
	}
	return out
}

func defaultEnterpriseMemberPricingFactor(tenant *EnterpriseTenant) float64 {
	if tenant == nil {
		return DefaultPricingDiscountFactor
	}
	if tenant.MemberDefaultPricingFactor > 0 {
		return normalizeEnterprisePricingFactor(tenant.MemberDefaultPricingFactor)
	}
	return normalizeEnterprisePricingFactor(tenant.PricingFloorFactor)
}

func defaultEnterpriseContextMemberPricingFactor(ctx *EnterpriseContext) float64 {
	if ctx == nil {
		return DefaultPricingDiscountFactor
	}
	if ctx.MemberDefaultPricingFactor > 0 {
		return normalizeEnterprisePricingFactor(ctx.MemberDefaultPricingFactor)
	}
	return normalizeEnterprisePricingFactor(ctx.PricingFloorFactor)
}

func floatRateMapToPointers(rates map[int64]float64) map[int64]*float64 {
	if rates == nil {
		return nil
	}
	out := make(map[int64]*float64, len(rates))
	for groupID, rate := range rates {
		if groupID <= 0 {
			continue
		}
		v := normalizeEnterprisePricingFactor(rate)
		out[groupID] = &v
	}
	return out
}
