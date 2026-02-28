package recipe

import "context"

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderGroq     ProviderType = "groq"
	ProviderCerebras ProviderType = "cerebras"
	ProviderOpenAI   ProviderType = "openai"
)

// RecipeProvider defines the interface for AI recipe generation providers
type RecipeProvider interface {
	GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error)
}
