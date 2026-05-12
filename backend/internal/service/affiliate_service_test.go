//go:build unit

package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type affiliateRepoStub struct {
	ensureUserAffiliateFn          func(ctx context.Context, userID int64) (*AffiliateSummary, error)
	countEligibleBalanceCreditsFn  func(ctx context.Context, inviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error)
	accrueQuotaFn                  func(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error)
	getAccruedRebateFromInviteeFn  func(ctx context.Context, inviterID, inviteeUserID int64) (float64, error)
}

func (s *affiliateRepoStub) EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error) {
	if s.ensureUserAffiliateFn == nil {
		return nil, errors.New("unexpected EnsureUserAffiliate")
	}
	return s.ensureUserAffiliateFn(ctx, userID)
}

func (s *affiliateRepoStub) GetAffiliateByCode(ctx context.Context, code string) (*AffiliateSummary, error) {
	return nil, errors.New("unexpected GetAffiliateByCode")
}

func (s *affiliateRepoStub) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	return false, errors.New("unexpected BindInviter")
}

func (s *affiliateRepoStub) AccrueQuota(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error) {
	if s.accrueQuotaFn == nil {
		return false, errors.New("unexpected AccrueQuota")
	}
	return s.accrueQuotaFn(ctx, inviterID, inviteeUserID, amount, freezeHours, sourceOrderID)
}

func (s *affiliateRepoStub) GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeUserID int64) (float64, error) {
	if s.getAccruedRebateFromInviteeFn == nil {
		return 0, nil
	}
	return s.getAccruedRebateFromInviteeFn(ctx, inviterID, inviteeUserID)
}

func (s *affiliateRepoStub) CountEligibleBalanceCredits(ctx context.Context, inviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error) {
	if s.countEligibleBalanceCreditsFn == nil {
		return 0, nil
	}
	return s.countEligibleBalanceCreditsFn(ctx, inviteeUserID, excludeOrderID, excludeRedeemCode)
}

func (s *affiliateRepoStub) ThawFrozenQuota(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("unexpected ThawFrozenQuota")
}

func (s *affiliateRepoStub) TransferQuotaToBalance(ctx context.Context, userID int64) (float64, float64, error) {
	return 0, 0, errors.New("unexpected TransferQuotaToBalance")
}

func (s *affiliateRepoStub) ListInvitees(ctx context.Context, inviterID int64, limit int) ([]AffiliateInvitee, error) {
	return nil, errors.New("unexpected ListInvitees")
}

func (s *affiliateRepoStub) UpdateUserAffCode(ctx context.Context, userID int64, newCode string) error {
	return errors.New("unexpected UpdateUserAffCode")
}

func (s *affiliateRepoStub) ResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	return "", errors.New("unexpected ResetUserAffCode")
}

func (s *affiliateRepoStub) SetUserRebateRate(ctx context.Context, userID int64, ratePercent *float64) error {
	return errors.New("unexpected SetUserRebateRate")
}

func (s *affiliateRepoStub) BatchSetUserRebateRate(ctx context.Context, userIDs []int64, ratePercent *float64) error {
	return errors.New("unexpected BatchSetUserRebateRate")
}

func (s *affiliateRepoStub) ListUsersWithCustomSettings(ctx context.Context, filter AffiliateAdminFilter) ([]AffiliateAdminEntry, int64, error) {
	return nil, 0, errors.New("unexpected ListUsersWithCustomSettings")
}

func (s *affiliateRepoStub) ListAffiliateInviteRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateInviteRecord, int64, error) {
	return nil, 0, errors.New("unexpected ListAffiliateInviteRecords")
}

func (s *affiliateRepoStub) ListAffiliateRebateRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateRebateRecord, int64, error) {
	return nil, 0, errors.New("unexpected ListAffiliateRebateRecords")
}

func (s *affiliateRepoStub) ListAffiliateTransferRecords(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateTransferRecord, int64, error) {
	return nil, 0, errors.New("unexpected ListAffiliateTransferRecords")
}

