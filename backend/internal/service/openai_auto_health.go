package service

import (
	"context"
	"net/http"
)

var openAIAutoHealthFailoverBody = []byte(`{"error":{"message":"Upstream request failed","type":"upstream_error"}}`)

func (s *OpenAIGatewayService) MaybeTempUnscheduleSlowFirstToken(ctx context.Context, account *Account, firstTokenMs *int, model string) bool {
	if s == nil || s.rateLimitService == nil {
		return false
	}
	return s.rateLimitService.MaybeTempUnscheduleSlowFirstToken(ctx, account, firstTokenMs, model)
}

func (s *OpenAIGatewayService) MaybeTempUnscheduleOpenAIRequestError(ctx context.Context, account *Account, message string) bool {
	if s == nil || s.rateLimitService == nil {
		return false
	}
	return s.rateLimitService.MaybeTempUnscheduleAutoHealthRequestError(ctx, account, message)
}

func (s *OpenAIGatewayService) newOpenAIRequestErrorFailover(account *Account) *UpstreamFailoverError {
	retryable := false
	if account != nil && account.IsPoolMode() {
		retryable = true
	}
	return &UpstreamFailoverError{
		StatusCode:             http.StatusBadGateway,
		ResponseBody:           openAIAutoHealthFailoverBody,
		RetryableOnSameAccount: retryable,
	}
}
