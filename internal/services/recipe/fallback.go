package recipe

import (
	"context"
	"log/slog"

	"github.com/socialchef/remy/internal/errors"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// FallbackProvider implements RecipeProvider with fallback logic
type FallbackProvider struct {
	Primary   RecipeProvider
	Secondary RecipeProvider
}

// NewFallbackProvider creates a new fallback provider
func NewFallbackProvider(primary, secondary RecipeProvider) *FallbackProvider {
	return &FallbackProvider{
		Primary:   primary,
		Secondary: secondary,
	}
}

// GenerateRecipe tries the primary provider first, falls back to secondary on retryable errors
func (f *FallbackProvider) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	// Try primary provider first
	result, err := f.Primary.GenerateRecipe(ctx, description, transcript, platform)

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
		result, fallbackErr := f.Secondary.GenerateRecipe(ctx, description, transcript, platform)
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

// GenerateCategories tries the primary provider first, falls back to secondary on retryable errors
func (f *FallbackProvider) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	// Check if primary implements GenerateCategories
	catProvider, primaryOk := f.Primary.(interface {
		GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error)
	})

	if primaryOk {
		// Try primary provider first
		result, err := catProvider.GenerateCategories(ctx, prompt)

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
				"operation", "generate_categories")

			// Record fallback metric
			metrics.ProviderFallbackTotal.Add(ctx, 1, metric.WithAttributes(
				attribute.String("from_provider", providerErr.Provider),
				attribute.String("to_provider", "secondary"),
				attribute.String("reason", providerErr.Type),
				attribute.String("operation", "generate_categories"),
			))

			// Try secondary provider
			return f.trySecondaryCategories(ctx, prompt)
		}

		// Not a retryable error (e.g., 4xx), return original error
		slog.Info("Primary provider failed with non-retryable error, not attempting fallback",
			"error_type", providerErr.Type,
			"error", err.Error(),
			"operation", "generate_categories")

		return nil, err
	}

	// Primary doesn't implement GenerateCategories, try secondary directly
	return f.trySecondaryCategories(ctx, prompt)
}

// trySecondaryCategories attempts to generate categories using the secondary provider
func (f *FallbackProvider) trySecondaryCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	// Check if secondary implements GenerateCategories
	if catProvider, ok := f.Secondary.(interface {
		GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error)
	}); ok {
		result, err := catProvider.GenerateCategories(ctx, prompt)
		if err == nil {
			slog.Info("Fallback provider succeeded for categories",
				"operation", "generate_categories")
			return result, nil
		}

		// Secondary also failed
		fallbackProviderErr := ClassifyError(err, "secondary")
		slog.Error("Both primary and secondary providers failed for categories",
			"fallback_error_type", fallbackProviderErr.Type,
			"fallback_error", err.Error(),
			"operation", "generate_categories")

		// Graceful degradation: return empty response, not error
		return &ai.CategoryAIResponse{}, nil
	}

	// Secondary doesn't implement GenerateCategories either
	slog.Info("Neither provider implements GenerateCategories, returning empty response",
		"operation", "generate_categories")
	return &ai.CategoryAIResponse{}, nil
}

// GenerateRichInstructions tries the primary provider first, falls back to secondary on retryable errors
func (f *FallbackProvider) GenerateRichInstructions(ctx context.Context, recipe *Recipe) (*RichInstructionResponse, error) {
	// Check if primary implements RichInstructionProvider
	if richProvider, ok := f.Primary.(RichInstructionProvider); ok {
		// Try primary provider first
		result, err := richProvider.GenerateRichInstructions(ctx, recipe)

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
				"operation", "generate_rich_instructions")

			// Record fallback metric
			metrics.ProviderFallbackTotal.Add(ctx, 1, metric.WithAttributes(
				attribute.String("from_provider", providerErr.Provider),
				attribute.String("to_provider", "secondary"),
				attribute.String("reason", providerErr.Type),
				attribute.String("operation", "generate_rich_instructions"),
			))

			// Try secondary provider
			return f.trySecondaryRichInstructions(ctx, recipe)
		}

		// Not a retryable error (e.g., 4xx), return original error
		slog.Info("Primary provider failed with non-retryable error, not attempting fallback",
			"error_type", providerErr.Type,
			"error", err.Error(),
			"operation", "generate_rich_instructions")

		return nil, err
	}

	// Primary doesn't implement RichInstructionProvider, try secondary directly
	return f.trySecondaryRichInstructions(ctx, recipe)
}

// trySecondaryRichInstructions attempts to generate rich instructions using the secondary provider
func (f *FallbackProvider) trySecondaryRichInstructions(ctx context.Context, recipe *Recipe) (*RichInstructionResponse, error) {
	// Check if secondary implements RichInstructionProvider
	if richProvider, ok := f.Secondary.(RichInstructionProvider); ok {
		result, err := richProvider.GenerateRichInstructions(ctx, recipe)
		if err == nil {
			slog.Info("Fallback provider succeeded for rich instructions",
				"operation", "generate_rich_instructions")
			return result, nil
		}

		// Secondary also failed
		fallbackProviderErr := ClassifyError(err, "secondary")
		slog.Error("Both primary and secondary providers failed for rich instructions",
			"fallback_error_type", fallbackProviderErr.Type,
			"fallback_error", err.Error(),
			"operation", "generate_rich_instructions")

		// Graceful degradation: return nil, nil (not error)
		return nil, nil
	}

	// Secondary doesn't implement RichInstructionProvider either
	slog.Info("Neither provider implements RichInstructionProvider, returning nil",
		"operation", "generate_rich_instructions")
	return nil, nil
}
