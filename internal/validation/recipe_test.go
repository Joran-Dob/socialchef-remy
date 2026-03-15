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
				{Name: "Flour", Quantity: "2", Unit: "cups"},
				{Name: "Milk", Quantity: "1.5", Unit: "cups"},
				{Name: "Egg", Quantity: "1", Unit: "large"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix all ingredients in a large bowl until smooth."},
				{StepNumber: 2, Instruction: "Heat a non-stick pan over medium heat and pour batter."},
				{StepNumber: 3, Instruction: "Cook until bubbles form on surface, then flip and cook until golden brown."},
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
				{Name: "unknown", Quantity: "", Unit: ""},
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
				{Name: "Bread", Quantity: "1", Unit: "slice"},
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

	t.Run("Valid split recipe with parts", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Layered Cake",
			Description: "A delicious multi-layered cake with frosting",
			PrepTime:    intPtr(30),
			CookingTime: intPtr(45),
			Parts: []RecipePart{
				{
					Name:         "Cake Layers",
					DisplayOrder: 0,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Flour", Quantity: "3", Unit: "cups"},
						{Name: "Sugar", Quantity: "2", Unit: "cups"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Mix flour and sugar in a large bowl"},
						{StepNumber: 2, Instruction: "Add eggs and milk, beat until smooth"},
					},
				},
				{
					Name:         "Frosting",
					DisplayOrder: 1,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Butter", Quantity: "1", Unit: "cup"},
						{Name: "Powdered Sugar", Quantity: "4", Unit: "cups"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Cream butter until smooth"},
						{StepNumber: 2, Instruction: "Gradually add powdered sugar"},
						{StepNumber: 3, Instruction: "Beat until light and fluffy"},
					},
				},
			},
			CuisineCategories: []string{"American"},
			MealTypes:         []string{"Dessert"},
		}

		result := ValidateRecipe(recipe, config)
		if !result.IsValid {
			t.Errorf("Expected split recipe to be valid, got issues: %v", result.Issues)
		}
		if result.QualityScore < 80 {
			t.Errorf("Expected high quality score, got %d", result.QualityScore)
		}
	})

	t.Run("Split recipe with optional part", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Pasta with Optional Sauce",
			Description: "Simple pasta with an optional sauce",
			PrepTime:    intPtr(10),
			CookingTime: intPtr(15),
			Parts: []RecipePart{
				{
					Name:         "Pasta",
					DisplayOrder: 0,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Pasta", Quantity: "1", Unit: "lb"},
						{Name: "Water", Quantity: "4", Unit: "cups"},
						{Name: "Salt", Quantity: "1", Unit: "tsp"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Boil water in a large pot"},
						{StepNumber: 2, Instruction: "Add salt and pasta, cook until al dente"},
						{StepNumber: 3, Instruction: "Drain the pasta and serve"},
					},
				},
				{
					Name:         "Optional Sauce",
					DisplayOrder: 1,
					IsOptional:   true,
					Ingredients:  []Ingredient{},
					Instructions: []Instruction{},
				},
			},
			CuisineCategories: []string{"Italian"},
			MealTypes:         []string{"Main Course"},
		}

		result := ValidateRecipe(recipe, config)
		if !result.IsValid {
			t.Errorf("Expected split recipe with optional part to be valid, got issues: %v", result.Issues)
		}
	})

	t.Run("Split recipe with invalid part", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Invalid Split Recipe",
			Description: "A split recipe with an invalid part",
			Parts: []RecipePart{
				{
					Name:         "", // Missing name
					DisplayOrder: 0,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Flour", Quantity: "1", Unit: "cup"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Mix ingredients"},
					},
				},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with invalid part to be invalid")
		}
	})

	t.Run("Split recipe with too few total ingredients", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Invalid Split Recipe",
			Description: "A split recipe with too few ingredients",
			Parts: []RecipePart{
				{
					Name:         "Part 1",
					DisplayOrder: 0,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Ingredient 1", Quantity: "1", Unit: "cup"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Do something"},
						{StepNumber: 2, Instruction: "Do something else"},
						{StepNumber: 3, Instruction: "Do a third thing"},
					},
				},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with too few ingredients to be invalid")
		}
	})

	t.Run("Split recipe with too few total instructions", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Invalid Split Recipe",
			Description: "A split recipe with too few instructions",
			Parts: []RecipePart{
				{
					Name:         "Part 1",
					DisplayOrder: 0,
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Ingredient 1", Quantity: "1", Unit: "cup"},
						{Name: "Ingredient 2", Quantity: "1", Unit: "cup"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Do something"},
					},
				},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with too few instructions to be invalid")
		}
	})

	t.Run("Split recipe with negative display order", func(t *testing.T) {
		recipe := Recipe{
			RecipeName:  "Invalid Split Recipe",
			Description: "A split recipe with negative display order",
			Parts: []RecipePart{
				{
					Name:         "Part 1",
					DisplayOrder: -1, // Negative display order
					IsOptional:   false,
					Ingredients: []Ingredient{
						{Name: "Ingredient 1", Quantity: "1", Unit: "cup"},
						{Name: "Ingredient 2", Quantity: "1", Unit: "cup"},
					},
					Instructions: []Instruction{
						{StepNumber: 1, Instruction: "Do something"},
						{StepNumber: 2, Instruction: "Do something else"},
						{StepNumber: 3, Instruction: "Do a third thing"},
					},
				},
			},
		}

		result := ValidateRecipe(recipe, config)
		if result.IsValid {
			t.Error("Expected recipe with negative display order to be invalid")
		}
	})
}

