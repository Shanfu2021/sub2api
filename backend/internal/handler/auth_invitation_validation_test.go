//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type authInvitationValidationSettingRepo struct {
	values map[string]string
}

func (r *authInvitationValidationSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *authInvitationValidationSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (r *authInvitationValidationSettingRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (r *authInvitationValidationSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *authInvitationValidationSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (r *authInvitationValidationSettingRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *authInvitationValidationSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type authInvitationValidationUserRepo struct {
	users map[int64]*service.User
}

func (r *authInvitationValidationUserRepo) Create(context.Context, *service.User) error {
	panic("unexpected Create call")
}

func (r *authInvitationValidationUserRepo) GetByID(_ context.Context, id int64) (*service.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	return user, nil
}

func (r *authInvitationValidationUserRepo) GetByEmail(context.Context, string) (*service.User, error) {
	panic("unexpected GetByEmail call")
}

func (r *authInvitationValidationUserRepo) GetFirstAdmin(context.Context) (*service.User, error) {
	for _, user := range r.users {
		if user.Role == service.RoleAdmin {
			return user, nil
		}
	}
	return nil, service.ErrUserNotFound
}

func (r *authInvitationValidationUserRepo) Update(context.Context, *service.User) error {
	panic("unexpected Update call")
}

func (r *authInvitationValidationUserRepo) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (r *authInvitationValidationUserRepo) HardDelete(context.Context, int64) error {
	panic("unexpected HardDelete call")
}

func (r *authInvitationValidationUserRepo) GetUserAvatar(context.Context, int64) (*service.UserAvatar, error) {
	panic("unexpected GetUserAvatar call")
}

func (r *authInvitationValidationUserRepo) UpsertUserAvatar(context.Context, int64, service.UpsertUserAvatarInput) (*service.UserAvatar, error) {
	panic("unexpected UpsertUserAvatar call")
}

func (r *authInvitationValidationUserRepo) DeleteUserAvatar(context.Context, int64) error {
	panic("unexpected DeleteUserAvatar call")
}

func (r *authInvitationValidationUserRepo) List(context.Context, pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *authInvitationValidationUserRepo) ListWithFilters(context.Context, pagination.PaginationParams, service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *authInvitationValidationUserRepo) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserIDs call")
}

func (r *authInvitationValidationUserRepo) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	panic("unexpected GetLatestUsedAtByUserID call")
}

func (r *authInvitationValidationUserRepo) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	panic("unexpected UpdateUserLastActiveAt call")
}

func (r *authInvitationValidationUserRepo) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}

func (r *authInvitationValidationUserRepo) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}

func (r *authInvitationValidationUserRepo) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}

func (r *authInvitationValidationUserRepo) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchSetConcurrency call")
}

func (r *authInvitationValidationUserRepo) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	panic("unexpected BatchAddConcurrency call")
}

func (r *authInvitationValidationUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (r *authInvitationValidationUserRepo) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (r *authInvitationValidationUserRepo) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (r *authInvitationValidationUserRepo) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (r *authInvitationValidationUserRepo) ListUserAuthIdentities(context.Context, int64) ([]service.UserAuthIdentityRecord, error) {
	panic("unexpected ListUserAuthIdentities call")
}

func (r *authInvitationValidationUserRepo) UnbindUserAuthProvider(context.Context, int64, string) error {
	panic("unexpected UnbindUserAuthProvider call")
}

func (r *authInvitationValidationUserRepo) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (r *authInvitationValidationUserRepo) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}

func (r *authInvitationValidationUserRepo) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

func (r *authInvitationValidationUserRepo) GetByIDIncludeDeleted(ctx context.Context, id int64) (*service.User, error) {
	return r.GetByID(ctx, id)
}

type authInvitationValidationRedeemRepo struct{}

func (r authInvitationValidationRedeemRepo) Create(context.Context, *service.RedeemCode) error {
	panic("unexpected Create call")
}

