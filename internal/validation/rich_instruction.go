package validation

import (
	"fmt"
	"regexp"
	"strconv"
)

// Regex pattern to match rich instruction placeholders like {{ingredient:0}} or {{timer:5}}
var richInstructionPattern = regexp.MustCompile(`\{\{(ingredient|timer):(\d+)\}\}`)

// ValidateRichInstructionFormat validates that all placeholders in the text follow the correct Mustache syntax.
// Valid formats: {{ingredient:N}} or {{timer:N}} where N is a numeric index
// Returns an error if any placeholder has invalid format
func ValidateRichInstructionFormat(text string) error {
	matches := richInstructionPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil // No placeholders to validate
	}

	for _, match := range matches {
		placeholderType := match[1]
		indexStr := match[2]

		// Verify index is numeric (regex already ensures this, but double-check)
		_, err := strconv.Atoi(indexStr)
		if err != nil {
			return fmt.Errorf("invalid placeholder format: {{%s:%s}} - index must be numeric", placeholderType, indexStr)
		}

		// Verify type is valid (should be "ingredient" or "timer" due to regex)
		if placeholderType != "ingredient" && placeholderType != "timer" {
			return fmt.Errorf("invalid placeholder type: {{%s:%s}} - type must be 'ingredient' or 'timer'", placeholderType, indexStr)
		}
	}

	return nil
}

// ValidateRichInstructionBounds validates that all placeholder indices are within valid bounds.
// ingredientCount: number of ingredients available (valid indices: 0 to ingredientCount-1)
// timerCount: number of timers available (valid indices: 0 to timerCount-1)
// Returns an error if any index is out of bounds
func ValidateRichInstructionBounds(text string, ingredientCount, timerCount int) error {
	matches := richInstructionPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil // No placeholders to validate
	}

	for _, match := range matches {
		placeholderType := match[1]
		indexStr := match[2]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return fmt.Errorf("invalid index in placeholder {{%s:%s}}", placeholderType, indexStr)
		}

		switch placeholderType {
		case "ingredient":
			if index < 0 || index >= ingredientCount {
				return fmt.Errorf("ingredient index out of bounds: {{ingredient:%d}} (valid range: 0-%d)", index, ingredientCount-1)
			}
		case "timer":
			if index < 0 || index >= timerCount {
				return fmt.Errorf("timer index out of bounds: {{timer:%d}} (valid range: 0-%d)", index, timerCount-1)
			}
		}
	}

	return nil
}
