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
			name:    "valid ingredient placeholder",
			text:    "Add {{ingredient:0}} to the pan",
			wantErr: false,
		},
		{
			name:    "valid timer placeholder",
			text:    "Cook for {{timer:5}} minutes",
			wantErr: false,
		},
		{
			name:    "multiple valid placeholders",
			text:    "Mix {{ingredient:0}} and {{ingredient:1}}, then {{timer:2}}",
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
			name:    "malformed type - won't match regex",
			text:    "Add {{abc:0}} to the pan",
			wantErr: false, // Regex won't match invalid types, so no error
		},
		{
			name:    "missing colon",
			text:    "Add {{ingredient0}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "missing index",
			text:    "Add {{ingredient:}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "non-numeric index",
			text:    "Add {{ingredient:abc}} to the pan",
			wantErr: false, // Regex won't match, so no error
		},
		{
			name:    "malformed braces",
			text:    "Add {ingredient:0} to the pan",
			wantErr: false, // Regex won't match, so no error
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
	tests := []struct {
		name            string
		text            string
		ingredientCount int
		timerCount      int
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "valid ingredient index",
			text:            "Add {{ingredient:0}} to the pan",
			ingredientCount: 3,
			timerCount:      0,
			wantErr:         false,
		},
		{
			name:            "valid ingredient index at max",
			text:            "Add {{ingredient:2}} to the pan",
			ingredientCount: 3,
			timerCount:      0,
			wantErr:         false,
		},
		{
			name:            "valid timer index",
			text:            "Cook for {{timer:5}} minutes",
			ingredientCount: 0,
			timerCount:      10,
			wantErr:         false,
		},
		{
			name:            "valid timer index at max",
			text:            "Cook for {{timer:9}} minutes",
			ingredientCount: 0,
			timerCount:      10,
			wantErr:         false,
		},
		{
			name:            "multiple valid placeholders",
			text:            "Mix {{ingredient:0}} and {{ingredient:2}}, then {{timer:5}}",
			ingredientCount: 3,
			timerCount:      10,
			wantErr:         false,
		},
		{
			name:            "no placeholders",
			text:            "Just regular text",
			ingredientCount: 3,
			timerCount:      5,
			wantErr:         false,
		},
		{
			name:            "ingredient index out of bounds - too high",
			text:            "Add {{ingredient:5}} to the pan",
			ingredientCount: 3,
			timerCount:      0,
			wantErr:         true,
			errMsg:          "ingredient index out of bounds",
		},
		{
			name:            "ingredient index out of bounds - negative",
			text:            "Add {{ingredient:-1}} to the pan",
			ingredientCount: 3,
			timerCount:      0,
			wantErr:         false, // Regex won't match negative, so no error
		},
		{
			name:            "timer index out of bounds - too high",
			text:            "Cook for {{timer:15}} minutes",
			ingredientCount: 0,
			timerCount:      10,
			wantErr:         true,
			errMsg:          "timer index out of bounds",
		},
		{
			name:            "timer index out of bounds - negative",
			text:            "Cook for {{timer:-1}} minutes",
			ingredientCount: 0,
			timerCount:      10,
			wantErr:         false, // Regex won't match negative, so no error
		},
		{
			name:            "zero ingredients - any index out of bounds",
			text:            "Add {{ingredient:0}} to the pan",
			ingredientCount: 0,
			timerCount:      0,
			wantErr:         true,
			errMsg:          "ingredient index out of bounds",
		},
		{
			name:            "zero timers - any index out of bounds",
			text:            "Cook for {{timer:0}} minutes",
			ingredientCount: 0,
			timerCount:      0,
			wantErr:         true,
			errMsg:          "timer index out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRichInstructionBounds(tt.text, tt.ingredientCount, tt.timerCount)
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
	// Test combined validation scenario
	text := "Mix {{ingredient:0}} and {{ingredient:2}}, then cook for {{timer:1}} minutes"

	// Format should be valid
	if err := ValidateRichInstructionFormat(text); err != nil {
		t.Errorf("Format validation failed: %v", err)
	}

	// Bounds should be valid with sufficient counts
	if err := ValidateRichInstructionBounds(text, 3, 2); err != nil {
		t.Errorf("Bounds validation failed: %v", err)
	}

	// Bounds should fail with insufficient ingredient count
	if err := ValidateRichInstructionBounds(text, 2, 2); err == nil {
		t.Error("Expected bounds error for ingredient:2 with only 2 ingredients")
	}

	// Bounds should fail with insufficient timer count
	if err := ValidateRichInstructionBounds(text, 3, 1); err == nil {
		t.Error("Expected bounds error for timer:1 with only 1 timer")
	}
}
