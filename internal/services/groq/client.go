package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/recipe"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Type aliases for backward compatibility
type Recipe = recipe.Recipe
type RecipePart = recipe.RecipePart
type Ingredient = recipe.Ingredient
type Instruction = recipe.Instruction
type Nutrition = recipe.Nutrition
type StringOrNumber = recipe.StringOrNumber

type Client struct {
	apiKey string
}

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

var ErrNoResponse = errors.New("no response from Groq")

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

func (c *Client) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "groq"), attribute.String("operation", "generate_categories")}
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	type chatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}

	req := chatRequest{
		Model: "openai/gpt-oss-120b",
		ResponseFormat: struct {
			Type string `json:"type"`
		}{Type: "json_object"},
	}
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "system", Content: prompt})
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: "Categorize this recipe according to the instructions."})

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Groq"), "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("groq API error: %s", string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, ErrNoResponse
	}

	content := chatResp.Choices[0].Message.Content

	var result ai.CategoryAIResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GenerateRichInstructions(ctx context.Context, r *Recipe) (*recipe.RichInstructionResponse, error) {
	provider := recipe.NewGroqProvider(c.apiKey)
	return provider.GenerateRichInstructions(ctx, r)
}

func (c *Client) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "groq")}
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	var systemPrompt string
	if platform == "firecrawl" {
		systemPrompt = ai.BuildFirecrawlPrompt()
	} else {
		systemPrompt = ai.BuildRecipePrompt(platform)
	}

	userContent := description
	if transcript != "" {
		userContent += "\n\nVideo Transcript:\n" + transcript
	}

	type jsonSchema struct {
		Name   string                 `json:"name"`
		Schema map[string]interface{} `json:"schema"`
		Strict bool                   `json:"strict"`
	}

	type responseFormat struct {
		Type       string     `json:"type"`
		JSONSchema jsonSchema `json:"json_schema"`
	}

	type chatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat responseFormat `json:"response_format"`
	}

	// JSON Schema for structured recipe output with step-level ingredient extraction
	recipeJSONSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"recipe": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recipe_name":           map[string]interface{}{"type": "string"},
					"description":           map[string]interface{}{"type": "string"},
					"prep_time":             map[string]interface{}{"type": []string{"integer", "null"}},
					"cooking_time":          map[string]interface{}{"type": []string{"integer", "null"}},
					"total_time":            map[string]interface{}{"type": []string{"integer", "null"}},
					"original_serving_size": map[string]interface{}{"type": []string{"integer", "null"}},
					"difficulty_rating":     map[string]interface{}{"type": []string{"integer", "null"}},
					"focused_diet":          map[string]interface{}{"type": "string"},
					"estimated_calories":    map[string]interface{}{"type": []string{"integer", "null"}},
				},
			},
			"ingredients": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"original_quantity": map[string]interface{}{"type": []string{"string", "number", "null"}},
						"original_unit":     map[string]interface{}{"type": "string"},
						"quantity":          map[string]interface{}{"type": []string{"number", "null"}},
						"unit":              map[string]interface{}{"type": "string"},
						"name":              map[string]interface{}{"type": "string"},
					},
				},
			},
			"instructions": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"step_number": map[string]interface{}{"type": "integer"},
						"instruction": map[string]interface{}{"type": "string"},
						"timer_data": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"duration_seconds": map[string]interface{}{"type": []string{"integer", "null"}},
									"duration_text":    map[string]interface{}{"type": "string"},
									"label":            map[string]interface{}{"type": "string"},
									"type":             map[string]interface{}{"type": "string"},
									"category":         map[string]interface{}{"type": "string"},
								},
							},
						},
						"ingredients_used": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"ingredient_name": map[string]interface{}{"type": "string"},
									"quantity_used":   map[string]interface{}{"type": "string"},
								},
							},
						},
					},
				},
			},
			"nutrition": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"protein": map[string]interface{}{"type": []string{"number", "null"}},
					"carbs":   map[string]interface{}{"type": []string{"number", "null"}},
					"fat":     map[string]interface{}{"type": []string{"number", "null"}},
					"fiber":   map[string]interface{}{"type": []string{"number", "null"}},
				},
			},
			"language": map[string]interface{}{"type": "string"},
			"parts": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":          map[string]interface{}{"type": "string"},
						"description":   map[string]interface{}{"type": "string"},
						"display_order": map[string]interface{}{"type": "integer"},
						"is_optional":   map[string]interface{}{"type": "boolean"},
						"prep_time":     map[string]interface{}{"type": []string{"integer", "null"}},
						"cooking_time":  map[string]interface{}{"type": []string{"integer", "null"}},
						"ingredients": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"original_quantity": map[string]interface{}{"type": []string{"string", "number", "null"}},
									"original_unit":     map[string]interface{}{"type": "string"},
									"quantity":          map[string]interface{}{"type": []string{"number", "null"}},
									"unit":              map[string]interface{}{"type": "string"},
									"name":              map[string]interface{}{"type": "string"},
								},
							},
						},
						"instructions": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"step_number": map[string]interface{}{"type": "integer"},
									"instruction": map[string]interface{}{"type": "string"},
									"timer_data": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"duration_seconds": map[string]interface{}{"type": []string{"integer", "null"}},
												"duration_text":    map[string]interface{}{"type": "string"},
												"label":            map[string]interface{}{"type": "string"},
												"type":             map[string]interface{}{"type": "string"},
												"category":         map[string]interface{}{"type": "string"},
											},
										},
									},
									"ingredients_used": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"ingredient_name": map[string]interface{}{"type": "string"},
												"quantity_used":   map[string]interface{}{"type": "string"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	req := chatRequest{
		Model: "openai/gpt-oss-120b",
		ResponseFormat: responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchema{
				Name:   "recipe_extraction",
				Schema: recipeJSONSchema,
				Strict: false, // Use best-effort mode as recommended by Groq
			},
		},
	}
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "system", Content: systemPrompt})
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: userContent})

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Groq"), "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("groq API error: %s", string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, ErrNoResponse
	}

	content := chatResp.Choices[0].Message.Content

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
			if part.DisplayOrder == 0 {
				part.DisplayOrder = i + 1
			}
			for j := range part.Ingredients {
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
