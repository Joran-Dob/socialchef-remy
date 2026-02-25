package validation

import (
	"strings"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestDetectPlaceholders(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"N/A", true},
		{"unknown", true},
		{"Not Specified", true},
		{"[placeholder]", true},
		{"<TBD>", true},
		{"valid ingredient", false},
		{"Salt", false},
		{"", true},
		{"   ", true},
		{"xxx", true},
	}

	for _, tt := range tests {
		result := DetectPlaceholders(tt.text)
		if result != tt.expected {
			t.Errorf("DetectPlaceholders(%q) = %v; want %v", tt.text, result, tt.expected)
		}
	}
}

func TestValidateRecipe(t *testing.T) {
	config := DefaultRecipeValidationConfig()

	t.Run("High quality recipe", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Classic Pancakes",
			Description: "A delicious and fluffy pancake recipe for your breakfast.",
			PrepTime:    intPtr(10),
			CookingTime: intPtr(15),
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: 2, Unit: "cups"},
				{Name: "Milk", Quantity: 1.5, Unit: "cups"},
				{Name: "Egg", Quantity: 1, Unit: "large"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix all ingredients in a large bowl until smooth."},
				{StepNumber: 2, Instruction: "Heat a non-stick pan over medium heat and pour batter."},
			},
			CuisineCategories: []string{"American"},
			MealTypes:         []string{"Breakfast"},
		}

		result := ValidateRecipe(recipe, config)
		if !result.IsValid {
			t.Errorf("Expected recipe to be valid, got issues: %v", result.Issues)
		}
		if result.QualityScore < 80 {
			t.Errorf("Expected high quality score, got %d", result.QualityScore)
		}
	})

	t.Run("Placeholder detection", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "N/A",
			Description: "TBD",
			Ingredients: []Ingredient{
				{Name: "unknown", Quantity: 0, Unit: ""},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "follow recipe"},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with placeholders to be invalid")
		}
		if !result.HasPlaceholders {
			t.Error("Expected HasPlaceholders to be true")
		}
	})

	t.Run("Empty recipe fails", func(t *testing.T) {
		recipe := Recipe{}
		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected empty recipe to be invalid")
		}
		if result.QualityScore != 0 {
			t.Errorf("Expected quality score 0, got %d", result.QualityScore)
		}
	})

	t.Run("Minimum requirements not met", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Simple Toast",
			Description: "Toast bread.",
			Ingredients: []Ingredient{
				{Name: "Bread", Quantity: 1, Unit: "slice"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Toast it."},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with too few ingredients/instructions to be invalid")
		}
		foundIngredientIssue := false
		foundInstructionIssue := false
		for _, issue := range result.Issues {
			if strings.Contains(issue, "Too few ingredients") {
				foundIngredientIssue = true
			}
			if strings.Contains(issue, "Too few instructions") {
				foundInstructionIssue = true
			}
		}
		if !foundIngredientIssue {
			t.Error("Expected ingredient count issue")
		}
		if !foundInstructionIssue {
			t.Error("Expected instruction count issue")
		}
	})
}
