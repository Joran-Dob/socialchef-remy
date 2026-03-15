package openai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/recipe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Client struct {
	apiKey string
}

// Type aliases for backward compatibility
type Recipe = recipe.Recipe
type RecipePart = recipe.RecipePart
type Ingredient = recipe.Ingredient
type Instruction = recipe.Instruction
type Nutrition = recipe.Nutrition
type StringOrNumber = recipe.StringOrNumber

type recipeResponse struct {
	Recipe              RecipeResponseInner `json:"recipe"`
	Ingredients         []Ingredient        `json:"ingredients"`
	Instructions        []Instruction       `json:"instructions"`
	Parts               []RecipePart        `json:"parts,omitempty"`
	Nutrition           Nutrition           `json:"nutrition"`
	CuisineCategories   []string            `json:"cuisine_categories"`
	MealTypes           []string            `json:"meal_types"`
	Occasions           []string            `json:"occasions"`
	DietaryRestrictions []string            `json:"dietary_restrictions"`
	Equipment           []string            `json:"equipment"`
	Language            string              `json:"language"`
}

type RecipeResponseInner struct {
	RecipeName        string `json:"recipe_name"`
	Description       string `json:"description"`
	PrepTime          *int   `json:"prep_time"`
	CookingTime       *int   `json:"cooking_time"`
	TotalTime         *int   `json:"total_time"`
	OriginalServings  *int   `json:"original_serving_size"`
	DifficultyRating  *int   `json:"difficulty_rating"`
	FocusedDiet       string `json:"focused_diet"`
	EstimatedCalories *int   `json:"estimated_calories"`
}

var (
	ErrNoResponse  = errors.New("no response from OpenAI")
	ErrNoEmbedding = errors.New("no embedding returned")
)

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	return generateRecipeWithOpenAI(ctx, c.apiKey, description, transcript, platform)
}

func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return generateEmbeddingWithOpenAI(ctx, c.apiKey, text)
}

// Complete sends a completion request to OpenAI for general text completion
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	content, err := callOpenAIChat(ctx, c.apiKey, "gpt-4o-mini", "", prompt, false, 100, 0.3)
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", ErrNoResponse
	}
	return strings.TrimSpace(content), nil
}

func generateRecipeWithOpenAI(ctx context.Context, apiKey, description, transcript, platform string) (*Recipe, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attribute.String("provider", "openai")))
	}()

	systemPrompt := ai.BuildRecipePrompt(platform)
	userContent := description
	if transcript != "" {
		userContent += "\n\nVideo Transcript:\n" + transcript
	}

	content, err := callOpenAIChat(ctx, apiKey, "gpt-3.5-turbo-1106", systemPrompt, userContent, true, 0, 0)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, ErrNoResponse
	}

	var raw recipeResponse
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, err
	}

	var ingredients []Ingredient
	var instructions []Instruction
	var parts []RecipePart

	if len(raw.Parts) > 0 {
		parts = raw.Parts
		for i := range parts {
			part := &parts[i]
			if part.ID == "" {
				part.ID = uuid.New().String()
			}
			if part.DisplayOrder == 0 {
				part.DisplayOrder = i + 1
			}
			for j := range part.Ingredients {
				if part.Ingredients[j].ID == "" {
					part.Ingredients[j].ID = uuid.New().String()
				}
				part.Ingredients[j].PartID = &part.ID
			}
			for j := range part.Instructions {
				part.Instructions[j].PartID = &part.ID
			}
			ingredients = append(ingredients, part.Ingredients...)
			instructions = append(instructions, part.Instructions...)
		}
	} else {
		ingredients = raw.Ingredients
		instructions = raw.Instructions
	}

	return &Recipe{
		RecipeName:          raw.Recipe.RecipeName,
		Description:         raw.Recipe.Description,
		PrepTime:            raw.Recipe.PrepTime,
		CookingTime:         raw.Recipe.CookingTime,
		TotalTime:           raw.Recipe.TotalTime,
		OriginalServings:    raw.Recipe.OriginalServings,
		DifficultyRating:    raw.Recipe.DifficultyRating,
		FocusedDiet:         raw.Recipe.FocusedDiet,
		EstimatedCalories:   raw.Recipe.EstimatedCalories,
		Ingredients:         ingredients,
		Instructions:        instructions,
		Parts:               parts,
		Nutrition:           raw.Nutrition,
		CuisineCategories:   raw.CuisineCategories,
		MealTypes:           raw.MealTypes,
		Occasions:           raw.Occasions,
		DietaryRestrictions: raw.DietaryRestrictions,
		Equipment:           raw.Equipment,
		Language:            raw.Language,
	}, nil
}

func generateEmbeddingWithOpenAI(ctx context.Context, apiKey, text string) ([]float32, error) {
	embedding, err := callOpenAIEmbedding(ctx, apiKey, "text-embedding-ada-002", text)
	if err != nil {
		return nil, err
	}
	if len(embedding) == 0 {
		return nil, ErrNoEmbedding
	}
	return embedding, nil
}