func (s *affiliateRepoStub) GetAffiliateUserOverview(ctx context.Context, userID int64) (*AffiliateUserOverview, error) {
	return nil, errors.New("unexpected GetAffiliateUserOverview")
}

type affiliateSettingRepoStub struct {
	values map[string]string
}

func (s *affiliateSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *affiliateSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	v, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (s *affiliateSettingRepoStub) Set(ctx context.Context, key, value string) error {
	return errors.New("unexpected Set")
}

func (s *affiliateSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return nil, errors.New("unexpected GetMultiple")
}

func (s *affiliateSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	return errors.New("unexpected SetMultiple")
}

func (s *affiliateSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return nil, errors.New("unexpected GetAll")
}

func (s *affiliateSettingRepoStub) Delete(ctx context.Context, key string) error {
	return errors.New("unexpected Delete")
}

// TestResolveRebateRatePercent_PerUserOverride verifies that per-inviter
// AffRebateRatePercent overrides the global rate, that NULL falls back to the
// global rate, and that out-of-range exclusive rates are clamped silently.
//
// SettingService is left nil here so globalRebateRatePercent returns the
// documented default (AffiliateRebateRateDefault = 20%) — this exercises the
// fallback path without spinning up a settings stub.
func TestResolveRebateRatePercent_PerUserOverride(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}

	// nil exclusive rate → falls back to global default (20%)
	require.InDelta(t, AffiliateRebateRateDefault,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{}), 1e-9)

	// exclusive rate set → overrides global
	rate := 50.0
	require.InDelta(t, 50.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &rate}), 1e-9)

	// exclusive rate 0 → returns 0 (no rebate, intentional)
	zero := 0.0
	require.InDelta(t, 0.0,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &zero}), 1e-9)

	// exclusive rate above max → clamped to Max
	tooHigh := 250.0
	require.InDelta(t, AffiliateRebateRateMax,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooHigh}), 1e-9)

	// exclusive rate below min → clamped to Min
	tooLow := -5.0
	require.InDelta(t, AffiliateRebateRateMin,
		svc.resolveRebateRatePercent(context.Background(), &AffiliateSummary{AffRebateRatePercent: &tooLow}), 1e-9)
}

// TestIsEnabled_NilSettingServiceReturnsDefault verifies that IsEnabled
// safely handles a nil settingService dependency by returning the default
// (off). This protects callers from nil-pointer crashes in misconfigured
// environments.
func TestIsEnabled_NilSettingServiceReturnsDefault(t *testing.T) {
	t.Parallel()
	svc := &AffiliateService{}
	require.False(t, svc.IsEnabled(context.Background()))
	require.Equal(t, AffiliateEnabledDefault, svc.IsEnabled(context.Background()))
}

// TestValidateExclusiveRate_BoundaryAndInvalid covers the validator used by
// admin-facing rate setters: nil is always valid (clear), in-range values
// are accepted, NaN/Inf and out-of-range values produce a typed BadRequest.
func TestValidateExclusiveRate_BoundaryAndInvalid(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateExclusiveRate(nil))

	for _, v := range []float64{0, 0.01, 50, 99.99, 100} {
		v := v
		require.NoError(t, validateExclusiveRate(&v), "value %v should be valid", v)
	}

	for _, v := range []float64{-0.01, 100.01, -100, 200} {
		v := v
		require.Error(t, validateExclusiveRate(&v), "value %v should be rejected", v)
	}

	nan := math.NaN()
	require.Error(t, validateExclusiveRate(&nan))
	posInf := math.Inf(1)
	require.Error(t, validateExclusiveRate(&posInf))
	negInf := math.Inf(-1)
	require.Error(t, validateExclusiveRate(&negInf))
}

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	require.Equal(t, "a***@g***.com", maskEmail("alice@gmail.com"))
	require.Equal(t, "x***@d***", maskEmail("x@domain"))
	require.Equal(t, "", maskEmail(""))
}

