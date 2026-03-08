package recipe

import (
	"context"

	"github.com/socialchef/remy/internal/services/ai"
)

// GroqClientAdapter wraps a RecipeProvider to implement the GroqClient interface
// This provides backward compatibility with existing code that expects the GroqClient interface
type GroqClientAdapter struct {
	provider RecipeProvider
}

// NewGroqClientAdapter creates a new adapter that wraps a RecipeProvider
func NewGroqClientAdapter(provider RecipeProvider) *GroqClientAdapter {
	return &GroqClientAdapter{provider: provider}
}

// GenerateRecipe delegates to the wrapped RecipeProvider
// This method signature matches the existing GroqClient interface in worker/handlers.go
func (a *GroqClientAdapter) GenerateRecipe(ctx context.Context, caption, transcript, platform string) (*Recipe, error) {
	return a.provider.GenerateRecipe(ctx, caption, transcript, platform)
}

func (a *GroqClientAdapter) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	if catProvider, ok := a.provider.(interface {
		GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error)
	}); ok {
		return catProvider.GenerateCategories(ctx, prompt)
	}
	return &ai.CategoryAIResponse{}, nil
}