func (r authInvitationValidationRedeemRepo) CreateBatch(context.Context, []service.RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (r authInvitationValidationRedeemRepo) GetByID(context.Context, int64) (*service.RedeemCode, error) {
	panic("unexpected GetByID call")
}

func (r authInvitationValidationRedeemRepo) GetByCode(context.Context, string) (*service.RedeemCode, error) {
	return nil, service.ErrRedeemCodeNotFound
}

func (r authInvitationValidationRedeemRepo) Update(context.Context, *service.RedeemCode) error {
	panic("unexpected Update call")
}

func (r authInvitationValidationRedeemRepo) BatchUpdate(context.Context, []int64, service.RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (r authInvitationValidationRedeemRepo) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (r authInvitationValidationRedeemRepo) Use(context.Context, int64, int64) error {
	panic("unexpected Use call")
}

func (r authInvitationValidationRedeemRepo) List(context.Context, pagination.PaginationParams) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r authInvitationValidationRedeemRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r authInvitationValidationRedeemRepo) ListByUser(context.Context, int64, int) ([]service.RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (r authInvitationValidationRedeemRepo) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (r authInvitationValidationRedeemRepo) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

type authInvitationValidationAffiliateRepo struct {
	codeOwners map[string]int64
}

func (r *authInvitationValidationAffiliateRepo) EnsureUserAffiliate(context.Context, int64) (*service.AffiliateSummary, error) {
	panic("unexpected EnsureUserAffiliate call")
}

func (r *authInvitationValidationAffiliateRepo) GetAffiliateByCode(_ context.Context, code string) (*service.AffiliateSummary, error) {
	userID, ok := r.codeOwners[code]
	if !ok {
		return nil, service.ErrAffiliateProfileNotFound
	}
	return &service.AffiliateSummary{UserID: userID, AffCode: code}, nil
}

func (r *authInvitationValidationAffiliateRepo) BindInviter(context.Context, int64, int64) (bool, error) {
	panic("unexpected BindInviter call")
}

func (r *authInvitationValidationAffiliateRepo) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	panic("unexpected AccrueQuota call")
}

func (r *authInvitationValidationAffiliateRepo) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (r *authInvitationValidationAffiliateRepo) ThawFrozenQuota(context.Context, int64) (float64, error) {
	panic("unexpected ThawFrozenQuota call")
}

func (r *authInvitationValidationAffiliateRepo) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}

func (r *authInvitationValidationAffiliateRepo) ListInvitees(context.Context, int64, int) ([]service.AffiliateInvitee, error) {
	panic("unexpected ListInvitees call")
}

func (r *authInvitationValidationAffiliateRepo) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (r *authInvitationValidationAffiliateRepo) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}

func (r *authInvitationValidationAffiliateRepo) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}

func (r *authInvitationValidationAffiliateRepo) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (r *authInvitationValidationAffiliateRepo) ListUsersWithCustomSettings(context.Context, service.AffiliateAdminFilter) ([]service.AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (r *authInvitationValidationAffiliateRepo) ListAffiliateInviteRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (r *authInvitationValidationAffiliateRepo) ListAffiliateRebateRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (r *authInvitationValidationAffiliateRepo) ListAffiliateTransferRecords(context.Context, service.AffiliateRecordFilter) ([]service.AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}

func (r *authInvitationValidationAffiliateRepo) GetAffiliateUserOverview(context.Context, int64) (*service.AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}

func TestValidateInvitationCodeAcceptsAffiliateCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	settingSvc := service.NewSettingService(&authInvitationValidationSettingRepo{
		values: map[string]string{
			service.SettingKeyInvitationCodeEnabled: "true",
			service.SettingKeyAffiliateEnabled:      "true",
		},
	}, cfg)
	redeemRepo := authInvitationValidationRedeemRepo{}
	affiliateSvc := service.NewAffiliateService(
		&authInvitationValidationAffiliateRepo{codeOwners: map[string]int64{"AGENTAFF": 2}},
		settingSvc,
		nil,
		nil,
	)
	authSvc := service.NewAuthService(
		nil,
		&authInvitationValidationUserRepo{users: map[int64]*service.User{
			2: {ID: 2, Role: service.RoleAgentLevel1, Status: service.StatusActive},
		}},
		redeemRepo,
		nil,
		cfg,
		settingSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		affiliateSvc,
		nil,
	)
	handler := &AuthHandler{
		authService:   authSvc,
		settingSvc:    settingSvc,
		redeemService: service.NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil),
	}

	body := bytes.NewBufferString(`{"code":"AGENTAFF"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/validate-invitation-code", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.ValidateInvitationCode(c)

	require.Equal(t, http.StatusOK, w.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	raw, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var payload ValidateInvitationCodeResponse
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.True(t, payload.Valid)
	require.Empty(t, payload.ErrorCode)
}
