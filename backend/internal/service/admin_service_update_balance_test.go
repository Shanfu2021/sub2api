//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type balanceUserRepoStub struct {
	*userRepoStub
	updateErr error
	updated   []*User
}

func (s *balanceUserRepoStub) Update(ctx context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if user == nil {
		return nil
	}
	clone := *user
	s.updated = append(s.updated, &clone)
	if s.userRepoStub != nil {
		s.userRepoStub.user = &clone
	}
	return nil
}

type balanceRedeemRepoStub struct {
	*redeemRepoStub
	created []*RedeemCode
}

func (s *balanceRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return nil
	}
	clone := *code
	s.created = append(s.created, &clone)
	return nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

func TestAdminService_UpdateUserBalance_InvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "")
	require.NoError(t, err)
	require.Equal(t, []int64{7}, invalidator.userIDs)
	require.Len(t, redeemRepo.created, 1)
}

func TestAdminService_UpdateUserBalance_PositiveAdminBalanceAccruesAffiliateRebate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	inviterID := int64(99)
	var gotAmount float64
	var gotCode string
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
		affiliateService: &AffiliateService{
			repo: &affiliateRepoStub{
				ensureUserAffiliateFn: func(ctx context.Context, userID int64) (*AffiliateSummary, error) {
					switch userID {
					case 7:
						return &AffiliateSummary{UserID: 7, InviterID: &inviterID, CreatedAt: time.Now()}, nil
					case inviterID:
						return &AffiliateSummary{UserID: inviterID, CreatedAt: time.Now()}, nil
					default:
						return nil, errors.New("unexpected user")
					}
				},
				countEligibleBalanceCreditsFn: func(ctx context.Context, inviteeUserID int64, excludeOrderID *int64, excludeRedeemCode string) (int, error) {
					require.Equal(t, int64(7), inviteeUserID)
					require.Nil(t, excludeOrderID)
					gotCode = excludeRedeemCode
					return 0, nil
				},
				accrueQuotaFn: func(ctx context.Context, gotInviterID, gotInviteeUserID int64, amount float64, freezeHours int, sourceOrderID *int64) (bool, error) {
					require.Equal(t, inviterID, gotInviterID)
					require.Equal(t, int64(7), gotInviteeUserID)
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
		},
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 50, "add", "")
	require.NoError(t, err)
	require.Len(t, redeemRepo.created, 1)
	require.Contains(t, redeemRepo.created[0].Code, AdminBalanceRedeemCodePrefix)
	require.Equal(t, redeemRepo.created[0].Code, gotCode)
	require.InDelta(t, 5, gotAmount, 1e-9)
}

func TestAdminService_UpdateUserBalance_NegativeAdminBalanceDoesNotAccrueAffiliateRebate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	svc := &adminServiceImpl{
		userRepo:       repo,
		redeemCodeRepo: redeemRepo,
		affiliateService: &AffiliateService{
			repo: &affiliateRepoStub{
				ensureUserAffiliateFn: func(ctx context.Context, userID int64) (*AffiliateSummary, error) {
					t.Fatalf("negative admin balance must not accrue affiliate rebate")
					return nil, nil
				},
			},
			settingService: &SettingService{settingRepo: &affiliateSettingRepoStub{
				values: map[string]string{
					SettingKeyAffiliateEnabled:    "true",
					SettingKeyAffiliateRebateRate: "10",
				},
			}},
		},
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 2, "subtract", "")
	require.NoError(t, err)
	require.Len(t, redeemRepo.created, 1)
	require.Equal(t, -2.0, redeemRepo.created[0].Value)
}

func TestAdminService_UpdateUserBalance_NoChangeNoInvalidate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 10, "set", "")
	require.NoError(t, err)
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, redeemRepo.created)
}
