//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type settingRepoStub struct {
	values map[string]string
	err    error
}

type authInvitationRedeemRepoStub struct {
	code     *RedeemCode
	getCalls []string
	useCalls []struct {
		id     int64
		userID int64
	}
}

func (s *authInvitationRedeemRepoStub) Create(context.Context, *RedeemCode) error {
	panic("unexpected Create call")
}

func (s *authInvitationRedeemRepoStub) CreateBatch(context.Context, []RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (s *authInvitationRedeemRepoStub) GetByID(context.Context, int64) (*RedeemCode, error) {
	panic("unexpected GetByID call")
}

func (s *authInvitationRedeemRepoStub) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	s.getCalls = append(s.getCalls, code)
	if s.code == nil || s.code.Code != code {
		return nil, ErrRedeemCodeNotFound
	}
	clone := *s.code
	return &clone, nil
}

func (s *authInvitationRedeemRepoStub) Update(context.Context, *RedeemCode) error {
	panic("unexpected Update call")
}

func (s *authInvitationRedeemRepoStub) BatchUpdate(context.Context, []int64, RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (s *authInvitationRedeemRepoStub) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *authInvitationRedeemRepoStub) Use(_ context.Context, id, userID int64) error {
	s.useCalls = append(s.useCalls, struct {
		id     int64
		userID int64
	}{id: id, userID: userID})
	return nil
}

