package recipe

import (
	"testing"

	"github.com/socialchef/remy/internal/config"
)

func TestFactory_Groq(t *testing.T) {
	cfg := config.RecipeGenerationConfig{
		Provider:        "groq",
		FallbackEnabled: false,
	}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*GroqProvider); !ok {
		t.Errorf("Expected GroqProvider, got %T", provider)
	}
}

func TestFactory_Cerebras(t *testing.T) {
	cfg := config.RecipeGenerationConfig{
		Provider:        "cerebras",
		FallbackEnabled: false,
	}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*CerebrasProvider); !ok {
		t.Errorf("Expected CerebrasProvider, got %T", provider)
	}
}

func TestFactory_OpenAI(t *testing.T) {
	cfg := config.RecipeGenerationConfig{
		Provider:        "openai",
		FallbackEnabled: false,
	}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Errorf("Expected OpenAIProvider, got %T", provider)
	}
}

func TestFactory_Default(t *testing.T) {
	cfg := config.RecipeGenerationConfig{}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*GroqProvider); !ok {
		t.Errorf("Expected default GroqProvider, got %T", provider)
	}
}

func TestFactory_WithFallback(t *testing.T) {
	cfg := config.RecipeGenerationConfig{
		Provider:         "groq",
		FallbackEnabled:  true,
		FallbackProvider: "cerebras",
	}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*FallbackProvider); !ok {
		t.Errorf("Expected FallbackProvider, got %T", provider)
	}

	fallbackProvider := provider.(*FallbackProvider)
	if _, ok := fallbackProvider.Primary.(*GroqProvider); !ok {
		t.Errorf("Expected primary to be GroqProvider, got %T", fallbackProvider.Primary)
	}

	if _, ok := fallbackProvider.Secondary.(*CerebrasProvider); !ok {
		t.Errorf("Expected secondary to be CerebrasProvider, got %T", fallbackProvider.Secondary)
	}
}

func TestFactory_WithFallbackToOpenAI(t *testing.T) {
	cfg := config.RecipeGenerationConfig{
		Provider:         "cerebras",
		FallbackEnabled:  true,
		FallbackProvider: "openai",
	}

	provider := NewProvider(cfg, "test-groq-key", "test-cerebras-key", "test-openai-key")

	if _, ok := provider.(*FallbackProvider); !ok {
		t.Errorf("Expected FallbackProvider, got %T", provider)
	}

	fallbackProvider := provider.(*FallbackProvider)
	if _, ok := fallbackProvider.Primary.(*CerebrasProvider); !ok {
		t.Errorf("Expected primary to be CerebrasProvider, got %T", fallbackProvider.Primary)
	}

	if _, ok := fallbackProvider.Secondary.(*OpenAIProvider); !ok {
		t.Errorf("Expected secondary to be OpenAIProvider, got %T", fallbackProvider.Secondary)
	}
}
