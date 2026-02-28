package recipe

import (
	"context"
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
