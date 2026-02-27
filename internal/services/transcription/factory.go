package transcription

import (
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/errors"
)

// NewProvider creates a new transcription provider based on the configuration
// It can optionally wrap the provider in a fallback wrapper if enabled
func NewProvider(cfg config.TranscriptionConfig, openAIKey, groqKey string) TranscriptionProvider {
	var primary TranscriptionProvider

	// Determine which provider to use as primary
	switch cfg.Provider {
	case "openai":
		primary = NewOpenAIProvider(openAIKey)
	default:
		// Default to groq
		primary = NewGroqProvider(groqKey)
	}

	// If fallback is enabled, wrap the primary provider
	if cfg.FallbackEnabled {
		var secondary TranscriptionProvider

		// Determine which provider to use as fallback
		switch cfg.FallbackProvider {
		case "groq":
			secondary = NewGroqProvider(groqKey)
		default:
			// Default to openai
			secondary = NewOpenAIProvider(openAIKey)
		}

		return NewFallbackProvider(primary, secondary)
	}

	return primary
}

// isRetryableError checks if an error is retryable (5xx errors)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's an AppError with retryable status code
	if appErr, ok := err.(*errors.AppError); ok {
		return appErr.StatusCode >= 500
	}

	// Check for "status 5" in error message as fallback
	errorMsg := err.Error()
	return containsSubstring(errorMsg, "status 5") || containsSubstring(errorMsg, "HTTP 5")
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
