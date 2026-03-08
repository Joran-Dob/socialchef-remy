package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// mockDB implements CategoryDBQueries for testing
type mockDB struct {
	cuisineCategories   []string
	mealTypes           []string
	occasions           []string
	dietaryRestrictions []string
	equipment           []string
	err                 error
}

func (m *mockDB) GetCuisineCategoriesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.cuisineCategories, nil
}

func (m *mockDB) GetMealTypesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.mealTypes, nil
}

func (m *mockDB) GetOccasionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.occasions, nil
}

func (m *mockDB) GetDietaryRestrictionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dietaryRestrictions, nil
}

func (m *mockDB) GetEquipmentByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.equipment, nil
}

// mockAI implements AIClient for testing
type mockAI struct {
	response *CategoryAIResponse
	err      error
}

func (m *mockAI) GenerateCategories(ctx context.Context, prompt string) (*CategoryAIResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// TestSuggestCategories_WithExistingCategories tests successful category suggestion with existing categories
func TestSuggestCategories_WithExistingCategories(t *testing.T) {
	// Setup mocks with existing categories
	mockDB := &mockDB{
		cuisineCategories:   []string{"Italian", "Mexican", "Asian"},
		mealTypes:           []string{"Dinner", "Lunch"},
		occasions:           []string{"Weeknight", "Special Occasion"},
		dietaryRestrictions: []string{"Vegetarian"},
		equipment:           []string{"Oven", "Stovetop"},
	}

	mockAI := &mockAI{
		response: &CategoryAIResponse{
			CuisineCategories:   []string{"Italian"},
			MealTypes:           []string{"Dinner"},
			Occasions:           []string{"Weeknight"},
			DietaryRestrictions: []string{"Vegetarian"},
			Equipment:           []string{"Oven", "Stovetop"},
		},
	}

	service := NewCategoryService(mockDB, mockAI)

	// Create test recipe with realistic ingredients
	recipe := RecipeInfo{
		Name:        "Classic Margherita Pizza",
		Description: "A traditional Italian pizza with fresh tomatoes, mozzarella, and basil",
		Ingredients: []string{"pizza dough", "tomato sauce", "fresh mozzarella", "fresh basil", "olive oil"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Execute
	result, err := service.SuggestCategories(ctx, recipe, userID)

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}

	if len(result.CuisineCategories) == 0 {
		t.Error("expected at least one cuisine category")
	}

	if len(result.MealTypes) == 0 {
		t.Error("expected at least one meal type")
	}

	// Verify categories are from existing list
	validCuisines := map[string]bool{"Italian": true, "Mexican": true, "Asian": true}
	for _, cuisine := range result.CuisineCategories {
		if !validCuisines[cuisine] {
			t.Errorf("cuisine %s not in existing categories", cuisine)
		}
	}
}

// TestSuggestCategories_NoExistingCategories tests fallback behavior when user has no categories
func TestSuggestCategories_NoExistingCategories(t *testing.T) {
	// Setup empty mocks (no existing categories)
	mockDB := &mockDB{
		cuisineCategories:   []string{},
		mealTypes:           []string{},
		occasions:           []string{},
		dietaryRestrictions: []string{},
		equipment:           []string{},
	}

	mockAI := &mockAI{
		response: &CategoryAIResponse{
			CuisineCategories:   []string{"Italian"},
			MealTypes:           []string{"Dinner"},
			Occasions:           []string{"Weeknight"},
			DietaryRestrictions: []string{},
			Equipment:           []string{"Oven"},
			NewCategorySuggestions: &NewCategorySet{
				CuisineCategories:   []string{"Italian"},
				MealTypes:           []string{"Dinner"},
				Occasions:           []string{},
				DietaryRestrictions: []string{},
				Equipment:           []string{},
			},
		},
	}

	service := NewCategoryService(mockDB, mockAI)

	// Create test recipe with pasta, tomatoes, basil
	recipe := RecipeInfo{
		Name:        "Pasta Primavera",
		Description: "Fresh pasta with seasonal vegetables",
		Ingredients: []string{"pasta", "tomatoes", "basil", "zucchini", "bell peppers", "olive oil"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Execute
	result, err := service.SuggestCategories(ctx, recipe, userID)

	// Verify
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}

	// Should return categories even though user has none
	if len(result.CuisineCategories) == 0 {
		t.Error("expected at least one cuisine category from fallback")
	}

	if len(result.MealTypes) == 0 {
		t.Error("expected at least one meal type from fallback")
	}
}

// TestSuggestCategories_AIError tests error handling when AI fails
func TestSuggestCategories_AIError(t *testing.T) {
	// Setup mocks
	mockDB := &mockDB{
		cuisineCategories:   []string{"Italian"},
		mealTypes:           []string{"Dinner"},
		occasions:           []string{},
		dietaryRestrictions: []string{},
		equipment:           []string{},
	}

	aiError := errors.New("AI service unavailable")
	mockAI := &mockAI{
		err: aiError,
	}

	service := NewCategoryService(mockDB, mockAI)

	// Create test recipe
	recipe := RecipeInfo{
		Name:        "Test Recipe",
		Description: "Test description",
		Ingredients: []string{"pasta", "tomatoes", "basil"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Execute
	result, err := service.SuggestCategories(ctx, recipe, userID)

	// Verify
	if err == nil {
		t.Fatal("expected AI error to be returned, got nil")
	}

	if !strings.Contains(err.Error(), "failed to generate categories") {
		t.Errorf("expected error message to contain 'failed to generate categories', got %v", err)
	}

	if result != nil {
		t.Error("expected result to be nil on AI error")
	}
}

// TestSuggestCategories_DBError tests error handling when DB fails
func TestSuggestCategories_DBError(t *testing.T) {
	// Setup mock with DB error
	dbError := errors.New("database connection failed")
	mockDB := &mockDB{
		err: dbError,
	}

	mockAI := &mockAI{
		response: &CategoryAIResponse{
			CuisineCategories: []string{},
		},
	}

	service := NewCategoryService(mockDB, mockAI)

	// Create test recipe
	recipe := RecipeInfo{
		Name:        "Test Recipe",
		Description: "Test description",
		Ingredients: []string{"pasta"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Execute
	result, err := service.SuggestCategories(ctx, recipe, userID)

	// Verify
	if err == nil {
		t.Fatal("expected DB error to be returned, got nil")
	}

	if !strings.Contains(err.Error(), "failed to fetch existing categories") {
		t.Errorf("expected error message to contain 'failed to fetch existing categories', got %v", err)
	}

	if result != nil {
		t.Error("expected result to be nil on DB error")
	}
}

// TestSuggestCategories_InvalidUserID tests error handling for invalid user ID
func TestSuggestCategories_InvalidUserID(t *testing.T) {
	// Setup mocks
	mockDB := &mockDB{
		cuisineCategories: []string{},
	}
	mockAI := &mockAI{
		response: &CategoryAIResponse{},
	}

	service := NewCategoryService(mockDB, mockAI)

	// Create test recipe
	recipe := RecipeInfo{
		Name:        "Test Recipe",
		Description: "Test description",
		Ingredients: []string{"pasta"},
	}

	ctx := context.Background()
	invalidUserID := "not-a-valid-uuid"

	// Execute
	result, err := service.SuggestCategories(ctx, recipe, invalidUserID)

	// Verify
	if err == nil {
		t.Fatal("expected error for invalid user ID, got nil")
	}

	if !strings.Contains(err.Error(), "invalid user ID") {
		t.Errorf("expected error message to contain 'invalid user ID', got %v", err)
	}

	if result != nil {
		t.Error("expected result to be nil for invalid user ID")
	}
}

// TestHasNoCategories tests the hasNoCategories helper method
func TestHasNoCategories(t *testing.T) {
	service := &CategoryService{}

	tests := []struct {
		name     string
		existing *CategorySet
		expected bool
	}{
		{
			name:     "All empty",
			existing: &CategorySet{},
			expected: true,
		},
		{
			name: "Some categories present",
			existing: &CategorySet{
				CuisineCategories:   []string{"Italian"},
				MealTypes:           []string{},
				Occasions:           []string{},
				DietaryRestrictions: []string{},
				Equipment:           []string{},
			},
			expected: false,
		},
		{
			name: "Multiple categories present",
			existing: &CategorySet{
				CuisineCategories:   []string{"Italian", "Mexican"},
				MealTypes:           []string{"Dinner"},
				Occasions:           []string{},
				DietaryRestrictions: []string{},
				Equipment:           []string{},
			},
			expected: false,
		},
		{
			name: "All categories present",
			existing: &CategorySet{
				CuisineCategories:   []string{"Italian"},
				MealTypes:           []string{"Dinner"},
				Occasions:           []string{"Weeknight"},
				DietaryRestrictions: []string{"Vegetarian"},
				Equipment:           []string{"Oven"},
			},
			expected: false,
		},
		{
			name: "Empty slices present",
			existing: &CategorySet{
				CuisineCategories:   []string{},
				MealTypes:           []string{},
				Occasions:           []string{},
				DietaryRestrictions: []string{},
				Equipment:           []string{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.hasNoCategories(tt.existing)
			if result != tt.expected {
				t.Errorf("hasNoCategories() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestBuildFallbackCategoryPrompt tests the fallback prompt generation
func TestBuildFallbackCategoryPrompt(t *testing.T) {
	tests := []struct {
		name        string
		recipe      RecipeInfo
		expect      []string
		expectNot   []string
		description string
	}{
		{
			name: "Complete recipe with all fields",
			recipe: RecipeInfo{
				Name:        "Spaghetti Carbonara",
				Description: "Classic Italian pasta dish with eggs, cheese, and pancetta",
				Ingredients: []string{"spaghetti", "eggs", "parmesan cheese", "pancetta", "black pepper"},
			},
			expect: []string{
				"<ROLE>",
				"Spaghetti Carbonara",
				"Classic Italian pasta dish with eggs, cheese, and pancetta",
				"- spaghetti",
				"- eggs",
				"- parmesan cheese",
				"- pancetta",
				"- black pepper",
				"<RECIPE_CONTEXT>",
				"<INSTRUCTIONS>",
				"CUISINE CATEGORIES",
				"MEAL TYPES",
				"OCCASIONS",
				"DIETARY RESTRICTIONS",
				"EQUIPMENT",
				"<OUTPUT_FORMAT>",
				"cuisine_categories",
				"meal_types",
				"occasions",
				"dietary_restrictions",
				"equipment",
			},
			expectNot: []string{
				"<EXISTING_CATEGORIES>", // Should NOT appear in fallback prompt
			},
			description: "Should generate a complete prompt with recipe name, description, and ingredients",
		},
		{
			name: "Recipe with name only",
			recipe: RecipeInfo{
				Name:        "Pasta Primavera",
				Description: "",
				Ingredients: []string{},
			},
			expect: []string{
				"<ROLE>",
				"Pasta Primavera",
				"<RECIPE_CONTEXT>",
			},
			description: "Should generate prompt even with minimal recipe info",
		},
		{
			name: "Recipe with name and ingredients only",
			recipe: RecipeInfo{
				Name:        "Margherita Pizza",
				Description: "",
				Ingredients: []string{"pizza dough", "tomato sauce", "fresh mozzarella", "basil"},
			},
			expect: []string{
				"<ROLE>",
				"Margherita Pizza",
				"- pizza dough",
				"- tomato sauce",
				"- fresh mozzarella",
				"- basil",
				"Ingredients:",
			},
			description: "Should generate prompt with name and ingredients",
		},
		{
			name: "Realistic pasta recipe",
			recipe: RecipeInfo{
				Name:        "Classic Marinara",
				Description: "Simple tomato pasta sauce",
				Ingredients: []string{"pasta", "tomatoes", "basil", "garlic", "olive oil", "salt", "pepper"},
			},
			expect: []string{
				"Classic Marinara",
				"Simple tomato pasta sauce",
				"- pasta",
				"- tomatoes",
				"- basil",
				"- garlic",
			},
			description: "Should handle realistic recipe with common ingredients",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildFallbackCategoryPrompt(tt.recipe)

			// Check expected substrings
			for _, exp := range tt.expect {
				if !strings.Contains(result, exp) {
					t.Errorf("expected prompt to contain '%s', but it was not found", exp)
				}
			}

			// Check substrings that should NOT appear
			for _, notExp := range tt.expectNot {
				if strings.Contains(result, notExp) {
					t.Errorf("expected prompt to NOT contain '%s', but it was found", notExp)
				}
			}
		})
	}
}

// TestSuggestCategories_MultipleCategoriesPerType tests that multiple categories can be returned per type
func TestSuggestCategories_MultipleCategoriesPerType(t *testing.T) {
	mockDB := &mockDB{
		cuisineCategories:   []string{"Italian", "Mediterranean", "Asian", "Mexican"},
		mealTypes:           []string{"Breakfast", "Lunch", "Dinner", "Snack"},
		occasions:           []string{"Weeknight", "Weekend", "Holiday", "Party"},
		dietaryRestrictions: []string{"Vegetarian", "Vegan", "Gluten-Free"},
		equipment:           []string{"Oven", "Stovetop", "Microwave", "Blender"},
	}

	mockAI := &mockAI{
		response: &CategoryAIResponse{
			CuisineCategories:   []string{"Italian", "Mediterranean"},
			MealTypes:           []string{"Dinner", "Lunch"},
			Occasions:           []string{"Weeknight", "Weekend"},
			DietaryRestrictions: []string{"Vegetarian"},
			Equipment:           []string{"Oven", "Stovetop"},
		},
	}

	service := NewCategoryService(mockDB, mockAI)

	recipe := RecipeInfo{
		Name:        "Mediterranean Pasta Salad",
		Description: "A refreshing pasta salad with Mediterranean flavors",
		Ingredients: []string{"pasta", "olive oil", "tomatoes", "olives", "feta cheese", "basil"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	result, err := service.SuggestCategories(ctx, recipe, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.CuisineCategories) < 2 {
		t.Errorf("expected at least 2 cuisine categories, got %d", len(result.CuisineCategories))
	}

	if len(result.MealTypes) < 2 {
		t.Errorf("expected at least 2 meal types, got %d", len(result.MealTypes))
	}
}

// TestSuggestCategories_EmptyAIResponse tests handling of empty AI response
func TestSuggestCategories_EmptyAIResponse(t *testing.T) {
	mockDB := &mockDB{
		cuisineCategories:   []string{},
		mealTypes:           []string{},
		occasions:           []string{},
		dietaryRestrictions: []string{},
		equipment:           []string{},
	}

	mockAI := &mockAI{
		response: &CategoryAIResponse{
			CuisineCategories:   []string{},
			MealTypes:           []string{},
			Occasions:           []string{},
			DietaryRestrictions: []string{},
			Equipment:           []string{},
		},
	}

	service := NewCategoryService(mockDB, mockAI)

	recipe := RecipeInfo{
		Name:        "Simple Dish",
		Description: "A simple dish",
		Ingredients: []string{"rice", "water"},
	}

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000"

	result, err := service.SuggestCategories(ctx, recipe, userID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}

	// Empty response should be valid
	if len(result.CuisineCategories) != 0 {
		t.Errorf("expected empty cuisine categories, got %v", result.CuisineCategories)
	}

	if len(result.MealTypes) != 0 {
		t.Errorf("expected empty meal types, got %v", result.MealTypes)
	}
}
