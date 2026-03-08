package validation

import (
	"fmt"
	"regexp"
	"strconv"
)

// Regex pattern to match ingredient placeholders with UUIDs: {{ingredient:550e8400-e29b-41d4-a716-446655440000}}
var ingredientPlaceholderPattern = regexp.MustCompile(`\{\{ingredient:([a-f0-9-]{36})\}\}`)

// Regex pattern to match timer placeholders with numeric indices: {{timer:0}}
var timerPlaceholderPattern = regexp.MustCompile(`\{\{timer:(\d+)\}\}`)

// ValidateRichInstructionFormat validates that all placeholders in the text follow the correct Mustache syntax.
// Valid formats: {{ingredient:UUID}} where UUID is a 36-character UUID, or {{timer:N}} where N is a numeric index
// Returns an error if any placeholder has invalid format
func ValidateRichInstructionFormat(text string) error {
	// Validate ingredient placeholders with UUIDs
	ingMatches := ingredientPlaceholderPattern.FindAllStringSubmatch(text, -1)
	for _, match := range ingMatches {
		uuidStr := match[1]
		if !isValidUUIDFormat(uuidStr) {
			return fmt.Errorf("invalid ingredient placeholder format: {{ingredient:%s}} - must be a valid 36-character UUID", uuidStr)
		}
	}

	// Validate timer placeholders with numeric indices
	timerMatches := timerPlaceholderPattern.FindAllStringSubmatch(text, -1)
	for _, match := range timerMatches {
		indexStr := match[1]
		_, err := strconv.Atoi(indexStr)
		if err != nil {
			return fmt.Errorf("invalid timer placeholder format: {{timer:%s}} - index must be numeric", indexStr)
		}
	}

	return nil
}

// isValidUUIDFormat checks if a string is a valid UUID format (36 characters with hyphens)
func isValidUUIDFormat(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check positions of hyphens (8-4-4-4-12 format)
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check all other characters are hexadecimal
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphen positions
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ValidateRichInstructionBounds validates that all placeholder indices are within valid bounds.
// validIngredientUUIDs: list of valid ingredient UUIDs that can be referenced
// timerCount: number of timers available (valid indices: 0 to timerCount-1)
// Returns an error if any ingredient UUID is not in the valid list or timer index is out of bounds
func ValidateRichInstructionBounds(text string, validIngredientUUIDs []string, timerCount int) error {
	// Build a set for O(1) lookup of valid ingredient UUIDs
	validUUIDSet := make(map[string]bool, len(validIngredientUUIDs))
	for _, uuid := range validIngredientUUIDs {
		validUUIDSet[uuid] = true
	}

	// Validate ingredient placeholders
	ingMatches := ingredientPlaceholderPattern.FindAllStringSubmatch(text, -1)
	for _, match := range ingMatches {
		uuidStr := match[1]
		if !validUUIDSet[uuidStr] {
			return fmt.Errorf("ingredient UUID not found: {{ingredient:%s}}", uuidStr)
		}
	}

	// Validate timer placeholders
	timerMatches := timerPlaceholderPattern.FindAllStringSubmatch(text, -1)
	for _, match := range timerMatches {
		indexStr := match[1]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return fmt.Errorf("invalid timer index: {{timer:%s}}", indexStr)
		}
		if index < 0 || index >= timerCount {
			return fmt.Errorf("timer index out of bounds: {{timer:%d}} (valid range: 0-%d)", index, timerCount-1)
		}
	}

	return nil
}
