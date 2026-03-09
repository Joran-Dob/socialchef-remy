package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// DBQueries interface for category operations
type CategoryDBQueries interface {
	GetCuisineCategoriesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetMealTypesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetOccasionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetDietaryRestrictionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetEquipmentByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
}

// CategoryAIResponse represents the AI response for category suggestions
type CategoryAIResponse struct {
	CuisineCategories      []string        `json:"cuisine_categories"`
	MealTypes              []string        `json:"meal_types"`
	Occasions              []string        `json:"occasions"`
	DietaryRestrictions    []string        `json:"dietary_restrictions"`
	Equipment              []string        `json:"equipment"`
	NewCategorySuggestions *NewCategorySet `json:"new_category_suggestions,omitempty"`
}

// NewCategorySet holds new category suggestions
type NewCategorySet struct {
	CuisineCategories   []string `json:"cuisine_categories"`
	MealTypes           []string `json:"meal_types"`
	Occasions           []string `json:"occasions"`
	DietaryRestrictions []string `json:"dietary_restrictions"`
	Equipment           []string `json:"equipment"`
}

// AIClient interface for AI generation
type AIClient interface {
	GenerateCategories(ctx context.Context, prompt string) (*CategoryAIResponse, error)
}

// CategorySuggestions holds the result of category matching
type CategorySuggestions struct {
	CuisineCategories   []string
	MealTypes           []string
	Occasions           []string
	DietaryRestrictions []string
	Equipment           []string
}

// CategoryService provides intelligent category suggestions
type CategoryService struct {
	db CategoryDBQueries
	ai AIClient
}

// NewCategoryService creates a new category service
func NewCategoryService(db CategoryDBQueries, ai AIClient) *CategoryService {
	return &CategoryService{db: db, ai: ai}
}

