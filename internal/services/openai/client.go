package openai

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/socialchef/remy/internal/services/ai"
)

type Client struct {
	apiKey string
}

type Recipe struct {
	RecipeName          string
	Description         string
	PrepTime            *int
	CookingTime         *int
	TotalTime           *int
	OriginalServings    *int
	DifficultyRating    *int
	FocusedDiet         string
	EstimatedCalories   *int
	Ingredients         []Ingredient
	Instructions        []Instruction
	Nutrition           Nutrition
	CuisineCategories   []string
	MealTypes           []string
	Occasions           []string
	DietaryRestrictions []string
	Equipment           []string
}

// StringOrNumber can unmarshal from JSON string or number
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = ""
		return nil
	}
	// Try unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrNumber(str)
		return nil
	}
	// Try as number
	var num float64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = StringOrNumber(strconv.FormatFloat(num, 'f', -1, 64))
	return nil
}

type Ingredient struct {
	OriginalQuantity StringOrNumber `json:"original_quantity"`
	OriginalUnit     string  `json:"original_unit"`
	Quantity         float64 `json:"quantity"`
	Unit             string  `json:"unit"`
	Name             string  `json:"name"`
}

type Instruction struct {
	StepNumber  int    `json:"step_number"`
	Instruction string `json:"instruction"`
}

type Nutrition struct {
	Protein float64 `json:"protein"`
	Carbs   float64 `json:"carbs"`
	Fat     float64 `json:"fat"`
	Fiber   float64 `json:"fiber"`
}

type recipeResponse struct {
	Recipe              RecipeResponseInner `json:"recipe"`
	Ingredients         []Ingredient        `json:"ingredients"`
	Instructions        []Instruction       `json:"instructions"`
	Nutrition           Nutrition           `json:"nutrition"`
	CuisineCategories   []string            `json:"cuisine_categories"`
	MealTypes           []string            `json:"meal_types"`
	Occasions           []string            `json:"occasions"`
	DietaryRestrictions []string            `json:"dietary_restrictions"`
	Equipment           []string            `json:"equipment"`
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

func generateRecipeWithOpenAI(ctx context.Context, apiKey, description, transcript, platform string) (*Recipe, error) {
	systemPrompt := ai.BuildRecipePrompt(platform)
	userContent := description
	if transcript != "" {
		userContent += "\n\nVideo Transcript:\n" + transcript
	}

	content, err := callOpenAIChat(ctx, apiKey, "gpt-3.5-turbo-1106", systemPrompt, userContent, true)
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
		Ingredients:         raw.Ingredients,
		Instructions:        raw.Instructions,
		Nutrition:           raw.Nutrition,
		CuisineCategories:   raw.CuisineCategories,
		MealTypes:           raw.MealTypes,
		Occasions:           raw.Occasions,
		DietaryRestrictions: raw.DietaryRestrictions,
		Equipment:           raw.Equipment,
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