func (s *authInvitationRedeemRepoStub) List(context.Context, pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *authInvitationRedeemRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *authInvitationRedeemRepoStub) ListByUser(context.Context, int64, int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (s *authInvitationRedeemRepoStub) ListByUserPaginated(context.Context, int64, pagination.PaginationParams, string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (s *authInvitationRedeemRepoStub) SumPositiveBalanceByUser(context.Context, int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

type authAffiliateRepoStub struct {
	codeOwners map[string]int64
	bindCalls  []struct {
		userID    int64
		inviterID int64
	}
}

func (s *authAffiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	return &AffiliateSummary{UserID: userID, AffCode: "SELF"}, nil
}

func (s *authAffiliateRepoStub) GetAffiliateByCode(_ context.Context, code string) (*AffiliateSummary, error) {
	inviterID, ok := s.codeOwners[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, ErrAffiliateProfileNotFound
	}
	return &AffiliateSummary{UserID: inviterID, AffCode: code}, nil
}

func (s *authAffiliateRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	s.bindCalls = append(s.bindCalls, struct {
		userID    int64
		inviterID int64
	}{userID: userID, inviterID: inviterID})
	return true, nil
}

func (s *authAffiliateRepoStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	panic("unexpected AccrueQuota call")
}

func (s *authAffiliateRepoStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	panic("unexpected GetAccruedRebateFromInvitee call")
}

func (s *authAffiliateRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) {
	return 0, nil
}

func (s *authAffiliateRepoStub) TransferQuotaToBalance(context.Context, int64) (float64, float64, error) {
	panic("unexpected TransferQuotaToBalance call")
}

func (s *authAffiliateRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) {
	panic("unexpected ListInvitees call")
}

func (s *authAffiliateRepoStub) UpdateUserAffCode(context.Context, int64, string) error {
	panic("unexpected UpdateUserAffCode call")
}

func (s *authAffiliateRepoStub) ResetUserAffCode(context.Context, int64) (string, error) {
	panic("unexpected ResetUserAffCode call")
}

func (s *authAffiliateRepoStub) SetUserRebateRate(context.Context, int64, *float64) error {
	panic("unexpected SetUserRebateRate call")
}

func (s *authAffiliateRepoStub) BatchSetUserRebateRate(context.Context, []int64, *float64) error {
	panic("unexpected BatchSetUserRebateRate call")
}

func (s *authAffiliateRepoStub) ListUsersWithCustomSettings(context.Context, AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	panic("unexpected ListUsersWithCustomSettings call")
}

func (s *authAffiliateRepoStub) ListAffiliateInviteRecords(context.Context, AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	panic("unexpected ListAffiliateInviteRecords call")
}

func (s *authAffiliateRepoStub) ListAffiliateRebateRecords(context.Context, AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	panic("unexpected ListAffiliateRebateRecords call")
}

func (s *authAffiliateRepoStub) ListAffiliateTransferRecords(context.Context, AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	panic("unexpected ListAffiliateTransferRecords call")
}

func (s *authAffiliateRepoStub) GetAffiliateUserOverview(context.Context, int64) (*AffiliateUserOverview, error) {
	panic("unexpected GetAffiliateUserOverview call")
}

func (s *settingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := s.values[key]; ok {
			result[key] = v
		}
	}
	return result, nil
}

func (s *settingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type emailCacheStub struct {
	data *VerificationCodeData
	err  error
}

type defaultSubscriptionAssignerStub struct {
	calls []AssignSubscriptionInput
	err   error
}

type refreshTokenCacheStub struct{}

type userPlatformQuotaRepoStub struct {
	bulkInsertCalls [][]UserPlatformQuotaRecord
	bulkInsertErr   error
}

func (s *userPlatformQuotaRepoStub) BulkInsertInitial(_ context.Context, records []UserPlatformQuotaRecord) error {
	cloned := make([]UserPlatformQuotaRecord, len(records))
	copy(cloned, records)
	s.bulkInsertCalls = append(s.bulkInsertCalls, cloned)
	return s.bulkInsertErr
}

func (s *userPlatformQuotaRepoStub) GetByUserPlatform(context.Context, int64, string) (*UserPlatformQuotaRecord, error) {
	panic("unexpected GetByUserPlatform call")
}

func (s *userPlatformQuotaRepoStub) ListByUser(context.Context, int64) ([]UserPlatformQuotaRecord, error) {
	panic("unexpected ListByUser call")
}

func (s *userPlatformQuotaRepoStub) IncrementUsageWithReset(context.Context, int64, string, float64, time.Time) error {
	panic("unexpected IncrementUsageWithReset call")
}

func (s *userPlatformQuotaRepoStub) UpsertForUser(context.Context, int64, []UserPlatformQuotaRecord) error {
	panic("unexpected UpsertForUser call")
}

func (s *userPlatformQuotaRepoStub) ResetExpiredWindow(context.Context, int64, string, string, time.Time) error {
	panic("unexpected ResetExpiredWindow call")
}

func (s *userPlatformQuotaRepoStub) BatchSnapshotUsage(_ context.Context, _ []UserPlatformQuotaSnapshot, _ time.Time) error {
	return nil
}

func (s *defaultSubscriptionAssignerStub) AssignOrExtendSubscription(_ context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	if input != nil {
		s.calls = append(s.calls, *input)
	}
	if s.err != nil {
		return nil, false, s.err
	}
	return &UserSubscription{UserID: input.UserID, GroupID: input.GroupID}, false, nil
}

func (s *refreshTokenCacheStub) StoreRefreshToken(context.Context, string, *RefreshTokenData, time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) GetRefreshToken(context.Context, string) (*RefreshTokenData, error) {
	return nil, ErrRefreshTokenNotFound
}

func (s *refreshTokenCacheStub) DeleteRefreshToken(context.Context, string) error {
	return nil
}

func (s *refreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}

func (s *refreshTokenCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}

func (s *refreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *refreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *refreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}

func (s *emailCacheStub) GetVerificationCode(ctx context.Context, email string) (*VerificationCodeData, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *emailCacheStub) SetVerificationCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) DeleteVerificationCode(ctx context.Context, email string) error {
	return nil
}

func (s *emailCacheStub) GetNotifyVerifyCode(ctx context.Context, email string) (*VerificationCodeData, error) {
	return nil, nil
}

func (s *emailCacheStub) SetNotifyVerifyCode(ctx context.Context, email string, data *VerificationCodeData, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) DeleteNotifyVerifyCode(ctx context.Context, email string) error {
	return nil
}

func (s *emailCacheStub) GetPasswordResetToken(ctx context.Context, email string) (*PasswordResetTokenData, error) {
	return nil, nil
}

func (s *emailCacheStub) SetPasswordResetToken(ctx context.Context, email string, data *PasswordResetTokenData, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) DeletePasswordResetToken(ctx context.Context, email string) error {
	return nil
}

func (s *emailCacheStub) IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool {
	return false
}

func (s *emailCacheStub) SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error {
	return nil
}

func (s *emailCacheStub) GetNotifyCodeUserRate(ctx context.Context, userID int64) (int64, error) {
	return 0, nil
}

func (s *emailCacheStub) IncrNotifyCodeUserRate(ctx context.Context, userID int64, window time.Duration) (int64, error) {
	return 0, nil
}

func newAuthService(repo *userRepoStub, settings map[string]string, emailCache EmailCache, quotaRepo UserPlatformQuotaRepository) *AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireHour: 1,
		},
		Default: config.DefaultConfig{
			UserBalance:     3.5,
			UserConcurrency: 2,
		},
	}

	var settingService *SettingService
	if settings != nil {
		settingService = NewSettingService(&settingRepoStub{values: settings}, cfg)
	}

	var emailService *EmailService
	if emailCache != nil {
		emailService = NewEmailService(&settingRepoStub{values: settings}, emailCache)
	}

	return NewAuthService(
		nil, // entClient
		repo,
		nil, // redeemRepo
		nil, // refreshTokenCache
		cfg,
		settingService,
		emailService,
		nil,
		nil,
		nil, // promoService
		nil, // defaultSubAssigner
		nil, // affiliateService
		quotaRepo,
	)
}

func finiteAgentInviteProfile(userID int64) AgentProfile {
	return AgentProfile{
		UserID:                   userID,
		PoolConcurrency:          10,
		PoolRPM:                  100,
		InviteDefaultConcurrency: DefaultAgentInviteConcurrency,
		InviteDefaultRPM:         DefaultAgentInviteRPM,
	}
}

func attachAgentManagementForRegistrationTest(authService *AuthService, repo *userRepoStub, profiles map[int64]AgentProfile) {
	usersByID := map[int64]*User{}
	if repo.user != nil {
		clone := *repo.user
		usersByID[clone.ID] = &clone
	}
	for _, user := range repo.usersByEmail {
		clone := *user
		usersByID[clone.ID] = &clone
	}
	users := make([]*User, 0, len(usersByID))
	for _, user := range usersByID {
		users = append(users, user)
	}
	agentRepo := newAgentManagementRepoStub(users...)
	agentRepo.agentProfiles = profiles
	authService.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))
}