func TestIsValidAffiliateCodeFormat(t *testing.T) {
	t.Parallel()

	// 邀请码格式校验同时服务于：
	// 1) 系统自动生成的 12 位随机码（A-Z 去 I/O，2-9 去 0/1）
	// 2) 管理员设置的自定义专属码（如 "VIP2026"、"NEW_USER-1"）
	// 因此校验放宽到 [A-Z0-9_-]{4,32}（要求调用方先 ToUpper）。
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"valid canonical 12-char", "ABCDEFGHJKLM", true},
		{"valid all digits 2-9", "234567892345", true},
		{"valid mixed", "A2B3C4D5E6F7", true},
		{"valid admin custom short", "VIP1", true},
		{"valid admin custom with hyphen", "NEW-USER", true},
		{"valid admin custom with underscore", "VIP_2026", true},
		{"valid 32-char max", "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345", true},
		// Previously-excluded chars (I/O/0/1) are now allowed since admins may use them.
		{"letter I now allowed", "IBCDEFGHJKLM", true},
		{"letter O now allowed", "OBCDEFGHJKLM", true},
		{"digit 0 now allowed", "0BCDEFGHJKLM", true},
		{"digit 1 now allowed", "1BCDEFGHJKLM", true},
		{"too short (3 chars)", "ABC", false},
		{"too long (33 chars)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456", false},
		{"lowercase rejected (caller must ToUpper first)", "abcdefghjklm", false},
		{"empty", "", false},
		{"utf8 non-ascii", "ÄÄÄÄÄÄ", false}, // bytes out of charset
		{"ascii punctuation .", "ABCDEFGHJK.M", false},
		{"whitespace", "ABCDEFGHJK M", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isValidAffiliateCodeFormat(tc.in))
		})
	}
}

func TestAccrueInviteRebateForOrder_StopsAfterFirstThreeEligibleBalanceOrders(t *testing.T) {
	t.Parallel()

	var accrueCalled bool
	orderID := int64(99)
	inviterID := int64(200)
	inviteeID := int64(100)
	now := time.Now()
	svc := &AffiliateService{
		repo: &affiliateRepoStub{
			ensureUserAffiliateFn: func(ctx context.Context, userID int64) (*AffiliateSummary, error) {
				switch userID {
				case inviteeID:
					return &AffiliateSummary{
						UserID:    inviteeID,
						InviterID: &inviterID,
						CreatedAt: now,
					}, nil
				case inviterID:
					return &AffiliateSummary{
						UserID:    inviterID,
						CreatedAt: now,
					}, nil
				default:
					return nil, errors.New("unexpected user")
				}
			},
			countEligibleBalanceCreditsFn: func(ctx context.Context, gotInviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error) {
				require.Equal(t, inviteeID, gotInviteeUserID)
				require.NotNil(t, excludeOrderID)
				require.Equal(t, orderID, *excludeOrderID)
				require.Empty(t, excludeRedeemCode)
				return 3, nil
			},
			accrueQuotaFn: func(ctx context.Context, inviterID, inviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error) {
				accrueCalled = true
				return true, nil
			},
		},
		settingService: &SettingService{settingRepo: &affiliateSettingRepoStub{
			values: map[string]string{
				SettingKeyAffiliateEnabled:    "true",
				SettingKeyAffiliateRebateRate: "10",
			},
		}},
	}

	rebate, err := svc.AccrueInviteRebateForOrder(context.Background(), inviteeID, 100, &orderID)
	require.NoError(t, err)
	require.Zero(t, rebate)
	require.False(t, accrueCalled)
}

