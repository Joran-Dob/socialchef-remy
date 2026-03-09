package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/validation"
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
		Model: "openai/gpt-oss-120b",
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

	// Debug logging for ingredient quantities
	var debugResp recipeResponseOuter
	if err := json.Unmarshal([]byte(content), &debugResp); err == nil {
		if len(debugResp.Ingredients) > 0 {
			fmt.Printf("[DEBUG] Recipe: %s, Servings: %d\n", debugResp.Recipe.RecipeName, debugResp.Recipe.OriginalServings)
			for i, ing := range debugResp.Ingredients[:min(3, len(debugResp.Ingredients))] {
				fmt.Printf("[DEBUG] Ingredient %d: %s - original: %s %s, per-serving: %s %s\n",
					i, ing.Name, ing.OriginalQuantity, ing.OriginalUnit, ing.Quantity, ing.Unit)
			}
		}
	}

	var raw recipeResponseOuter
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
		Language:            raw.Language,
	}, nil
}

// GenerateCategories generates category suggestions using Groq's API
func (p *GroqProvider) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
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

	var result ai.CategoryAIResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *GroqProvider) GenerateRichInstructions(ctx context.Context, recipe *Recipe) (*RichInstructionResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "groq"), attribute.String("operation", "generate_rich_instructions")}
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	recipeForPrompt := ai.RecipeForPrompt{
		Name: recipe.RecipeName,
	}
	validIngredientUUIDs := make([]string, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		ingID := ing.ID
		if ingID == "" {
			ingID = uuid.New().String()
		}
		validIngredientUUIDs = append(validIngredientUUIDs, ingID)
		recipeForPrompt.Ingredients = append(recipeForPrompt.Ingredients, ai.IngredientForPrompt{
			ID:   ingID,
			Name: ing.Name,
		})
	}
	for _, inst := range recipe.Instructions {
		var timers []ai.Timer
		for _, td := range inst.TimerData {
			if td.DurationSeconds > 0 {
				timers = append(timers, ai.Timer{
					DurationSeconds: td.DurationSeconds,
					DurationText:    td.DurationText,
					Label:           td.Label,
					Type:            td.Type,
					Category:        td.Category,
				})
			}
		}
		recipeForPrompt.Instructions = append(recipeForPrompt.Instructions, ai.InstructionForPrompt{
			StepNumber:  inst.StepNumber,
			Instruction: inst.Instruction,
			TimerData:   timers,
		})
	}

	systemPrompt := ai.BuildPlaceholderPrompt(recipeForPrompt)

	recipeJSON, err := json.Marshal(recipe)
	if err != nil {
		return nil, ClassifyError(fmt.Errorf("failed to marshal recipe: %w", err), "groq")
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
		Model: "openai/gpt-oss-120b",
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
	}{Role: "user", Content: string(recipeJSON)})

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Groq"), "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, ClassifyError(err, "groq")
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, ClassifyError(err, "groq")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ClassifyError(err, "groq")
	}

	if resp.StatusCode >= 400 {
		return nil, ClassifyError(fmt.Errorf("Groq API error (status %d): %s", resp.StatusCode, string(respBody)), "groq")
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, ClassifyError(fmt.Errorf("failed to unmarshal API response: %w", err), "groq")
	}

	if len(chatResp.Choices) == 0 {
		return nil, ClassifyError(fmt.Errorf("no response from Groq"), "groq")
	}

	content := chatResp.Choices[0].Message.Content

	var result RichInstructionResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, ClassifyError(fmt.Errorf("failed to unmarshal rich instructions: %w", err), "groq")
	}

	timerCountByStep := make(map[int]int)
	for _, inst := range recipe.Instructions {
		timerCountByStep[inst.StepNumber] = len(inst.TimerData)
	}

	for _, inst := range result.Instructions {
		if err := validation.ValidateRichInstructionFormat(inst.InstructionRich); err != nil {
			return nil, ClassifyError(fmt.Errorf("invalid placeholder format in step %d: %w", inst.StepNumber, err), "groq")
		}
		timerCount := timerCountByStep[inst.StepNumber]
		if err := validation.ValidateRichInstructionBounds(inst.InstructionRich, validIngredientUUIDs, timerCount); err != nil {
			return nil, ClassifyError(fmt.Errorf("placeholder out of bounds in step %d: %w", inst.StepNumber, err), "groq")
		}
	}

	result.PromptVersion = ai.RichInstructionPromptVersion

	return &result, nil
}
