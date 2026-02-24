package validation

import (
	"context"
	"testing"
)

type mockGroqClient struct {
	chatFunc func(ctx context.Context, model string, messages []ChatMessage, responseFormat string) (string, error)
}

func (m *mockGroqClient) Chat(ctx context.Context, model string, messages []ChatMessage, responseFormat string) (string, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, model, messages, responseFormat)
	}
	return "", nil
}

func TestQuickValidate(t *testing.T) {
	tests := []struct {
		name        string
		description string
		transcript  string
		wantIsValid bool
		wantConf    Confidence
	}{
		{
			name:        "Empty content",
			description: "",
			transcript:  "",
			wantIsValid: false,
			wantConf:    ConfidenceHigh,
		},
		{
			name:        "Short description only",
			description: "Too short",
			transcript:  "",
			wantIsValid: false,
			wantConf:    ConfidenceHigh,
		},
		{
			name:        "Sufficient description with keywords",
			description: "To make this cake, you need flour, sugar, and eggs. Bake for 30 minutes.",
			transcript:  "",
			wantIsValid: true,
			wantConf:    ConfidenceHigh,
		},
		{
			name:        "Sufficient description without keywords",
			description: "This is a long description that does not have any of the common things we are searching for in this test case.",
			transcript:  "",
			wantIsValid: true,
			wantConf:    ConfidenceMedium,
		},
		{
			name:        "Sufficient transcript",
			description: "",
			transcript:  "First, you want to chop the onions. Then saute them in a pan with some oil until they are soft.",
			wantIsValid: true,
			wantConf:    ConfidenceHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuickValidate(tt.description, tt.transcript)
			if got.IsValid != tt.wantIsValid {
				t.Errorf("QuickValidate() IsValid = %v, want %v", got.IsValid, tt.wantIsValid)
			}
			if got.Confidence != tt.wantConf {
				t.Errorf("QuickValidate() Confidence = %v, want %v", got.Confidence, tt.wantConf)
			}
		})
	}
}

func TestAIValidate(t *testing.T) {
	mock := &mockGroqClient{
		chatFunc: func(ctx context.Context, model string, messages []ChatMessage, responseFormat string) (string, error) {
			return `{"has_recipe": true, "confidence": "high", "reason": "Clear instructions and ingredients", "missing": []}`, nil
		},
	}

	res, err := AIValidate(context.Background(), "Cook this", "Instructions here", mock, "model")
	if err != nil {
		t.Fatalf("AIValidate failed: %v", err)
	}

	if !res.IsValid {
		t.Errorf("AIValidate() IsValid = %v, want true", res.IsValid)
	}
	if res.Confidence != ConfidenceHigh {
		t.Errorf("AIValidate() Confidence = %v, want high", res.Confidence)
	}
}

func TestValidateContent(t *testing.T) {
	config := ContentValidationConfig{
		EnableAIValidation: true,
		ValidationModel:    "test-model",
	}

	t.Run("Quick pass high confidence", func(t *testing.T) {
		res, err := ValidateContent(context.Background(), "Recipe: Cake. Ingredients: flour, sugar. Bake for 30 mins.", "", config, nil, "instagram")
		if err != nil {
			t.Fatalf("ValidateContent failed: %v", err)
		}
		if !res.IsValid || res.Confidence != ConfidenceHigh {
			t.Errorf("ValidateContent() expected valid high confidence, got %v (%v)", res.IsValid, res.Confidence)
		}
	})

	t.Run("Quick borderline, AI pass", func(t *testing.T) {
		mock := &mockGroqClient{
			chatFunc: func(ctx context.Context, model string, messages []ChatMessage, responseFormat string) (string, error) {
				return `{"has_recipe": true, "confidence": "high", "reason": "Found recipe", "missing": []}`, nil
			},
		}
		// Medium confidence because of sufficient length but no keywords (actually "Cake" is a keyword? No, I didn't add it)
		// Let's use a description that has length but no keywords from the list.
		desc := "This is a very long description that is just talking about random things but might be a recipe if we look closer at the transcript."
		res, err := ValidateContent(context.Background(), desc, "", config, mock, "instagram")
		if err != nil {
			t.Fatalf("ValidateContent failed: %v", err)
		}
		if !res.IsValid || res.Confidence != ConfidenceHigh {
			t.Errorf("ValidateContent() expected AI to return valid high confidence, got %v (%v)", res.IsValid, res.Confidence)
		}
	})
}