func TestAuthService_Register_Disabled(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "false",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrRegDisabled)
}

func TestAuthService_Register_DisabledByDefault(t *testing.T) {
	// 当 settings 为 nil（设置项不存在）时，注册应该默认关闭
	repo := &userRepoStub{}
	service := newAuthService(repo, nil, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrRegDisabled)
}

func TestAuthService_Register_SnapshotsPlatformQuotaDefaults(t *testing.T) {
	repo := &userRepoStub{nextID: 77}
	quotaRepo := &userPlatformQuotaRepoStub{}

	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyDefaultPlatformQuotas: `{"openai": {"weekly": 12.34}}`,
	}, nil, quotaRepo)

	_, user, err := service.Register(context.Background(), "newuser@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)

	require.Len(t, quotaRepo.bulkInsertCalls, 1)

	records := quotaRepo.bulkInsertCalls[0]
	var openaiRecord *UserPlatformQuotaRecord
	for i := range records {
		if records[i].Platform == "openai" {
			openaiRecord = &records[i]
			break
		}
	}
	require.NotNil(t, openaiRecord, "expected openai platform record")
	require.Equal(t, int64(77), openaiRecord.UserID)
	require.NotNil(t, openaiRecord.WeeklyLimitUSD)
	require.InDelta(t, 12.34, *openaiRecord.WeeklyLimitUSD, 0.0001)
}

func TestAuthService_Register_DoesNotSnapshotOnDisabled(t *testing.T) {
	repo := &userRepoStub{}
	quotaRepo := &userPlatformQuotaRepoStub{}

	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "false",
	}, nil, quotaRepo)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrRegDisabled)

	require.Empty(t, quotaRepo.bulkInsertCalls, "registration rejected before user creation must not snapshot")
}

func TestAuthService_Register_EmailVerifyEnabledButServiceNotConfigured(t *testing.T) {
	repo := &userRepoStub{}
	// 邮件验证开启但 emailCache 为 nil（emailService 未配置）
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, nil, nil)

	// 应返回服务不可用错误，而不是允许绕过验证
	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "any-code", "", "", "")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_EmailVerifyRequired(t *testing.T) {
	repo := &userRepoStub{}
	cache := &emailCacheStub{} // 配置 emailService
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, cache, nil)

	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "", "", "", "")
	require.ErrorIs(t, err, ErrEmailVerifyRequired)
}

func TestAuthService_Register_EmailVerifyInvalid(t *testing.T) {
	repo := &userRepoStub{}
	cache := &emailCacheStub{
		data: &VerificationCodeData{Code: "expected", Attempts: 0},
	}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
		SettingKeyEmailVerifyEnabled:  "true",
	}, cache, nil)

	_, _, err := service.RegisterWithVerification(context.Background(), "user@test.com", "password", "wrong", "", "", "")
	require.ErrorIs(t, err, ErrInvalidVerifyCode)
	require.ErrorContains(t, err, "verify code")
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	repo := &userRepoStub{exists: true}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestAuthService_Register_CheckEmailError(t *testing.T) {
	repo := &userRepoStub{existsErr: errors.New("db down")}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_ReservedEmail(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "linuxdo-123@linuxdo-connect.invalid", "password")
	require.ErrorIs(t, err, ErrEmailReserved)
}

func TestAuthService_Register_EmailSuffixNotAllowed(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["@example.com","@company.com"]`,
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@other.com", "password")
	require.ErrorIs(t, err, ErrEmailSuffixNotAllowed)
	appErr := infraerrors.FromError(err)
	require.Contains(t, appErr.Message, "@example.com")
	require.Contains(t, appErr.Message, "@company.com")
	require.Equal(t, "EMAIL_SUFFIX_NOT_ALLOWED", appErr.Reason)
	require.Equal(t, "2", appErr.Metadata["allowed_suffix_count"])
	require.Equal(t, "@example.com,@company.com", appErr.Metadata["allowed_suffixes"])
}

func TestAuthService_Register_EmailSuffixAllowed(t *testing.T) {
	repo := &userRepoStub{nextID: 8}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["example.com"]`,
	}, nil, nil)

	_, user, err := service.Register(context.Background(), "user@example.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(8), user.ID)
}

func TestAuthService_SendVerifyCode_EmailSuffixNotAllowed(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyRegistrationEmailSuffixWhitelist: `["@example.com","@company.com"]`,
	}, nil, nil)

	err := service.SendVerifyCode(context.Background(), "user@other.com")
	require.ErrorIs(t, err, ErrEmailSuffixNotAllowed)
	appErr := infraerrors.FromError(err)
	require.Contains(t, appErr.Message, "@example.com")
	require.Contains(t, appErr.Message, "@company.com")
	require.Equal(t, "2", appErr.Metadata["allowed_suffix_count"])
}

func TestAuthService_Register_CreateError(t *testing.T) {
	repo := &userRepoStub{createErr: errors.New("create failed")}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrServiceUnavailable)
}

