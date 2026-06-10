//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newEmailOAuthAutoAuthService(
	userRepo UserRepository,
	settings map[string]string,
	quotaRepo UserPlatformQuotaRepository,
) *AuthService {
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
		Default: config.DefaultConfig{
			UserBalance:     3.5,
			UserConcurrency: 2,
		},
	}

	settingService := NewSettingService(&settingRepoStub{values: settings}, cfg)

	return NewAuthService(
		nil, // entClient — nil, updateUserSignupSource early return
		userRepo,
		nil, // redeemRepo — invitationCode="" 时不触发
		&refreshTokenCacheStub{},
		cfg,
		settingService,
		nil, // emailService
		nil, // turnstileService
		nil, // emailQueueService
		nil, // promoService
		nil, // defaultSubAssigner — nil, assignSubscriptions early return
		nil, // affiliateService — nil, bindOAuthAffiliate early return
		quotaRepo,
	)
}

func TestEmailOAuthAuto_SnapshotsPlatformQuotaDefaults(t *testing.T) {
	userRepo := &userRepoStub{nextID: 88}
	quotaRepo := &userPlatformQuotaRepoStub{}

	svc := newEmailOAuthAutoAuthService(
		userRepo,
		map[string]string{
			SettingKeyRegistrationEnabled:   "true",
			SettingKeyDefaultPlatformQuotas: `{"gemini": {"monthly": 100.0}}`,
		},
		quotaRepo,
	)

	user, err := svc.createEmailOAuthUser(
		context.Background(),
		"newoauth@example.com",
		"newoauth",
		"github",
		"", // invitationCode
		"", // affiliateCode
	)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(88), user.ID)

	require.Len(t, quotaRepo.bulkInsertCalls, 1, "createEmailOAuthUser must snapshot platform quotas via BulkInsertInitial")

	records := quotaRepo.bulkInsertCalls[0]
	var geminiRecord *UserPlatformQuotaRecord
	for i := range records {
		if records[i].Platform == "gemini" {
			geminiRecord = &records[i]
			break
		}
	}
	require.NotNil(t, geminiRecord, "expected gemini platform record")
	require.NotNil(t, geminiRecord.MonthlyLimitUSD)
	require.InDelta(t, 100.0, *geminiRecord.MonthlyLimitUSD, 0.0001)
}

func TestEmailOAuthAuto_AffiliateInvitationKeepsOfficialInviterBinding(t *testing.T) {
	rootID := int64(1)
	agentID := int64(2)
	ordinaryID := int64(3)
	userRepo := &userRepoStub{
		nextID: 89,
		usersByEmail: map[string]*User{
			"admin@example.com":    {ID: rootID, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
			"agent@example.com":    {ID: agentID, Email: "agent@example.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
			"ordinary@example.com": {ID: ordinaryID, Email: "ordinary@example.com", Role: RoleUser, ParentUserID: &agentID, Concurrency: 1, RPMLimit: 1, Status: StatusActive},
		},
	}
	affiliateRepo := &authAffiliateRepoStub{codeOwners: map[string]int64{"USERAFF": ordinaryID}}
	svc := newEmailOAuthAutoAuthService(
		userRepo,
		map[string]string{
			SettingKeyRegistrationEnabled:   "true",
			SettingKeyInvitationCodeEnabled: "true",
			SettingKeyAffiliateEnabled:      "true",
		},
		nil,
	)
	svc.affiliateService = NewAffiliateService(affiliateRepo, svc.settingService, nil, nil)
	agentRepo := newAgentManagementRepoStub(
		&User{ID: rootID, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
		&User{ID: agentID, Email: "agent@example.com", Role: RoleAgentLevel1, ParentUserID: &rootID, Status: StatusActive},
		&User{ID: ordinaryID, Email: "ordinary@example.com", Role: RoleUser, ParentUserID: &agentID, Concurrency: 1, RPMLimit: 1, Status: StatusActive},
	)
	agentRepo.agentProfiles = map[int64]AgentProfile{
		agentID: finiteAgentInviteProfile(agentID),
	}
	userRepo.onCreate = func(user *User) {
		clone := *user
		agentRepo.users[user.ID] = &clone
	}
	svc.SetAgentManagementService(NewAgentManagementService(agentRepo, userRepo, nil, nil))

	user, err := svc.createEmailOAuthUser(
		context.Background(),
		"oauth-affiliate@example.com",
		"oauth-affiliate",
		"github",
		"USERAFF",
		"",
	)

	require.NoError(t, err)
	require.NotNil(t, user.ParentUserID)
	require.Equal(t, agentID, *user.ParentUserID)
	require.Equal(t, []struct {
		userID    int64
		inviterID int64
	}{{userID: 89, inviterID: ordinaryID}}, affiliateRepo.bindCalls)
}