func TestAccrueInviteRebateForOrder_AppliesWithinFirstThreeEligibleBalanceOrders(t *testing.T) {
	t.Parallel()

	orderID := int64(77)
	inviterID := int64(200)
	inviteeID := int64(100)
	now := time.Now()
	var gotAmount float64
	var gotFreezeHours int
	var gotSourceOrderID int64
	svc := &AffiliateService{
		repo: &affiliateRepoStub{
			ensureUserAffiliateFn: func(ctx context.Context, userID int64) (*AffiliateSummary, error) {
				switch userID {
				case inviteeID:
					return &AffiliateSummary{
						UserID:    inviteeID,
						InviterID: &inviterID,
						CreatedAt: now,
					}, nil
				case inviterID:
					return &AffiliateSummary{
						UserID:    inviterID,
						CreatedAt: now,
					}, nil
				default:
					return nil, errors.New("unexpected user")
				}
			},
			countEligibleBalanceCreditsFn: func(ctx context.Context, gotInviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error) {
				require.Equal(t, inviteeID, gotInviteeUserID)
				require.NotNil(t, excludeOrderID)
				require.Equal(t, orderID, *excludeOrderID)
				require.Empty(t, excludeRedeemCode)
				return 2, nil
			},
			accrueQuotaFn: func(ctx context.Context, gotInviterID, gotInviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error) {
				require.Equal(t, inviterID, gotInviterID)
				require.Equal(t, inviteeID, gotInviteeUserID)
				require.NotNil(t, sourceOrderID)
				gotSourceOrderID = *sourceOrderID
				gotAmount = amount
				gotFreezeHours = freezeHours
				return true, nil
			},
		},
		settingService: &SettingService{settingRepo: &affiliateSettingRepoStub{
			values: map[string]string{
				SettingKeyAffiliateEnabled:          "true",
				SettingKeyAffiliateRebateRate:       "10",
				SettingKeyAffiliateRebateFreezeHours: "24",
			},
		}},
	}

	rebate, err := svc.AccrueInviteRebateForOrder(context.Background(), inviteeID, 100, &orderID)
	require.NoError(t, err)
	require.InDelta(t, 10, rebate, 1e-9)
	require.InDelta(t, 10, gotAmount, 1e-9)
	require.Equal(t, 24, gotFreezeHours)
	require.Equal(t, orderID, gotSourceOrderID)
}

func TestAccrueInviteRebateForRedeemCode_AllowsThirdBalanceCredit(t *testing.T) {
	t.Parallel()

	inviterID := int64(200)
	inviteeID := int64(100)
	now := time.Now()
	var gotAmount float64
	svc := &AffiliateService{
		repo: &affiliateRepoStub{
			ensureUserAffiliateFn: func(ctx context.Context, userID int64) (*AffiliateSummary, error) {
				switch userID {
				case inviteeID:
					return &AffiliateSummary{
						UserID:    inviteeID,
						InviterID: &inviterID,
						CreatedAt: now,
					}, nil
				case inviterID:
					return &AffiliateSummary{
						UserID:    inviterID,
						CreatedAt: now,
					}, nil
				default:
					return nil, errors.New("unexpected user")
				}
			},
			countEligibleBalanceCreditsFn: func(ctx context.Context, gotInviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error) {
				require.Equal(t, inviteeID, gotInviteeUserID)
				require.Nil(t, excludeOrderID)
				require.Equal(t, "BALANCE-CARD-003", excludeRedeemCode)
				return 2, nil
			},
			accrueQuotaFn: func(ctx context.Context, gotInviterID, gotInviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error) {
				require.Equal(t, inviterID, gotInviterID)
				require.Equal(t, inviteeID, gotInviteeUserID)
				require.Nil(t, sourceOrderID)
				gotAmount = amount
				return true, nil
			},
		},
		settingService: &SettingService{settingRepo: &affiliateSettingRepoStub{
			values: map[string]string{
				SettingKeyAffiliateEnabled:    "true",
				SettingKeyAffiliateRebateRate: "10",
			},
		}},
	}

	rebate, err := svc.AccrueInviteRebateForRedeemCode(context.Background(), inviteeID, 100, "BALANCE-CARD-003")
	require.NoError(t, err)
	require.InDelta(t, 10, rebate, 1e-9)
	require.InDelta(t, 10, gotAmount, 1e-9)
}