// SuggestCategories suggests categories for a recipe based on user's existing categories
func (s *CategoryService) SuggestCategories(
	ctx context.Context,
	recipe RecipeInfo,
	userID string,
) (*CategorySuggestions, error) {
	logger := slog.With("user_id", userID, "recipe_name", recipe.Name)

	// 1. Parse userID to pgtype.UUID
	var pgUserID pgtype.UUID
	if err := pgUserID.Scan(userID); err != nil {
		logger.Error("failed to parse user ID", "error", err)
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 2. Fetch existing categories from DB
	existing, err := s.fetchExistingCategories(ctx, pgUserID)
	if err != nil {
		logger.Error("failed to fetch existing categories", "error", err)
		return nil, fmt.Errorf("failed to fetch existing categories: %w", err)
	}

	// 3. Build prompt based on whether user has existing categories
	var prompt string
	if s.hasNoCategories(existing) {
		logger.Debug("no existing categories found, using fallback prompt")
		prompt = BuildFallbackCategoryPrompt(recipe)
	} else {
		logger.Debug("using existing categories for prompt",
			"cuisine_count", len(existing.CuisineCategories),
			"meal_type_count", len(existing.MealTypes))
		prompt = BuildCategoryPrompt(recipe, *existing)
	}

	// 4. Call AI to get suggestions
	aiResponse, err := s.ai.GenerateCategories(ctx, prompt)
	if err != nil {
		logger.Error("failed to generate categories", "error", err)
		return nil, fmt.Errorf("failed to generate categories: %w", err)
	}

	// 5. Merge new_category_suggestions into main arrays
	mergeNewSuggestions(aiResponse)

	// 6. Parse response and return CategorySuggestions
	suggestions := &CategorySuggestions{
		CuisineCategories:   aiResponse.CuisineCategories,
		MealTypes:           aiResponse.MealTypes,
		Occasions:           aiResponse.Occasions,
		DietaryRestrictions: aiResponse.DietaryRestrictions,
		Equipment:           aiResponse.Equipment,
	}

	logger.Debug("category suggestions generated",
		"cuisine_count", len(suggestions.CuisineCategories),
		"meal_type_count", len(suggestions.MealTypes),
		"occasion_count", len(suggestions.Occasions),
		"dietary_count", len(suggestions.DietaryRestrictions),
		"equipment_count", len(suggestions.Equipment))

	return suggestions, nil
}

// fetchExistingCategories retrieves all existing categories for a user
func (s *CategoryService) fetchExistingCategories(ctx context.Context, userID pgtype.UUID) (*CategorySet, error) {
	var existing CategorySet
	var errs []error

	if cats, err := s.db.GetCuisineCategoriesByUser(ctx, userID); err != nil {
		errs = append(errs, fmt.Errorf("cuisine categories: %w", err))
	} else {
		existing.CuisineCategories = cats
	}

	if types, err := s.db.GetMealTypesByUser(ctx, userID); err != nil {
		errs = append(errs, fmt.Errorf("meal types: %w", err))
	} else {
		existing.MealTypes = types
	}

	if occasions, err := s.db.GetOccasionsByUser(ctx, userID); err != nil {
		errs = append(errs, fmt.Errorf("occasions: %w", err))
	} else {
		existing.Occasions = occasions
	}

	if restrictions, err := s.db.GetDietaryRestrictionsByUser(ctx, userID); err != nil {
		errs = append(errs, fmt.Errorf("dietary restrictions: %w", err))
	} else {
		existing.DietaryRestrictions = restrictions
	}

	if equipment, err := s.db.GetEquipmentByUser(ctx, userID); err != nil {
		errs = append(errs, fmt.Errorf("equipment: %w", err))
	} else {
		existing.Equipment = equipment
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to fetch some categories: %v", errs)
	}

	return &existing, nil
}

// mergeNewSuggestions appends entries from NewCategorySuggestions into the main
// arrays so that new suggestions are never silently dropped.
func mergeNewSuggestions(resp *CategoryAIResponse) {
	if resp == nil || resp.NewCategorySuggestions == nil {
		return
	}
	ns := resp.NewCategorySuggestions
	resp.CuisineCategories = appendUnique(resp.CuisineCategories, ns.CuisineCategories)
	resp.MealTypes = appendUnique(resp.MealTypes, ns.MealTypes)
	resp.Occasions = appendUnique(resp.Occasions, ns.Occasions)
	resp.DietaryRestrictions = appendUnique(resp.DietaryRestrictions, ns.DietaryRestrictions)
	resp.Equipment = appendUnique(resp.Equipment, ns.Equipment)
}

// appendUnique appends additions to existing, skipping case-insensitive duplicates.
func appendUnique(existing, additions []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, s := range existing {
		seen[strings.ToLower(s)] = true
	}
	for _, s := range additions {
		if !seen[strings.ToLower(s)] {
			existing = append(existing, s)
			seen[strings.ToLower(s)] = true
		}
	}
	return existing
}

// hasNoCategories checks if user has no existing categories
func (s *CategoryService) hasNoCategories(existing *CategorySet) bool {
	return len(existing.CuisineCategories) == 0 &&
		len(existing.MealTypes) == 0 &&
		len(existing.Occasions) == 0 &&
		len(existing.DietaryRestrictions) == 0 &&
		len(existing.Equipment) == 0
}

// BuildFallbackCategoryPrompt builds a prompt for first-time users with no existing categories
func BuildFallbackCategoryPrompt(recipe RecipeInfo) string {
	var sb strings.Builder

	sb.WriteString(`<ROLE>
You are a specialized AI assistant for recipe categorization. Your task is to analyze a recipe and suggest appropriate categories. Since this is the user's first recipe, generate categories freely based on the recipe content.
</ROLE>

`)

	sb.WriteString("<RECIPE_CONTEXT>\n")
	sb.WriteString(fmt.Sprintf("Recipe Name: %s\n", recipe.Name))
	if recipe.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", recipe.Description))
	}
	if len(recipe.Ingredients) > 0 {
		sb.WriteString("Ingredients:\n")
		for _, ing := range recipe.Ingredients {
			sb.WriteString(fmt.Sprintf("- %s\n", ing))
		}
	}
	sb.WriteString("</RECIPE_CONTEXT>\n\n")

	sb.WriteString(`<INSTRUCTIONS>
Analyze the recipe and suggest appropriate categories for each type:

1. CUISINE CATEGORIES: Identify the cuisine type based on ingredients, cooking techniques, and flavor profiles (e.g., Italian, Mexican, Asian, Mediterranean, American).

2. MEAL TYPES: Determine when this dish is typically eaten (e.g., Breakfast, Lunch, Dinner, Snack, Appetizer, Dessert).

3. OCCASIONS: Suggest when this dish would be served (e.g., Weeknight, Weekend, Special Occasion, Holiday, Party, Quick Meal).

4. DIETARY RESTRICTIONS: Only include if the recipe clearly fits specific diets (e.g., Vegetarian, Vegan, Gluten-Free, Keto, Paleo, Dairy-Free).

5. EQUIPMENT: List all necessary tools and equipment for preparation (e.g., Oven, Stovetop, Blender, Slow Cooker, Grill, Microwave).

Guidelines:
- Be specific and accurate based on the recipe content
- Multiple categories per type are allowed when appropriate
- Only suggest dietary restrictions if the recipe clearly fits
- Include all major equipment needed
- Use common, user-friendly category names
</INSTRUCTIONS>

`)

	sb.WriteString(`<OUTPUT_FORMAT>
Return ONLY a JSON object with the following structure (no additional text):

{
  "cuisine_categories": ["Italian", "Mediterranean"],
  "meal_types": ["Dinner"],
  "occasions": ["Weeknight", "Special Occasion"],
  "dietary_restrictions": ["Vegetarian"],
  "equipment": ["Oven", "Stovetop", "Baking Dish"],
  "new_category_suggestions": {
    "cuisine_categories": [],
    "meal_types": [],
    "occasions": [],
    "dietary_restrictions": [],
    "equipment": []
  }
}

Guidelines:
- Arrays can be empty if no categories fit
- Use descriptive, user-friendly names
- Equipment should include all tools needed for preparation
</OUTPUT_FORMAT>`)

	return sb.String()
}
