package recipe

import (
	"github.com/socialchef/remy/internal/config"
)

// NewProvider creates a new recipe provider based on the configuration
// It can optionally wrap the provider in a fallback wrapper if enabled
func NewProvider(cfg config.RecipeGenerationConfig, groqKey, cerebrasKey, openAIKey string) RecipeProvider {
	var primary RecipeProvider

	// Determine which provider to use as primary
	switch cfg.Provider {
	case "cerebras":
		primary = NewCerebrasProvider(cerebrasKey)
	case "openai":
		primary = NewOpenAIProvider(openAIKey)
	default:
		// Default to groq
		primary = NewGroqProvider(groqKey)
	}

	// If fallback is enabled, wrap the primary provider
	if cfg.FallbackEnabled {
		var secondary RecipeProvider

		// Determine which provider to use as fallback
		switch cfg.FallbackProvider {
		case "cerebras":
			secondary = NewCerebrasProvider(cerebrasKey)
		case "openai":
			secondary = NewOpenAIProvider(openAIKey)
		default:
			// Default to groq
			secondary = NewGroqProvider(groqKey)
		}

		return NewFallbackProvider(primary, secondary)
	}

	return primary
}
