package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// GroqProvider implements RecipeProvider for Groq API
type GroqProvider struct {
	apiKey string
}

// NewGroqProvider creates a new Groq recipe provider
func NewGroqProvider(apiKey string) *GroqProvider {
	return &GroqProvider{apiKey: apiKey}
}

// GenerateRecipe generates a recipe using Groq's API
func (p *GroqProvider) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "groq")}
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	systemPrompt := ai.BuildRecipePrompt(platform)

	userContent := description
	if transcript != "" {
		userContent += "\n\nVideo Transcript:\n" + transcript
	}

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
		Model: "llama-3.3-70b-versatile",
		ResponseFormat: struct {
			Type string `json:"type"`
		}{Type: "json_object"},
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
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
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
		return nil, fmt.Errorf("Groq API error (status %d): %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("no response from Groq")
	}

	content := chatResp.Choices[0].Message.Content

	var raw struct {
		Recipe              recipeResponseInner `json:"recipe"`
		Ingredients         []Ingredient        `json:"ingredients"`
		Instructions        []Instruction       `json:"instructions"`
		Nutrition           Nutrition           `json:"nutrition"`
		CuisineCategories   []string            `json:"cuisine_categories"`
		MealTypes           []string            `json:"meal_types"`
		Occasions           []string            `json:"occasions"`
		DietaryRestrictions []string            `json:"dietary_restrictions"`
		Equipment           []string            `json:"equipment"`
	}
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