func TestAuthService_Register_CreateEmailExistsRace(t *testing.T) {
	// 模拟竞态条件：ExistsByEmail 返回 false，但 Create 时因唯一约束失败
	repo := &userRepoStub{createErr: ErrEmailExists}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil, nil)

	_, _, err := service.Register(context.Background(), "user@test.com", "password")
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestAuthService_Register_Success(t *testing.T) {
	repo := &userRepoStub{nextID: 5}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)

	token, user, err := service.Register(context.Background(), "user@test.com", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, user)
	require.Equal(t, int64(5), user.ID)
	require.Equal(t, "user@test.com", user.Email)
	require.Equal(t, RoleUser, user.Role)
	require.Equal(t, StatusActive, user.Status)
	require.Equal(t, 3.5, user.Balance)
	require.Equal(t, 2, user.Concurrency)
	require.Len(t, repo.created, 1)
	require.True(t, user.CheckPassword("password"))
}

func TestRegisterInvitationOnlyAcceptsAffiliateCodeAsInvitation(t *testing.T) {
	repo := &userRepoStub{
		nextID: 100,
		usersByEmail: map[string]*User{
			"agent@test.com": {ID: 2, Email: "agent@test.com", Role: RoleAgentLevel1, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": 2}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	attachAgentManagementForRegistrationTest(service, repo, map[int64]AgentProfile{
		2: finiteAgentInviteProfile(2),
	})

	_, user, err := service.RegisterWithVerification(context.Background(), "affiliate-only@test.com", "password", "", "", "", "AGENTAFF")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(100), user.ID)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, int64(2), *user.ParentUserID)
	require.Equal(t, []struct {
		userID    int64
		inviterID int64
	}{{userID: 100, inviterID: 2}}, affiliateRepo.bindCalls)
}

