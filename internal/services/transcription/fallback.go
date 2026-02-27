package transcription

import (
	"context"
	"log/slog"

	"github.com/socialchef/remy/internal/errors"
)

// FallbackProvider implements the TranscriptionProvider interface with fallback logic
type FallbackProvider struct {
	primary   TranscriptionProvider
	secondary TranscriptionProvider
}

// NewFallbackProvider creates a new fallback provider
func NewFallbackProvider(primary, secondary TranscriptionProvider) *FallbackProvider {
	return &FallbackProvider{
		primary:   primary,
		secondary: secondary,
	}
}

// Transcribe tries the primary provider first, falls back to secondary on 5xx errors
func (f *FallbackProvider) Transcribe(ctx context.Context, audioPath string) (string, error) {
	// Try primary provider first
	result, err := f.primary.Transcribe(ctx, audioPath)

	if err == nil {
		// Primary succeeded, return result
		return result, nil
	}

	// Check if error is retryable (5xx)
	if isRetryableError(err) {
		slog.Info("Primary provider failed with retryable error, attempting fallback",
			"primary_error", err.Error(),
			"audio_path", audioPath)

		// Try secondary provider
		result, fallbackErr := f.secondary.Transcribe(ctx, audioPath)
		if fallbackErr == nil {
			slog.Info("Fallback provider succeeded",
				"primary_error", err.Error(),
				"audio_path", audioPath)
			return result, nil
		} else {
			slog.Error("Both primary and secondary providers failed",
				"primary_error", err.Error(),
				"fallback_error", fallbackErr.Error(),
				"audio_path", audioPath)
return "", errors.NewTranscriptionError(
"both primary and secondary providers failed",
"PROVIDER_FALLBACK_FAILED",
err,
)
		}
	}

	// Not a retryable error (e.g., 4xx), return original error
	slog.Info("Primary provider failed with non-retryable error, not attempting fallback",
		"error", err.Error(),
		"audio_path", audioPath)
	return "", err
}
