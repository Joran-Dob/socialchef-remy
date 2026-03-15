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

// CerebrasProvider implements RecipeProvider for Cerebras API
type CerebrasProvider struct {
	apiKey string
}

// NewCerebrasProvider creates a new Cerebras recipe provider
func NewCerebrasProvider(apiKey string) *CerebrasProvider {
	return &CerebrasProvider{apiKey: apiKey}
}

// GenerateRecipe generates a recipe using Cerebras's API
func (p *CerebrasProvider) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "cerebras")}
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
		Model: "gpt-oss-120b",
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
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Cerebras"), "POST", "https://api.cerebras.ai/v1/chat/completions", bytes.NewReader(body))
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
		return nil, fmt.Errorf("cerebras API error (status %d): %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("no response from Cerebras")
	}

	content := chatResp.Choices[0].Message.Content

	var raw recipeResponseOuter
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

// GenerateCategories generates category suggestions using Cerebras's API
func (p *CerebrasProvider) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{
			attribute.String("provider", "cerebras"),
			attribute.String("operation", "generate_categories"),
		}
		metrics.AIGenerationDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	// Build JSON schema for structured outputs with strict mode
	// Cerebras requires additionalProperties: false on ALL objects
	categorySchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"cuisine_categories": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"meal_types": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"occasions": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"dietary_restrictions": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"equipment": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"new_category_suggestions": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cuisine_categories": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"meal_types": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"occasions": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"dietary_restrictions": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
					"equipment": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []string{"cuisine_categories", "meal_types", "occasions", "dietary_restrictions", "equipment"},
		"additionalProperties": false,
	}

	type chatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type       string                 `json:"type"`
			JSONSchema map[string]interface{} `json:"json_schema"`
		} `json:"response_format"`
	}

	req := chatRequest{
		Model: "gpt-oss-120b",
	}
	req.ResponseFormat.Type = "json_schema"
	req.ResponseFormat.JSONSchema = map[string]interface{}{
		"name":   "category_response",
		"strict": true,
		"schema": categorySchema,
	}
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "system", Content: prompt})
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: "Categorize this recipe according to the instructions."})

	body, err := json.Marshal(req)
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}

	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Cerebras"), "POST", "https://api.cerebras.ai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}

	if resp.StatusCode >= 400 {
		return nil, ClassifyError(fmt.Errorf("Cerebras API error (status %d): %s", resp.StatusCode, string(respBody)), "cerebras")
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, ClassifyError(err, "cerebras")
	}

	if len(chatResp.Choices) == 0 {
		return nil, ClassifyError(fmt.Errorf("no response from Cerebras"), "cerebras")
	}

	content := chatResp.Choices[0].Message.Content

	var result ai.CategoryAIResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, ClassifyError(err, "cerebras")
	}

	return &result, nil
}

// GenerateRichInstructions generates rich instructions with placeholders using Cerebras's API
func (p *CerebrasProvider) GenerateRichInstructions(ctx context.Context, recipe *Recipe) (*RichInstructionResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{
			attribute.String("provider", "cerebras"),
			attribute.String("operation", "generate_rich_instructions"),
		}
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

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"instructions": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"step_number": map[string]interface{}{
							"type": "integer",
						},
						"instruction_rich": map[string]interface{}{
							"type": "string",
						},
					},
					"required":             []string{"step_number", "instruction_rich"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"instructions"},
		"additionalProperties": false,
	}

	type chatRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type       string                 `json:"type"`
			JSONSchema map[string]interface{} `json:"json_schema"`
		} `json:"response_format"`
	}

	req := chatRequest{
		Model: "gpt-oss-120b",
	}
	req.ResponseFormat.Type = "json_schema"
	req.ResponseFormat.JSONSchema = map[string]interface{}{
		"name":   "rich_instruction_response",
		"strict": true,
		"schema": schema,
	}
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "system", Content: systemPrompt})
	req.Messages = append(req.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: "Generate rich instructions with ingredient and timer placeholders for the recipe above."})

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "Cerebras"), "POST", "https://api.cerebras.ai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ClassifyError(err, "cerebras")
	}

	if resp.StatusCode >= 400 {
		return nil, ClassifyError(fmt.Errorf("Cerebras API error (status %d): %s", resp.StatusCode, string(respBody)), "cerebras")
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, ClassifyError(fmt.Errorf("failed to unmarshal API response: %w", err), "cerebras")
	}

	if len(chatResp.Choices) == 0 {
		return nil, ClassifyError(fmt.Errorf("no response from Cerebras"), "cerebras")
	}

	content := chatResp.Choices[0].Message.Content

	var result RichInstructionResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, ClassifyError(fmt.Errorf("failed to unmarshal rich instructions: %w", err), "cerebras")
	}

	timerCountByStep := make(map[int]int)
	for _, inst := range recipe.Instructions {
		timerCountByStep[inst.StepNumber] = len(inst.TimerData)
	}

	for _, inst := range result.Instructions {
		if err := validation.ValidateRichInstructionFormat(inst.InstructionRich); err != nil {
			return nil, ClassifyError(fmt.Errorf("invalid placeholder format in step %d: %w", inst.StepNumber, err), "cerebras")
		}
		timerCount := timerCountByStep[inst.StepNumber]
		if err := validation.ValidateRichInstructionBounds(inst.InstructionRich, validIngredientUUIDs, timerCount); err != nil {
			return nil, ClassifyError(fmt.Errorf("placeholder out of bounds in step %d: %w", inst.StepNumber, err), "cerebras")
		}
	}

	result.PromptVersion = ai.RichInstructionPromptVersion

	return &result, nil
}
