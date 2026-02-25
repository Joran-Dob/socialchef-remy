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
				"<REFERENCE_LISTS>",
				"<INGREDIENT_ANALYSIS>",
				"<CRITICAL_METRIC_REQUIREMENT>",
				"<INSTRUCTIONS>",
				"original_quantity",
				"original_unit",
				"quantity",
				"unit",
				"METRIC",
				"grams",
				"milliliters",
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