func TestValidateRecipePart(t *testing.T) {
	config := DefaultRecipeValidationConfig()

	t.Run("Valid part", func(t *testing.T) {
		part := RecipePart{
			Name:         "Cake Layer",
			DisplayOrder: 0,
			IsOptional:   false,
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: "2", Unit: "cups"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix ingredients well"},
			},
		}

		result := ValidateRecipePart(part, config)
		if !result.IsValid {
			t.Errorf("Expected part to be valid, got issues: %v", result.Issues)
		}
	})

	t.Run("Part with missing name", func(t *testing.T) {
		part := RecipePart{
			Name:         "",
			DisplayOrder: 0,
			IsOptional:   false,
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: "2", Unit: "cups"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix ingredients well"},
			},
		}

		result := ValidateRecipePart(part, config)
		if result.IsValid {
			t.Error("Expected part with missing name to be invalid")
		}
	})

	t.Run("Part with empty name", func(t *testing.T) {
		part := RecipePart{
			Name:         "   ",
			DisplayOrder: 0,
			IsOptional:   false,
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: "2", Unit: "cups"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix ingredients well"},
			},
		}

		result := ValidateRecipePart(part, config)
		if result.IsValid {
			t.Error("Expected part with empty name to be invalid")
		}
	})

	t.Run("Part with negative display order", func(t *testing.T) {
		part := RecipePart{
			Name:         "Test Part",
			DisplayOrder: -1,
			IsOptional:   false,
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: "2", Unit: "cups"},
			},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix ingredients well"},
			},
		}

		result := ValidateRecipePart(part, config)
		if result.IsValid {
			t.Error("Expected part with negative display order to be invalid")
		}
	})

	t.Run("Non-optional part without ingredients", func(t *testing.T) {
		part := RecipePart{
			Name:         "Test Part",
			DisplayOrder: 0,
			IsOptional:   false,
			Ingredients:  []Ingredient{},
			Instructions: []Instruction{
				{StepNumber: 1, Instruction: "Mix ingredients well"},
			},
		}

		result := ValidateRecipePart(part, config)
		if result.IsValid {
			t.Error("Expected non-optional part without ingredients to be invalid")
		}
	})

	t.Run("Non-optional part without instructions", func(t *testing.T) {
		part := RecipePart{
			Name:         "Test Part",
			DisplayOrder: 0,
			IsOptional:   false,
			Ingredients: []Ingredient{
				{Name: "Flour", Quantity: "2", Unit: "cups"},
			},
			Instructions: []Instruction{},
		}

		result := ValidateRecipePart(part, config)
		if result.IsValid {
			t.Error("Expected non-optional part without instructions to be invalid")
		}
	})

	t.Run("Optional part without ingredients and instructions", func(t *testing.T) {
		part := RecipePart{
			Name:         "Optional Garnish",
			DisplayOrder: 1,
			IsOptional:   true,
			Ingredients:  []Ingredient{},
			Instructions: []Instruction{},
		}

		result := ValidateRecipePart(part, config)
		if !result.IsValid {
			t.Errorf("Expected optional part to be valid even without ingredients/instructions, got issues: %v", result.Issues)
		}
	})
}
