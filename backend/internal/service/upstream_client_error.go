package service

import (
	"net/http"
	"strings"
)

const safeDefaultUpstreamErrorMessage = "Upstream request failed"

// SafePassthroughClientMessage returns a client-facing message for matched
// passthrough rules. Raw upstream bodies are intentionally never exposed.
func SafePassthroughClientMessage(customMessage *string, fallback string) string {
	if customMessage != nil {
		if msg := strings.TrimSpace(*customMessage); msg != "" {
			return msg
		}
	}
	if msg := strings.TrimSpace(fallback); msg != "" {
		return msg
	}
	return safeDefaultUpstreamErrorMessage
}

// SafeUpstreamClientError returns an OpenAI/Anthropic-compatible error type and
// generic message for an upstream status code without leaking provider details.
func SafeUpstreamClientError(upstreamStatus int, defaultErrType, fallbackMessage string) (errType string, message string) {
	switch upstreamStatus {
	case http.StatusBadRequest:
		return "invalid_request_error", "Invalid request"
	case http.StatusUnauthorized:
		return "upstream_error", "Upstream authentication failed, please contact administrator"
	case http.StatusPaymentRequired:
		return "upstream_error", "Upstream payment required: insufficient balance or billing issue"
	case http.StatusForbidden:
		return "upstream_error", "Upstream access forbidden, please contact administrator"
	case http.StatusNotFound:
		return "not_found_error", "Resource not found"
	case http.StatusTooManyRequests:
		return "rate_limit_error", "Upstream rate limit exceeded, please retry later"
	case 529:
		return "overloaded_error", "Upstream service overloaded, please retry later"
	case http.StatusGatewayTimeout:
		return "timeout_error", "Upstream service temporarily unavailable"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return "upstream_error", "Upstream service temporarily unavailable"
	}

	errType = strings.TrimSpace(defaultErrType)
	if errType == "" {
		errType = "upstream_error"
	}
	message = strings.TrimSpace(fallbackMessage)
	if message == "" {
		message = safeDefaultUpstreamErrorMessage
	}
	return errType, message
}

// SafeUpstreamClientMessage returns only the sanitized client-facing message.
func SafeUpstreamClientMessage(upstreamStatus int, fallbackMessage string) string {
	_, msg := SafeUpstreamClientError(upstreamStatus, "upstream_error", fallbackMessage)
	return msg
}
