package recipe

import (
	"context"
	"log/slog"

	"github.com/socialchef/remy/internal/errors"
	"github.com/socialchef/remy/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// FallbackProvider implements RecipeProvider with fallback logic
type FallbackProvider struct {
	primary   RecipeProvider
	secondary RecipeProvider
}

// NewFallbackProvider creates a new fallback provider
func NewFallbackProvider(primary, secondary RecipeProvider) *FallbackProvider {
	return &FallbackProvider{
		primary:   primary,
		secondary: secondary,
	}
}

// GenerateRecipe tries the primary provider first, falls back to secondary on retryable errors
func (f *FallbackProvider) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	// Try primary provider first
	result, err := f.primary.GenerateRecipe(ctx, description, transcript, platform)

	if err == nil {
		// Primary succeeded, return result
		return result, nil
	}

	// Classify the error
	providerErr := ClassifyError(err, "primary")

	// Check if error is retryable
	if IsRetryableError(err) {
		slog.Info("Primary provider failed with retryable error, attempting fallback",
			"error_type", providerErr.Type,
			"error", err.Error(),
			"platform", platform)

		// Record fallback metric
		metrics.ProviderFallbackTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("from_provider", providerErr.Provider),
			attribute.String("to_provider", "secondary"),
			attribute.String("reason", providerErr.Type),
		))

		// Try secondary provider
		result, fallbackErr := f.secondary.GenerateRecipe(ctx, description, transcript, platform)
		if fallbackErr == nil {
			slog.Info("Fallback provider succeeded",
				"primary_error_type", providerErr.Type,
				"platform", platform)
			return result, nil
		}

		// Both failed
		fallbackProviderErr := ClassifyError(fallbackErr, "secondary")
		slog.Error("Both primary and secondary providers failed",
			"primary_error_type", providerErr.Type,
			"primary_error", err.Error(),
			"fallback_error_type", fallbackProviderErr.Type,
			"fallback_error", fallbackErr.Error(),
			"platform", platform)

		return nil, errors.NewRecipeGenerationError(
			"both primary and secondary providers failed",
			"PROVIDER_FALLBACK_FAILED",
			err,
		)
	}

	// Not a retryable error (e.g., 4xx), return original error
	slog.Info("Primary provider failed with non-retryable error, not attempting fallback",
		"error_type", providerErr.Type,
		"error", err.Error(),
		"platform", platform)

	return nil, err
}
