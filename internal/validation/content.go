package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Confidence represents certainty in the validation result
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// ContentValidationResult contains the outcome of validation
type ContentValidationResult struct {
	IsValid    bool       `json:"is_valid"`
	Confidence Confidence `json:"confidence"`
	Reason     string     `json:"reason"`
	Missing    []string   `json:"missing"`
}

// ContentValidationConfig defines settings for validation
type ContentValidationConfig struct {
	EnableAIValidation    bool
	ValidationModel       string
	MinDescriptionLength  int
	MinTranscriptLength   int
	RequireRecipeKeywords bool
}

// GroqClient interface for AI validation
type GroqClient interface {
	Chat(ctx context.Context, model string, messages []ChatMessage, responseFormat string) (string, error)
}

// ChatMessage represents a message in an AI chat completion
type ChatMessage struct {
	Role    string
	Content string
}

// recipeKeywords for quick heuristic validation
var recipeKeywords = []string{
	// Cooking verbs
	"bake", "cook", "fry", "boil", "grill", "roast", "saute", "simmer", "steam",
	"mix", "whisk", "stir", "blend", "chop", "dice", "slice", "preheat", "prepare",
	// Ingredients indicators
	"ingredient", "cup", "tablespoon", "teaspoon", "tbsp", "tsp", "ounce", "oz", "gram", "ml", "liter",
	// Recipe terms
	"recipe", "dish", "meal", "serve", "serving", "minutes", "hours", "temperature", "degrees",
	// Common ingredients
	"flour", "sugar", "salt", "pepper", "oil", "butter", "egg", "milk", "water", "garlic", "onion",
}

// QuickValidate performs a fast heuristic check without API calls
func QuickValidate(description, transcript string) ContentValidationResult {
	content := strings.TrimSpace(description + " " + transcript)

	// Thresholds: 30 for description, 50 for transcript
	hasMinimumLength := (len(description) >= 30) || (len(transcript) >= 50)

	if !hasMinimumLength {
		reason := fmt.Sprintf("Content too short (%d chars). Need at least 30 chars in description or 50 chars in transcript.", len(content))
		if len(content) == 0 {
			reason = "No content provided"
		}
		return ContentValidationResult{
			IsValid:    false,
			Confidence: ConfidenceHigh,
			Reason:     reason,
			Missing:    []string{"sufficient content length"},
		}
	}

	foundKeywords := false
	lowerContent := strings.ToLower(content)
	for _, kw := range recipeKeywords {
		if strings.Contains(lowerContent, kw) {
			foundKeywords = true
			break
		}
	}

	if !foundKeywords {
		return ContentValidationResult{
			IsValid:    true,
			Confidence: ConfidenceMedium,
			Reason:     "Content has sufficient length but no common recipe keywords found",
			Missing:    []string{"recipe keywords"},
		}
	}

	return ContentValidationResult{
		IsValid:    true,
		Confidence: ConfidenceHigh,
		Reason:     "Content passed quick validation",
		Missing:    []string{},
	}
}

// AIValidate uses an LLM to classify content
func AIValidate(ctx context.Context, description, transcript string, groqClient GroqClient, model string) (ContentValidationResult, error) {
	if groqClient == nil {
		return ContentValidationResult{}, fmt.Errorf("groq client is required for AI validation")
	}

	var contentBuilder strings.Builder
	if description != "" {
		contentBuilder.WriteString("Description: ")
		contentBuilder.WriteString(description)
		contentBuilder.WriteString("\n\n")
	}
	if transcript != "" {
		contentBuilder.WriteString("Transcript: ")
		contentBuilder.WriteString(transcript)
	}
	content := contentBuilder.String()

	validationPrompt := fmt.Sprintf(`Analyze if this social media content contains enough information to extract a recipe.

A valid recipe must have at least ONE of the following:
- Clear ingredients mentioned (e.g., "2 cups flour", "1 egg", "garlic")
- Cooking instructions or steps (e.g., "mix together", "bake for 20 minutes")
- Food preparation steps (e.g., "chop the onions", "heat the pan")

Content to analyze:
%s

Respond with ONLY a JSON object (no additional text):
{
  "has_recipe": true or false,
  "confidence": "high", "medium", or "low",
  "reason": "brief explanation why this does or doesn't contain recipe info",
  "missing": ["list", "of", "missing", "elements"]
}`, content)

	messages := []ChatMessage{
		{
			Role:    "system",
			Content: "You are a recipe content validator. Analyze content and respond with JSON only.",
		},
		{
			Role:    "user",
			Content: validationPrompt,
		},
	}

	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	resp, err := groqClient.Chat(ctx, model, messages, "json_object")
	if err != nil {
		return ContentValidationResult{
			IsValid:    false,
			Confidence: ConfidenceLow,
			Reason:     fmt.Sprintf("AI validation failed: %v", err),
			Missing:    []string{"ai validation"},
		}, err
	}

	var parsed struct {
		HasRecipe  bool     `json:"has_recipe"`
		Confidence string   `json:"confidence"`
		Reason     string   `json:"reason"`
		Missing    []string `json:"missing"`
	}

	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		return ContentValidationResult{
			IsValid:    false,
			Confidence: ConfidenceLow,
			Reason:     fmt.Sprintf("Failed to parse AI response: %v", err),
			Missing:    []string{"ai validation parsing"},
		}, err
	}

	return ContentValidationResult{
		IsValid:    parsed.HasRecipe,
		Confidence: Confidence(parsed.Confidence),
		Reason:     parsed.Reason,
		Missing:    parsed.Missing,
	}, nil
}

// ValidateContent decides which validation strategy to use
func ValidateContent(ctx context.Context, description, transcript string, config ContentValidationConfig, groqClient GroqClient, platform string) (ContentValidationResult, error) {
	quickResult := QuickValidate(description, transcript)

	if !quickResult.IsValid && quickResult.Confidence == ConfidenceHigh {
		return quickResult, nil
	}

	if !config.EnableAIValidation || groqClient == nil {
		return quickResult, nil
	}

	shouldRunAI := !quickResult.IsValid ||
		quickResult.Confidence == ConfidenceMedium ||
		(platform == "tiktok" && (description == "" || len(description) < 100))

	if shouldRunAI {
		aiResult, err := AIValidate(ctx, description, transcript, groqClient, config.ValidationModel)
		if err != nil {
			if quickResult.IsValid {
				return quickResult, nil
			}
			return aiResult, err
		}

		if aiResult.Confidence == ConfidenceHigh || (!quickResult.IsValid && aiResult.IsValid) {
			return aiResult, nil
		}

		if quickResult.IsValid {
			return quickResult, nil
		}

		return aiResult, nil
	}

	return quickResult, nil
}
