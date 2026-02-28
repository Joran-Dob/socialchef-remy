package recipe

import (
	"strings"

	"github.com/socialchef/remy/internal/errors"
)

// ProviderError represents a classified error from an AI provider
type ProviderError struct {
	Type     string // "rate_limit", "credit_exhausted", "server_error", "client_error", "unknown"
	Message  string
	Provider string
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return e.Message
}

// ClassifyError analyzes an error and returns a ProviderError with classification
func ClassifyError(err error, provider string) *ProviderError {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Check for rate limit (429)
	if containsSubstring(msg, "status 429") ||
		containsSubstring(msg, "HTTP 429") ||
		containsSubstring(msg, "rate limit") ||
		containsSubstring(msg, "Rate limit") ||
		containsSubstring(msg, "too many requests") {
		return &ProviderError{
			Type:     "rate_limit",
			Message:  msg,
			Provider: provider,
		}
	}

	// Check for credit exhaustion (402 or credit-related messages)
	if containsSubstring(msg, "status 402") ||
		containsSubstring(msg, "HTTP 402") ||
		containsSubstring(msg, "insufficient credit") ||
		containsSubstring(msg, "Insufficient credit") ||
		containsSubstring(msg, "credit exhausted") ||
		containsSubstring(msg, "Credit exhausted") ||
		containsSubstring(msg, "billing") ||
		containsSubstring(msg, "Billing") {
		return &ProviderError{
			Type:     "credit_exhausted",
			Message:  msg,
			Provider: provider,
		}
	}

	// Check for AppError with status code
	if appErr, ok := err.(*errors.AppError); ok {
		if appErr.StatusCode >= 500 {
			return &ProviderError{
				Type:     "server_error",
				Message:  msg,
				Provider: provider,
			}
		}
		if appErr.StatusCode >= 400 {
			return &ProviderError{
				Type:     "client_error",
				Message:  msg,
				Provider: provider,
			}
		}
	}

	// Check for server errors (5xx) in message
	if containsSubstring(msg, "status 5") ||
		containsSubstring(msg, "HTTP 5") ||
		containsSubstring(msg, "server error") ||
		containsSubstring(msg, "Server error") ||
		containsSubstring(msg, "internal error") ||
		containsSubstring(msg, "Internal error") {
		return &ProviderError{
			Type:     "server_error",
			Message:  msg,
			Provider: provider,
		}
	}

	// Check for client errors (4xx) in message
	if containsSubstring(msg, "status 4") ||
		containsSubstring(msg, "HTTP 4") ||
		containsSubstring(msg, "bad request") ||
		containsSubstring(msg, "Bad request") ||
		containsSubstring(msg, "unauthorized") ||
		containsSubstring(msg, "Unauthorized") ||
		containsSubstring(msg, "forbidden") ||
		containsSubstring(msg, "Forbidden") {
		return &ProviderError{
			Type:     "client_error",
			Message:  msg,
			Provider: provider,
		}
	}

	// Default to unknown
	return &ProviderError{
		Type:     "unknown",
		Message:  msg,
		Provider: provider,
	}
}

// IsRetryableError returns true if the error is retryable (rate limit, credit exhausted, or server error)
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Try to classify the error
	providerErr := ClassifyError(err, "")
	if providerErr == nil {
		return false
	}

	// Retryable error types
	switch providerErr.Type {
	case "rate_limit", "credit_exhausted", "server_error":
		return true
	default:
		return false
	}
}

// containsSubstring checks if a string contains a substring (case-insensitive)
func containsSubstring(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