func TestRegisterAcceptsAffiliateCodeEnteredAsInvitationCode(t *testing.T) {
	repo := &userRepoStub{
		nextID: 104,
		usersByEmail: map[string]*User{
			"agent@test.com": {ID: 2, Email: "agent@test.com", Role: RoleAgentLevel1, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": 2}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.redeemRepo = &authInvitationRedeemRepoStub{}
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	attachAgentManagementForRegistrationTest(service, repo, map[int64]AgentProfile{
		2: finiteAgentInviteProfile(2),
	})

	_, user, err := service.RegisterWithVerification(context.Background(), "affiliate-field@test.com", "password", "", "", "AGENTAFF", "")
	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, int64(2), *user.ParentUserID)
	require.Equal(t, []struct {
		userID    int64
		inviterID int64
	}{{userID: 104, inviterID: 2}}, affiliateRepo.bindCalls)
}

func TestRegisterAssignsParentToAgentInviter(t *testing.T) {
	parentID := int64(2)
	repo := &userRepoStub{
		nextID: 101,
		usersByEmail: map[string]*User{
			"agent@test.com": {ID: parentID, Email: "agent@test.com", Role: RoleAgentLevel1, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": parentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	attachAgentManagementForRegistrationTest(service, repo, map[int64]AgentProfile{
		parentID: finiteAgentInviteProfile(parentID),
	})

	_, user, err := service.RegisterWithVerification(context.Background(), "agent-child@test.com", "password", "", "", "", "AGENTAFF")
	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, parentID, *user.ParentUserID)
}

func TestRegisterWithAgentInvitationUsesAgentInviteDefaultQuota(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	repo := &userRepoStub{
		nextID: 105,
		usersByEmail: map[string]*User{
			"admin@test.com": {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent@test.com": {ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": agentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyInvitationCodeEnabled:               "true",
		SettingKeyAffiliateEnabled:                    "true",
		SettingKeyAuthSourceDefaultEmailConcurrency:   "9",
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "true",
		SettingKeyDefaultUserRPMLimit:                 "90",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: {
			UserID:                   agentID,
			PoolConcurrency:          10,
			PoolRPM:                  100,
			InviteDefaultConcurrency: 3,
			InviteDefaultRPM:         30,
		},
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	_, user, err := service.RegisterWithVerification(context.Background(), "agent-invite@test.com", "password", "", "", "", "AGENTAFF")
	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, agentID, *user.ParentUserID)
	require.Equal(t, 3, user.Concurrency)
	require.Equal(t, 30, user.RPMLimit)
}

func TestRegisterWithAgentInvitationAppliesInviteGroupDefaults(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	groupID := int64(20)
	repo := &userRepoStub{
		nextID: 107,
		usersByEmail: map[string]*User{
			"admin@test.com": {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent@test.com": {ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": agentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: {UserID: agentID, PoolConcurrency: 10, PoolRPM: 100, InviteDefaultConcurrency: 2, InviteDefaultRPM: 20},
	}
	agentRepo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: agentID, groupID: groupID, rateMultiplier: 2.4},
	}
	repo.onCreate = func(user *User) {
		clone := *user
		agentRepo.users[user.ID] = &clone
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	_, user, err := service.RegisterWithVerification(context.Background(), "agent-group@test.com", "password", "", "", "", "AGENTAFF")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(107), user.ID)
	require.ElementsMatch(t, []int64{groupID}, user.AllowedGroups)
	require.Equal(t, []agentGroupDelegationRecord{
		{managerID: agentID, childID: user.ID, groupID: groupID, rateMultiplier: 2.4, canDelegate: false},
	}, agentRepo.groupDelegations)
	require.Equal(t, []struct {
		userID  int64
		groupID int64
	}{{userID: user.ID, groupID: groupID}}, repo.addedAllowedGroups)
	require.Equal(t, []struct {
		userID      int64
		concurrency int
		rpm         int
	}{{userID: agentID, concurrency: 8, rpm: 80}}, agentRepo.setEffectiveQuotas)
}

func TestRegisterWithAdminInvitationAppliesInviteGroupDefaults(t *testing.T) {
	rootID := int64(1)
	groupID := int64(20)
	repo := &userRepoStub{
		nextID: 109,
		usersByEmail: map[string]*User{
			"admin@test.com": {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"ADMINAFF": rootID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
	)
	agentRepo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: rootID, groupID: groupID, rateMultiplier: 2.4},
	}
	repo.onCreate = func(user *User) {
		clone := *user
		agentRepo.users[user.ID] = &clone
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	_, user, err := service.RegisterWithVerification(context.Background(), "admin-group@test.com", "password", "", "", "", "ADMINAFF")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, rootID, *user.ParentUserID)
	require.ElementsMatch(t, []int64{groupID}, user.AllowedGroups)
	require.Equal(t, []agentGroupDelegationRecord{
		{managerID: rootID, childID: user.ID, groupID: groupID, rateMultiplier: 2.4, canDelegate: false},
	}, agentRepo.groupDelegations)
	require.Equal(t, []struct {
		userID  int64
		groupID int64
	}{{userID: user.ID, groupID: groupID}}, repo.addedAllowedGroups)
	require.Empty(t, agentRepo.setEffectiveQuotas)
}

func TestRegisterWithAgentInvitationRollsBackCreatedUserWhenInviteDefaultsFail(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	groupID := int64(20)
	repo := &userRepoStub{
		nextID:      108,
		addGroupErr: errors.New("add group failed"),
		usersByEmail: map[string]*User{
			"admin@test.com": {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent@test.com": {ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": agentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: {UserID: agentID, PoolConcurrency: 10, PoolRPM: 100, InviteDefaultConcurrency: 2, InviteDefaultRPM: 20},
	}
	agentRepo.inviteGroupDefaults = []agentGroupDelegationRecord{
		{managerID: agentID, groupID: groupID, rateMultiplier: 2.4},
	}
	repo.onCreate = func(user *User) {
		clone := *user
		agentRepo.users[user.ID] = &clone
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	_, user, err := service.RegisterWithVerification(context.Background(), "agent-group-fail@test.com", "password", "", "", "", "AGENTAFF")
	require.Error(t, err)
	require.Nil(t, user)
	require.Equal(t, []int64{int64(108)}, repo.hardDeletedIDs)
}

func TestRegisterWithAgentInvitationRejectsWhenAgentInviteQuotaUnavailable(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	existingChildID := int64(3)
	repo := &userRepoStub{
		nextID: 106,
		usersByEmail: map[string]*User{
			"admin@test.com":    {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent@test.com":    {ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
			"existing@test.com": {ID: existingChildID, Email: "existing@test.com", Role: RoleUser, ParentUserID: &agentID, Concurrency: 10, RPMLimit: 100, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": agentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: existingChildID, Email: "existing@test.com", Role: RoleUser, ParentUserID: &agentID, Concurrency: 10, RPMLimit: 100, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: {UserID: agentID, PoolConcurrency: 10, PoolRPM: 100, InviteDefaultConcurrency: 1, InviteDefaultRPM: 1},
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	_, user, err := service.RegisterWithVerification(context.Background(), "blocked@test.com", "password", "", "", "", "AGENTAFF")
	require.ErrorIs(t, err, ErrInvitationAgentQuotaInsufficient)
	require.Nil(t, user)
	require.Empty(t, repo.created)
	require.Empty(t, affiliateRepo.bindCalls)
}

func TestRegisterAssignsParentToNearestAgentForOrdinaryInviter(t *testing.T) {
	rootID := int64(1)
	level2ID := int64(3)
	ordinaryID := int64(4)
	repo := &userRepoStub{
		nextID: 102,
		usersByEmail: map[string]*User{
			"admin@test.com":    {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent2@test.com":   {ID: level2ID, Email: "agent2@test.com", Role: RoleAgentLevel2, ParentUserID: &rootID, Status: StatusActive},
			"ordinary@test.com": {ID: ordinaryID, Email: "ordinary@test.com", Role: RoleUser, ParentUserID: &level2ID, Concurrency: 1, RPMLimit: 1, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"USERAFF": ordinaryID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	attachAgentManagementForRegistrationTest(service, repo, map[int64]AgentProfile{
		level2ID: finiteAgentInviteProfile(level2ID),
	})

	_, user, err := service.RegisterWithVerification(context.Background(), "ordinary-child@test.com", "password", "", "", "", "USERAFF")
	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, level2ID, *user.ParentUserID)
	require.Equal(t, []struct {
		userID    int64
		inviterID int64
	}{{userID: 102, inviterID: ordinaryID}}, affiliateRepo.bindCalls)
}

func TestRegisterAssignsParentToRootAdminWhenNoAgentInChain(t *testing.T) {
	rootID := int64(1)
	ordinaryID := int64(4)
	repo := &userRepoStub{
		nextID: 103,
		usersByEmail: map[string]*User{
			"admin@test.com":    {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"ordinary@test.com": {ID: ordinaryID, Email: "ordinary@test.com", Role: RoleUser, ParentUserID: &rootID, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"USERAFF": ordinaryID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
		SettingKeyAffiliateEnabled:      "true",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)

	_, user, err := service.RegisterWithVerification(context.Background(), "root-child@test.com", "password", "", "", "", "USERAFF")
	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, rootID, *user.ParentUserID)
	require.Equal(t, []struct {
		userID    int64
		inviterID int64
	}{{userID: 103, inviterID: ordinaryID}}, affiliateRepo.bindCalls)
}

func TestAuthService_ValidateToken_ExpiredReturnsClaimsWithError(t *testing.T) {
	repo := &userRepoStub{}
	service := newAuthService(repo, nil, nil, nil)

	// 创建用户并生成 token
	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	// 验证有效 token
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.Equal(t, int64(1), claims.UserID)

	// 模拟过期 token（通过创建一个过期很久的 token）
	service.cfg.JWT.ExpireHour = -1 // 设置为负数使 token 立即过期
	expiredToken, err := service.GenerateToken(user)
	require.NoError(t, err)
	service.cfg.JWT.ExpireHour = 1 // 恢复

	// 验证过期 token 应返回 claims 和 ErrTokenExpired
	claims, err = service.ValidateToken(expiredToken)
	require.ErrorIs(t, err, ErrTokenExpired)
	require.NotNil(t, claims, "claims should not be nil when token is expired")
	require.Equal(t, int64(1), claims.UserID)
	require.Equal(t, "test@test.com", claims.Email)
}

func TestAuthService_RefreshToken_ExpiredTokenNoPanic(t *testing.T) {
	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}
	repo := &userRepoStub{user: user}
	service := newAuthService(repo, nil, nil, nil)

	// 创建过期 token
	service.cfg.JWT.ExpireHour = -1
	expiredToken, err := service.GenerateToken(user)
	require.NoError(t, err)
	service.cfg.JWT.ExpireHour = 1

	// RefreshToken 使用过期 token 不应 panic
	require.NotPanics(t, func() {
		newToken, err := service.RefreshToken(context.Background(), expiredToken)
		require.NoError(t, err)
		require.NotEmpty(t, newToken)
	})
}

func TestAuthService_GetAccessTokenExpiresIn_FallbackToExpireHour(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 0

	require.Equal(t, 24*3600, service.GetAccessTokenExpiresIn())
}

func TestAuthService_GetAccessTokenExpiresIn_MinutesHasPriority(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 90

	require.Equal(t, 90*60, service.GetAccessTokenExpiresIn())
}

func TestAuthService_GenerateToken_UsesExpireHourWhenMinutesZero(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 0

	user := &User{
		ID:           1,
		Email:        "test@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}

	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)

	require.WithinDuration(t, claims.IssuedAt.Time.Add(24*time.Hour), claims.ExpiresAt.Time, 2*time.Second)
}

func TestAuthService_GenerateToken_UsesMinutesWhenConfigured(t *testing.T) {
	service := newAuthService(&userRepoStub{}, nil, nil, nil)
	service.cfg.JWT.ExpireHour = 24
	service.cfg.JWT.AccessTokenExpireMinutes = 90

	user := &User{
		ID:           2,
		Email:        "test2@test.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 1,
	}

	token, err := service.GenerateToken(user)
	require.NoError(t, err)

	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)

	require.WithinDuration(t, claims.IssuedAt.Time.Add(90*time.Minute), claims.ExpiresAt.Time, 2*time.Second)
}

func TestAuthService_Register_AssignsDefaultSubscriptions(t *testing.T) {
	repo := &userRepoStub{nextID: 42}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                `[{"group_id":11,"validity_days":30},{"group_id":12,"validity_days":7}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)
	service.defaultSubAssigner = assigner

	_, user, err := service.Register(context.Background(), "default-sub@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, assigner.calls, 2)
	require.Equal(t, int64(42), assigner.calls[0].UserID)
	require.Equal(t, int64(11), assigner.calls[0].GroupID)
	require.Equal(t, 30, assigner.calls[0].ValidityDays)
	require.Equal(t, int64(12), assigner.calls[1].GroupID)
	require.Equal(t, 7, assigner.calls[1].ValidityDays)
}

func TestAuthService_Register_UsesEmailAuthSourceDefaultsWhenGrantEnabled(t *testing.T) {
	repo := &userRepoStub{nextID: 52}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                `[{"group_id":91,"validity_days":3}]`,
		SettingKeyAuthSourceDefaultEmailBalance:       "12.5",
		SettingKeyAuthSourceDefaultEmailConcurrency:   "7",
		SettingKeyAuthSourceDefaultEmailSubscriptions: `[{"group_id":11,"validity_days":30}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "true",
	}, nil, nil)
	service.defaultSubAssigner = assigner

	_, user, err := service.Register(context.Background(), "email-defaults@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 12.5, user.Balance)
	require.Equal(t, 7, user.Concurrency)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(11), assigner.calls[0].GroupID)
	require.Equal(t, 30, assigner.calls[0].ValidityDays)
}

func TestAuthService_Register_GrantOnSignupFalseFallsBackToGlobalDefaults(t *testing.T) {
	repo := &userRepoStub{nextID: 53}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                `[{"group_id":31,"validity_days":5}]`,
		SettingKeyAuthSourceDefaultEmailBalance:       "99",
		SettingKeyAuthSourceDefaultEmailConcurrency:   "88",
		SettingKeyAuthSourceDefaultEmailSubscriptions: `[{"group_id":32,"validity_days":9}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)
	service.defaultSubAssigner = assigner

	_, user, err := service.Register(context.Background(), "email-global@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 3.5, user.Balance)
	require.Equal(t, 2, user.Concurrency)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(31), assigner.calls[0].GroupID)
	require.Equal(t, 5, assigner.calls[0].ValidityDays)
}

func TestAuthService_Register_GrantOnSignupMergesSourceOverridesWithGlobalDefaults(t *testing.T) {
	repo := &userRepoStub{nextID: 54}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultSubscriptions:                `[{"group_id":31,"validity_days":5}]`,
		SettingKeyAuthSourceDefaultEmailBalance:       "9.5",
		SettingKeyAuthSourceDefaultEmailConcurrency:   "5",
		SettingKeyAuthSourceDefaultEmailSubscriptions: `[]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "true",
	}, nil, nil)
	service.defaultSubAssigner = assigner

	_, user, err := service.Register(context.Background(), "email-merged@test.com", "password")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, 9.5, user.Balance)
	require.Equal(t, 5, user.Concurrency)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(31), assigner.calls[0].GroupID)
	require.Equal(t, 5, assigner.calls[0].ValidityDays)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_UsesLinuxDoAuthSourceDefaultsOnSignup(t *testing.T) {
	repo := &userRepoStub{nextID: 61}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                   "true",
		SettingKeyDefaultSubscriptions:                  `[{"group_id":81,"validity_days":1}]`,
		SettingKeyAuthSourceDefaultLinuxDoBalance:       "21.75",
		SettingKeyAuthSourceDefaultLinuxDoConcurrency:   "9",
		SettingKeyAuthSourceDefaultLinuxDoSubscriptions: `[{"group_id":22,"validity_days":14}]`,
		SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup: "true",
	}, nil, nil)
	service.defaultSubAssigner = assigner
	service.refreshTokenCache = &refreshTokenCacheStub{}

	tokenPair, user, err := service.LoginOrRegisterOAuthWithTokenPair(context.Background(), "linuxdo-123@linuxdo-connect.invalid", "linuxdo_user", "", "", "linuxdo")
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.NotNil(t, user)
	require.Equal(t, int64(61), user.ID)
	require.Equal(t, 21.75, user.Balance)
	require.Equal(t, 9, user.Concurrency)
	require.Len(t, repo.created, 1)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(22), assigner.calls[0].GroupID)
	require.Equal(t, 14, assigner.calls[0].ValidityDays)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_UsesAgentInviteDefaultQuota(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	repo := &userRepoStub{
		nextID: 62,
		usersByEmail: map[string]*User{
			"admin@test.com": {ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
			"agent@test.com": {ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"AGENTAFF": agentID}}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                   "true",
		SettingKeyInvitationCodeEnabled:                 "true",
		SettingKeyAffiliateEnabled:                      "true",
		SettingKeyAuthSourceDefaultLinuxDoBalance:       "21.75",
		SettingKeyAuthSourceDefaultLinuxDoConcurrency:   "9",
		SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup: "true",
		SettingKeyDefaultUserRPMLimit:                   "90",
	}, nil, nil)
	service.affiliateService = NewAffiliateService(affiliateRepo, service.settingService, nil, nil)
	service.refreshTokenCache = &refreshTokenCacheStub{}
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@test.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@test.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: {
			UserID:                   agentID,
			PoolConcurrency:          10,
			PoolRPM:                  100,
			InviteDefaultConcurrency: 5,
			InviteDefaultRPM:         50,
		},
	}
	service.SetAgentManagementService(NewAgentManagementService(agentRepo, repo, nil, nil))

	tokenPair, user, err := service.LoginOrRegisterOAuthWithTokenPair(context.Background(), "linuxdo-456@linuxdo-connect.invalid", "linuxdo_user", "AGENTAFF", "", "linuxdo")
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.NotNil(t, user)
	require.Equal(t, 5, user.Concurrency)
	require.Equal(t, 50, user.RPMLimit)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, agentID, *user.ParentUserID)
}

func TestAuthService_LoginOrRegisterOAuthWithTokenPair_ExistingUserDoesNotGrantAgain(t *testing.T) {
	existing := &User{
		ID:           88,
		Email:        "linuxdo-123@linuxdo-connect.invalid",
		Username:     "existing-linuxdo",
		Role:         RoleUser,
		Status:       StatusActive,
		Balance:      4,
		Concurrency:  1,
		TokenVersion: 2,
	}
	repo := &userRepoStub{user: existing}
	assigner := &defaultSubscriptionAssignerStub{}
	service := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                   "true",
		SettingKeyAuthSourceDefaultLinuxDoBalance:       "21.75",
		SettingKeyAuthSourceDefaultLinuxDoConcurrency:   "9",
		SettingKeyAuthSourceDefaultLinuxDoSubscriptions: `[{"group_id":22,"validity_days":14}]`,
		SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup: "true",
	}, nil, nil)
	service.defaultSubAssigner = assigner
	service.refreshTokenCache = &refreshTokenCacheStub{}

	tokenPair, user, err := service.LoginOrRegisterOAuthWithTokenPair(context.Background(), existing.Email, "linuxdo_user", "", "", "linuxdo")
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.Equal(t, existing.ID, user.ID)
	require.Equal(t, 4.0, user.Balance)
	require.Equal(t, 1, user.Concurrency)
	require.Empty(t, repo.created)
	require.Empty(t, assigner.calls)
}

// newAuthServiceWithDingTalkCfg 构建一个含完整 DingTalk config 的 AuthService，
// 用于测试 canBypassRegistrationDisabledForOAuth。
func newAuthServiceWithDingTalkCfg(settings map[string]string, dtCfg config.DingTalkConnectConfig) *AuthService {
	cfg := &config.Config{
		JWT:      config.JWTConfig{Secret: "test-secret", ExpireHour: 1},
		Default:  config.DefaultConfig{UserBalance: 3.5, UserConcurrency: 2},
		DingTalk: dtCfg,
	}
	settingService := NewSettingService(&settingRepoStub{values: settings}, cfg)
	return NewAuthService(nil, nil, nil, nil, cfg, settingService, nil, nil, nil, nil, nil, nil, nil)
}

// minDingTalkURLs 返回一个包含必填字段的基础 DingTalkConnectConfig（不设 Enabled/BypassRegistration/Policy）。
func minDingTalkURLs() config.DingTalkConnectConfig {
	return config.DingTalkConnectConfig{
		ClientID:            "test-client",
		ClientSecret:        "test-secret",
		AuthorizeURL:        "https://example.com/oauth2/auth",
		TokenURL:            "https://example.com/oauth2/token",
		UserInfoURL:         "https://example.com/oauth2/userinfo",
		RedirectURL:         "https://example.com/callback",
		FrontendRedirectURL: "https://example.com/auth/callback",
		DingTalkAppKind:     "internal_app",
		AppType:             "internal",
	}
}

func TestCanBypassRegistrationDisabledForOAuth(t *testing.T) {
	cases := []struct {
		name         string
		signupSource string
		settings     map[string]string
		dtCfg        config.DingTalkConnectConfig
		want         bool
	}{
		{
			name:         "non-dingtalk source → false",
			signupSource: "linuxdo",
			settings:     map[string]string{},
			dtCfg:        minDingTalkURLs(),
			want:         false,
		},
		{
			name:         "dingtalk but cfg.Enabled=false → false",
			signupSource: "dingtalk",
			settings: map[string]string{
				SettingKeyDingTalkConnectEnabled:               "false",
				SettingKeyDingTalkConnectBypassRegistration:    "true",
				SettingKeyDingTalkConnectCorpRestrictionPolicy: "internal_only",
			},
			dtCfg: minDingTalkURLs(),
			want:  false,
		},
		{
			name:         "dingtalk enabled but BypassRegistration=false → false",
			signupSource: "dingtalk",
			settings: map[string]string{
				SettingKeyDingTalkConnectEnabled:               "true",
				SettingKeyDingTalkConnectBypassRegistration:    "false",
				SettingKeyDingTalkConnectCorpRestrictionPolicy: "internal_only",
			},
			dtCfg: minDingTalkURLs(),
			want:  false,
		},
		{
			name:         "dingtalk enabled + bypass=true but policy=none → false",
			signupSource: "dingtalk",
			settings: map[string]string{
				SettingKeyDingTalkConnectEnabled:               "true",
				SettingKeyDingTalkConnectBypassRegistration:    "true",
				SettingKeyDingTalkConnectCorpRestrictionPolicy: "none",
			},
			dtCfg: minDingTalkURLs(),
			want:  false,
		},
		{
			name:         "dingtalk enabled + bypass=true + policy=internal_only → true",
			signupSource: "dingtalk",
			settings: map[string]string{
				SettingKeyDingTalkConnectEnabled:               "true",
				SettingKeyDingTalkConnectBypassRegistration:    "true",
				SettingKeyDingTalkConnectCorpRestrictionPolicy: "internal_only",
			},
			dtCfg: minDingTalkURLs(),
			want:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newAuthServiceWithDingTalkCfg(tc.settings, tc.dtCfg)
			got := svc.canBypassRegistrationDisabledForOAuth(context.Background(), tc.signupSource)
			require.Equal(t, tc.want, got)
		})
	}
}
