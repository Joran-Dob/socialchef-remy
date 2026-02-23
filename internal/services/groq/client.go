package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

type Ingredient struct {
	OriginalQuantity string  `json:"original_quantity"`
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

var ErrNoResponse = errors.New("no response from Groq")

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

const recipePrompt = `<ROLE>
You are a specialized AI assistant designed to parse recipe information from social media descriptions.
</ROLE>

<TASK>
Extract recipe information and output JSON.

<OUTPUT_FORMAT>
{
  "recipe": {"recipe_name": "", "description": "", "prep_time": null, "cooking_time": null, "total_time": null, "original_serving_size": null, "difficulty_rating": null, "focused_diet": "", "estimated_calories": null},
  "ingredients": [{"original_quantity": null, "original_unit": "", "quantity": null, "unit": "", "name": ""}],
  "instructions": [{"step_number": null, "instruction": ""}],
  "nutrition": {"protein": null, "carbs": null, "fat": null, "fiber": null},
  "cuisine_categories": [], "meal_types": [], "occasions": [], "dietary_restrictions": [], "equipment": []
}
</OUTPUT_FORMAT>
`

func getPlatformContext(platform string) string {
	switch platform {
	case "instagram":
		return `<PLATFORM_CONTEXT>Instagram post with detailed captions.</PLATFORM_CONTEXT>`
	case "tiktok":
		return `<PLATFORM_CONTEXT>TikTok video - rely on transcript.</PLATFORM_CONTEXT>`
	default:
		return ""
	}
}

func (c *Client) GenerateRecipe(ctx context.Context, description, transcript, platform string) (*Recipe, error) {
	systemPrompt := recipePrompt + getPlatformContext(platform)

	userContent := description
	if transcript != "" {
		userContent += "\n\nVideo Transcript:\n" + transcript
	}

	type chatRequest struct {
		Model       string `json:"model"`
		Messages    []struct {
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Groq API error: %s", string(respBody))
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
