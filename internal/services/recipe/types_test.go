package recipe

import (
	"testing"
)

func TestRecipe_HasParts(t *testing.T) {
	tests := []struct {
		name     string
		recipe   *Recipe
		expected bool
	}{
		{
			name: "Recipe with parts",
			recipe: &Recipe{
				Parts: []RecipePart{
					{ID: "part1", Name: "Part 1"},
				},
			},
			expected: true,
		},
		{
			name: "Recipe without parts",
			recipe: &Recipe{
				Parts: []RecipePart{},
			},
			expected: false,
		},
		{
			name: "Recipe with nil parts",
			recipe: &Recipe{
				Parts: nil,
			},
			expected: false,
		},
		{
			name:     "Recipe with empty parts slice",
			recipe:   &Recipe{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.recipe.HasParts()
			if result != tt.expected {
				t.Errorf("HasParts() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRecipe_FlattenIngredients(t *testing.T) {
	partID1 := "part1"
	partID2 := "part2"

	tests := []struct {
		name     string
		recipe   *Recipe
		expected []Ingredient
	}{
		{
			name: "Recipe with parts",
			recipe: &Recipe{
				Ingredients: []Ingredient{
					{ID: "ing1", Name: "Flour"},
				},
				Parts: []RecipePart{
					{
						ID: partID1,
						Ingredients: []Ingredient{
							{ID: "ing2", Name: "Sugar", PartID: &partID1},
							{ID: "ing3", Name: "Eggs", PartID: &partID1},
						},
					},
					{
						ID: partID2,
						Ingredients: []Ingredient{
							{ID: "ing4", Name: "Milk", PartID: &partID2},
						},
					},
				},
			},
			expected: []Ingredient{
				{ID: "ing2", Name: "Sugar", PartID: &partID1},
				{ID: "ing3", Name: "Eggs", PartID: &partID1},
				{ID: "ing4", Name: "Milk", PartID: &partID2},
			},
		},
		{
			name: "Recipe without parts",
			recipe: &Recipe{
				Ingredients: []Ingredient{
					{ID: "ing1", Name: "Flour"},
					{ID: "ing2", Name: "Sugar"},
				},
				Parts: []RecipePart{},
			},
			expected: []Ingredient{
				{ID: "ing1", Name: "Flour"},
				{ID: "ing2", Name: "Sugar"},
			},
		},
		{
			name: "Recipe with nil parts",
			recipe: &Recipe{
				Ingredients: []Ingredient{
					{ID: "ing1", Name: "Flour"},
				},
				Parts: nil,
			},
			expected: []Ingredient{
				{ID: "ing1", Name: "Flour"},
			},
		},
		{
			name: "Recipe with empty ingredients and parts",
			recipe: &Recipe{
				Ingredients: []Ingredient{},
				Parts:       []RecipePart{},
			},
			expected: []Ingredient{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.recipe.FlattenIngredients()
			if len(result) != len(tt.expected) {
				t.Errorf("FlattenIngredients() returned %d ingredients, expected %d", len(result), len(tt.expected))
				return
			}
			for i, ing := range result {
				if ing.ID != tt.expected[i].ID || ing.Name != tt.expected[i].Name {
					t.Errorf("Ingredient %d: got %+v, expected %+v", i, ing, tt.expected[i])
				}
			}
		})
	}
}

func TestRecipe_FlattenInstructions(t *testing.T) {
	partID1 := "part1"
	partID2 := "part2"

	tests := []struct {
		name     string
		recipe   *Recipe
		expected []Instruction
	}{
		{
			name: "Recipe with parts",
			recipe: &Recipe{
				Instructions: []Instruction{
					{StepNumber: 1, Instruction: "Preheat oven"},
				},
				Parts: []RecipePart{
					{
						ID: partID1,
						Instructions: []Instruction{
							{StepNumber: 2, Instruction: "Mix ingredients", PartID: &partID1},
							{StepNumber: 3, Instruction: "Bake cake", PartID: &partID1},
						},
					},
					{
						ID: partID2,
						Instructions: []Instruction{
							{StepNumber: 4, Instruction: "Make frosting", PartID: &partID2},
						},
					},
				},
			},
			expected: []Instruction{
				{StepNumber: 2, Instruction: "Mix ingredients", PartID: &partID1},
				{StepNumber: 3, Instruction: "Bake cake", PartID: &partID1},
				{StepNumber: 4, Instruction: "Make frosting", PartID: &partID2},
			},
		},
		{
			name: "Recipe without parts",
			recipe: &Recipe{
				Instructions: []Instruction{
					{StepNumber: 1, Instruction: "Preheat oven"},
					{StepNumber: 2, Instruction: "Mix ingredients"},
				},
				Parts: []RecipePart{},
			},
			expected: []Instruction{
				{StepNumber: 1, Instruction: "Preheat oven"},
				{StepNumber: 2, Instruction: "Mix ingredients"},
			},
		},
		{
			name: "Recipe with nil parts",
			recipe: &Recipe{
				Instructions: []Instruction{
					{StepNumber: 1, Instruction: "Preheat oven"},
				},
				Parts: nil,
			},
			expected: []Instruction{
				{StepNumber: 1, Instruction: "Preheat oven"},
			},
		},
		{
			name: "Recipe with empty instructions and parts",
			recipe: &Recipe{
				Instructions: []Instruction{},
				Parts:        []RecipePart{},
			},
			expected: []Instruction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.recipe.FlattenInstructions()
			if len(result) != len(tt.expected) {
				t.Errorf("FlattenInstructions() returned %d instructions, expected %d", len(result), len(tt.expected))
				return
			}
			for i, inst := range result {
				if inst.StepNumber != tt.expected[i].StepNumber || inst.Instruction != tt.expected[i].Instruction {
					t.Errorf("Instruction %d: got %+v, expected %+v", i, inst, tt.expected[i])
				}
			}
		})
	}
}
