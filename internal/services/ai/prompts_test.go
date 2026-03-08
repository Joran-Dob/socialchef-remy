package ai

import (
	"strings"
	"testing"
)

func TestBuildRecipePrompt(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		contains []string
	}{
		{
			name:     "Generic platform",
			platform: "",
			contains: []string{
				"<ROLE>",
				"<EXTRACTION_GUIDELINES>",
				"<INFERENCE>",
				"<OUTPUT_FORMAT>",
				"<INGREDIENT_ANALYSIS>",
				"<CRITICAL_METRIC_REQUIREMENT>",
				"<LANGUAGE_HANDLING>",
				"<INSTRUCTIONS>",
				"original_quantity",
				"original_unit",
				"quantity",
				"unit",
				"METRIC",
				"grams",
				"milliliters",
				"language",
				"ISO 639-1",
				"GENERATE a descriptive title",
				"Recipe Title Generation",
				"do NOT copy the source title verbatim",
			},
		},
		{
			name:     "Instagram platform",
			platform: "instagram",
			contains: []string{
				"<PLATFORM_CONTEXT>",
				"This recipe comes from Instagram",
				"Hashtags may indicate cuisine type",
			},
		},
		{
			name:     "TikTok platform",
			platform: "tiktok",
			contains: []string{
				"<PLATFORM_CONTEXT>",
				"This recipe comes from TikTok",
				"fast-paced with quick demonstrations",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildRecipePrompt(tt.platform)

			if len(prompt) == 0 {
				t.Errorf("BuildRecipePrompt() returned empty string")
			}

			// Verify line count (approximate, user requested >200 lines)
			lines := strings.Split(prompt, "\n")
			if len(lines) < 200 {
				t.Errorf("BuildRecipePrompt() returned %d lines, expected > 200", len(lines))
			}

			for _, s := range tt.contains {
				if !strings.Contains(prompt, s) {
					t.Errorf("BuildRecipePrompt() did not contain expected string: %s", s)
				}
			}
		})
	}
}

func TestGetPlatformContext(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Instagram",
			platform: "instagram",
			expected: "<PLATFORM_CONTEXT>",
		},
		{
			name:     "TikTok",
			platform: "tiktok",
			expected: "<PLATFORM_CONTEXT>",
		},
		{
			name:     "Unknown",
			platform: "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlatformContext(tt.platform)
			if tt.expected == "" {
				if result != "" {
					t.Errorf("getPlatformContext() = %v, expected empty string", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("getPlatformContext() = %v, expected it to contain %v", result, tt.expected)
				}
			}
		})
	}
}

func TestBuildFirecrawlPrompt(t *testing.T) {
	prompt := BuildFirecrawlPrompt()

	if len(prompt) == 0 {
		t.Errorf("BuildFirecrawlPrompt() returned empty string")
	}

	// Verify line count (approximate, should be similar to BuildRecipePrompt)
	lines := strings.Split(prompt, "\n")
	if len(lines) < 200 {
		t.Errorf("BuildFirecrawlPrompt() returned %d lines, expected > 200", len(lines))
	}

	// Verify Firecrawl-specific content
	expectedStrings := []string{
		"<PLATFORM_CONTEXT>",
		"website",
		"markdown",
		"schema.org",
		"JSON-LD",
		"<ROLE>",
		"<EXTRACTION_GUIDELINES>",
		"<INFERENCE>",
		"<OUTPUT_FORMAT>",
		"<INGREDIENT_ANALYSIS>",
		"<CRITICAL_METRIC_REQUIREMENT>",
		"<LANGUAGE_HANDLING>",
		"<INSTRUCTIONS>",
		"original_quantity",
		"original_unit",
		"quantity",
		"unit",
		"METRIC",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(prompt, s) {
			t.Errorf("BuildFirecrawlPrompt() did not contain expected string: %s", s)
		}
	}
}

func TestBuildPlaceholderPrompt(t *testing.T) {
	recipe := RecipeForPrompt{
		Name: "Chocolate Cake",
		Ingredients: []IngredientForPrompt{
			{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "flour"},
			{ID: "660e8400-e29b-41d4-a716-446655440001", Name: "sugar"},
			{ID: "770e8400-e29b-41d4-a716-446655440002", Name: "cocoa powder"},
		},
		Instructions: []InstructionForPrompt{
			{
				StepNumber:  1,
				Instruction: "Mix dry ingredients.",
				TimerData: []Timer{
					{DurationSeconds: 300, Label: "Mix", Type: "prep", Category: "active"},
				},
			},
		},
	}

	prompt := BuildPlaceholderPrompt(recipe)

	if len(prompt) == 0 {
		t.Errorf("BuildPlaceholderPrompt() returned empty string")
	}

	expectedStrings := []string{
		"<ROLE>",
		"<CONTEXT>",
		"Recipe Name: Chocolate Cake",
		"Ingredients (use these UUIDs for {{ingredient:UUID}} placeholders):",
		"[550e8400-e29b-41d4-a716-446655440000] flour",
		"[660e8400-e29b-41d4-a716-446655440001] sugar",
		"[770e8400-e29b-41d4-a716-446655440002] cocoa powder",
		"<OUTPUT_FORMAT>",
		"{{ingredient:550e8400-e29b-41d4-a716-446655440000}}",
		"{{timer:0}}",
		"36-character UUID format",
		"<INSTRUCTIONS>",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(prompt, s) {
			t.Errorf("BuildPlaceholderPrompt() did not contain expected string: %s", s)
		}
	}

	if strings.Contains(prompt, "{{ingredient:0}}") {
		t.Error("BuildPlaceholderPrompt() should not contain numeric ingredient index example")
	}

	if strings.Contains(prompt, "ingredient index") {
		t.Error("BuildPlaceholderPrompt() should reference UUIDs, not indices")
	}
}
