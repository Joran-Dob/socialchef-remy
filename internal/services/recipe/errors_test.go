package recipe

import (
	"errors"
	"testing"

	apperrors "github.com/socialchef/remy/internal/errors"
)

func TestClassifyError_RateLimit(t *testing.T) {
	testCases := []string{
		"API error: status 429",
		"rate limit exceeded",
		"Rate Limit Error",
		"too many requests",
	}

	for _, tc := range testCases {
		err := errors.New(tc)
		providerErr := ClassifyError(err, "groq")

		if providerErr.Type != "rate_limit" {
			t.Errorf("Expected rate_limit for '%s', got %s", tc, providerErr.Type)
		}
		if providerErr.Provider != "groq" {
			t.Errorf("Expected provider 'groq', got %s", providerErr.Provider)
		}
	}
}

func TestClassifyError_CreditExhausted(t *testing.T) {
	testCases := []string{
		"API error: status 402",
		"insufficient credits",
		"Credit exhausted",
		"billing issue",
	}

	for _, tc := range testCases {
		err := errors.New(tc)
		providerErr := ClassifyError(err, "cerebras")

		if providerErr.Type != "credit_exhausted" {
			t.Errorf("Expected credit_exhausted for '%s', got %s", tc, providerErr.Type)
		}
	}
}

func TestClassifyError_ServerError(t *testing.T) {
	testCases := []string{
		"API error: status 500",
		"HTTP 503",
		"server error occurred",
		"Internal Server Error",
	}

	for _, tc := range testCases {
		err := errors.New(tc)
		providerErr := ClassifyError(err, "openai")

		if providerErr.Type != "server_error" {
			t.Errorf("Expected server_error for '%s', got %s", tc, providerErr.Type)
		}
	}
}

func TestClassifyError_ClientError(t *testing.T) {
	testCases := []string{
		"API error: status 400",
		"HTTP 401",
		"bad request",
		"Unauthorized",
	}

	for _, tc := range testCases {
		err := errors.New(tc)
		providerErr := ClassifyError(err, "groq")

		if providerErr.Type != "client_error" {
			t.Errorf("Expected client_error for '%s', got %s", tc, providerErr.Type)
		}
	}
}

func TestClassifyError_AppError(t *testing.T) {
	// Test with AppError (500 status)
	appErr := apperrors.NewTranscriptionError("server failed", "SERVER_ERROR", nil)
	providerErr := ClassifyError(appErr, "groq")

	if providerErr.Type != "server_error" {
		t.Errorf("Expected server_error for AppError with 500 status, got %s", providerErr.Type)
	}

	// Test with AppError (400 status)
	appErr2 := apperrors.NewValidationError("bad input", "BAD_INPUT", "")
	providerErr2 := ClassifyError(appErr2, "groq")

	if providerErr2.Type != "client_error" {
		t.Errorf("Expected client_error for AppError with 400 status, got %s", providerErr2.Type)
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	err := errors.New("some random error")
	providerErr := ClassifyError(err, "groq")

	if providerErr.Type != "unknown" {
		t.Errorf("Expected unknown for random error, got %s", providerErr.Type)
	}
}

func TestClassifyError_Nil(t *testing.T) {
	providerErr := ClassifyError(nil, "groq")

	if providerErr != nil {
		t.Errorf("Expected nil for nil error, got %v", providerErr)
	}
}

func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"rate limit", errors.New("status 429"), true},
		{"credit exhausted", errors.New("insufficient credits"), true},
		{"server error", errors.New("status 500"), true},
		{"client error", errors.New("status 400"), false},
		{"unknown error", errors.New("random"), false},
		{"nil error", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsRetryableError(tc.err)
			if result != tc.expected {
				t.Errorf("IsRetryableError(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}
