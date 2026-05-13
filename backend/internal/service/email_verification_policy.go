package service

import (
	"context"
	"strings"
)

var registrationEmailVerificationBypassDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
}

func extractRegistrationEmailDomain(email string) string {
	value := strings.TrimSpace(strings.ToLower(email))
	atIndex := strings.LastIndex(value, "@")
	if atIndex <= 0 || atIndex >= len(value)-1 {
		return ""
	}
	return value[atIndex+1:]
}

func isGmailLikeRegistrationEmail(email string) bool {
	_, ok := registrationEmailVerificationBypassDomains[extractRegistrationEmailDomain(email)]
	return ok
}

func (s *AuthService) shouldSkipRegistrationEmailVerification(ctx context.Context, email string) bool {
	if s == nil || s.settingService == nil || !s.settingService.IsGmailVerificationBypassEnabled(ctx) {
		return false
	}
	return isGmailLikeRegistrationEmail(email)
}
