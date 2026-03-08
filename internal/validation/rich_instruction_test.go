package validation

import (
	"testing"
)

func TestValidateRichInstructionFormat(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid ingredient placeholder with UUID",
			text:    "Add {{ingredient:550e8400-e29b-41d4-a716-446655440000}} to the pan",
			wantErr: false,
		},
		{
			name:    "valid timer placeholder",
			text:    "Cook for {{timer:5}} minutes",
			wantErr: false,
		},
		{
			name:    "multiple valid placeholders",
			text:    "Mix {{ingredient:550e8400-e29b-41d4-a716-446655440000}} and {{ingredient:660e8400-e29b-41d4-a716-446655440001}}, then {{timer:2}}",
			wantErr: false,
		},
		{
			name:    "no placeholders",
			text:    "Just regular text with no placeholders",
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			wantErr: false,
		},
		{
			name:    "invalid ingredient format - numeric index instead of UUID",
			text:    "Add {{ingredient:0}} to the pan",
			wantErr: false, // Regex won't match invalid format, so no error
		},
		{
			name:    "invalid ingredient format - short UUID",
			text:    "Add {{ingredient:550e8400}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "missing colon",
			text:    "Add {{ingredient550e8400-e29b-41d4-a716-446655440000}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "missing UUID",
			text:    "Add {{ingredient:}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "malformed braces",
			text:    "Add {ingredient:550e8400-e29b-41d4-a716-446655440000} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "invalid timer index - non-numeric",
			text:    "Cook for {{timer:abc}} minutes",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "invalid timer - uppercase TIMER",
			text:    "Cook for {{TIMER:0}} minutes",
			wantErr: false, // Regex is case-sensitive, so no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRichInstructionFormat(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRichInstructionFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("ValidateRichInstructionFormat() error = %v, expected prefix %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateRichInstructionBounds(t *testing.T) {
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"660e8400-e29b-41d4-a716-446655440001",
		"770e8400-e29b-41d4-a716-446655440002",
	}

	tests := []struct {
		name                 string
		text                 string
		validIngredientUUIDs []string
		timerCount           int
		wantErr              bool
		errMsg               string
	}{
		{
			name:                 "valid ingredient UUID",
			text:                 "Add {{ingredient:550e8400-e29b-41d4-a716-446655440000}} to the pan",
			validIngredientUUIDs: validUUIDs,
			timerCount:           0,
			wantErr:              false,
		},
		{
			name:                 "valid ingredient UUID - second in list",
			text:                 "Add {{ingredient:660e8400-e29b-41d4-a716-446655440001}} to the pan",
			validIngredientUUIDs: validUUIDs,
			timerCount:           0,
			wantErr:              false,
		},
		{
			name:                 "valid timer index",
			text:                 "Cook for {{timer:5}} minutes",
			validIngredientUUIDs: validUUIDs,
			timerCount:           10,
			wantErr:              false,
		},
		{
			name:                 "valid timer index at max",
			text:                 "Cook for {{timer:9}} minutes",
			validIngredientUUIDs: validUUIDs,
			timerCount:           10,
			wantErr:              false,
		},
		{
			name:                 "multiple valid placeholders",
			text:                 "Mix {{ingredient:550e8400-e29b-41d4-a716-446655440000}} and {{ingredient:770e8400-e29b-41d4-a716-446655440002}}, then {{timer:5}}",
			validIngredientUUIDs: validUUIDs,
			timerCount:           10,
			wantErr:              false,
		},
		{
			name:                 "no placeholders",
			text:                 "Just regular text",
			validIngredientUUIDs: validUUIDs,
			timerCount:           5,
			wantErr:              false,
		},
		{
			name:                 "ingredient UUID not in valid list",
			text:                 "Add {{ingredient:999e8400-e29b-41d4-a716-446655440099}} to the pan",
			validIngredientUUIDs: validUUIDs,
			timerCount:           0,
			wantErr:              true,
			errMsg:               "ingredient UUID not found",
		},
		{
			name:                 "timer index out of bounds - too high",
			text:                 "Cook for {{timer:15}} minutes",
			validIngredientUUIDs: validUUIDs,
			timerCount:           10,
			wantErr:              true,
			errMsg:               "timer index out of bounds",
		},
		{
			name:                 "timer index out of bounds - negative (regex won't match)",
			text:                 "Cook for {{timer:-1}} minutes",
			validIngredientUUIDs: validUUIDs,
			timerCount:           10,
			wantErr:              false, // Regex won't match negative, so no error
		},
		{
			name:                 "empty valid UUID list - any ingredient UUID is invalid",
			text:                 "Add {{ingredient:550e8400-e29b-41d4-a716-446655440000}} to the pan",
			validIngredientUUIDs: []string{},
			timerCount:           0,
			wantErr:              true,
			errMsg:               "ingredient UUID not found",
		},
		{
			name:                 "zero timers - any index out of bounds",
			text:                 "Cook for {{timer:0}} minutes",
			validIngredientUUIDs: validUUIDs,
			timerCount:           0,
			wantErr:              true,
			errMsg:               "timer index out of bounds",
		},
		{
			name:                 "numeric ingredient index won't match UUID regex",
			text:                 "Add {{ingredient:0}} to the pan",
			validIngredientUUIDs: validUUIDs,
			timerCount:           0,
			wantErr:              false, // UUID regex won't match single digit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRichInstructionBounds(tt.text, tt.validIngredientUUIDs, tt.timerCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRichInstructionBounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("ValidateRichInstructionBounds() error = %v, expected prefix %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateRichInstructionFormatAndBounds(t *testing.T) {
	// Test combined validation scenario with UUID-based ingredients
	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"660e8400-e29b-41d4-a716-446655440001",
		"770e8400-e29b-41d4-a716-446655440002",
	}
	text := "Mix {{ingredient:550e8400-e29b-41d4-a716-446655440000}} and {{ingredient:770e8400-e29b-41d4-a716-446655440002}}, then cook for {{timer:1}} minutes"

	// Format should be valid
	if err := ValidateRichInstructionFormat(text); err != nil {
		t.Errorf("Format validation failed: %v", err)
	}

	// Bounds should be valid with all UUIDs present and sufficient timer count
	if err := ValidateRichInstructionBounds(text, validUUIDs, 2); err != nil {
		t.Errorf("Bounds validation failed: %v", err)
	}

	// Bounds should fail when first ingredient UUID is not in valid list
	invalidUUIDs := []string{
		"660e8400-e29b-41d4-a716-446655440001",
		"770e8400-e29b-41d4-a716-446655440002",
	}
	if err := ValidateRichInstructionBounds(text, invalidUUIDs, 2); err == nil {
		t.Error("Expected bounds error for missing ingredient UUID 550e8400-e29b-41d4-a716-446655440000")
	}

	// Bounds should fail with insufficient timer count
	if err := ValidateRichInstructionBounds(text, validUUIDs, 1); err == nil {
		t.Error("Expected bounds error for timer:1 with only 1 timer")
	}
}
