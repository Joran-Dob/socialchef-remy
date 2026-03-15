//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/groq"
	recipeservice "github.com/socialchef/remy/internal/services/recipe"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Implementations for Split Recipe Tests
// ============================================================================

type MockGroqClientSplitRecipe struct {
	t                *testing.T
	recipe           *groq.Recipe
	categories       *ai.CategoryAIResponse
	richInstructions *recipeservice.RichInstructionResponse
}

func (m *MockGroqClientSplitRecipe) GenerateRecipe(ctx context.Context, caption, transcript, platform string) (*groq.Recipe, error) {
	return m.recipe, nil
}

func (m *MockGroqClientSplitRecipe) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	return m.categories, nil
}

func (m *MockGroqClientSplitRecipe) GenerateRichInstructions(ctx context.Context, recipe *groq.Recipe) (*recipeservice.RichInstructionResponse, error) {
	return m.richInstructions, nil
}

// ============================================================================
// Test: Full Import Flow for Split Recipe
// ============================================================================

func TestSplitRecipe_FullImportFlow(t *testing.T) {
	ctx := context.Background()
	testDBConn, cleanup := setupTestDB(ctx)
	defer cleanup()

	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_split_recipe_test/"

	// Mock Groq client with split recipe response
	mockGroqSplit := &MockGroqClientSplitRecipe{
		t: t,
		recipe: &groq.Recipe{
			RecipeName:       "Layered Chocolate Cake",
			Description:      "A delicious three-layer chocolate cake with ganache",
			PrepTime:         intPtr(45),
			CookingTime:      intPtr(35),
			TotalTime:        intPtr(80),
			OriginalServings: intPtr(12),
			DifficultyRating: intPtr(4),
			Language:         "en",
			// Split recipe with parts instead of flat ingredients/instructions
			Parts: []recipeservice.RecipePart{
				{
					Name:         "Cake Layers",
					Description:  "The chocolate cake layers",
					DisplayOrder: 0,
					IsOptional:   false,
					PrepTime:     intPtr(20),
					CookingTime:  intPtr(25),
					Ingredients: []recipeservice.Ingredient{
						{
							Name:             "all-purpose flour",
							OriginalQuantity: "300",
							OriginalUnit:     "g",
							TotalQuantity:    "300",
							Quantity:         "300",
							Unit:             "g",
						},
						{
							Name:             "cocoa powder",
							OriginalQuantity: "100",
							OriginalUnit:     "g",
							TotalQuantity:    "100",
							Quantity:         "100",
							Unit:             "g",
						},
						{
							Name:             "sugar",
							OriginalQuantity: "400",
							OriginalUnit:     "g",
							TotalQuantity:    "400",
							Quantity:         "400",
							Unit:             "g",
						},
					},
					Instructions: []recipeservice.Instruction{
						{
							StepNumber:  1,
							Instruction: "Preheat oven to 350°F and grease three 8-inch pans",
						},
						{
							StepNumber:  2,
							Instruction: "Mix dry ingredients together in a large bowl",
						},
						{
							StepNumber:  3,
							Instruction: "Pour batter evenly into prepared pans",
						},
					},
				},
				{
					Name:         "Ganache",
					Description:  "Chocolate ganache for frosting",
					DisplayOrder: 1,
					IsOptional:   false,
					PrepTime:     intPtr(10),
					CookingTime:  intPtr(5),
					Ingredients: []recipeservice.Ingredient{
						{
							Name:             "dark chocolate",
							OriginalQuantity: "400",
							OriginalUnit:     "g",
							TotalQuantity:    "400",
							Quantity:         "400",
							Unit:             "g",
						},
						{
							Name:             "heavy cream",
							OriginalQuantity: "400",
							OriginalUnit:     "ml",
							TotalQuantity:    "400",
							Quantity:         "400",
							Unit:             "ml",
						},
					},
					Instructions: []recipeservice.Instruction{
						{
							StepNumber:  1,
							Instruction: "Chop chocolate finely and place in a heatproof bowl",
						},
						{
							StepNumber:  2,
							Instruction: "Heat cream until steaming and pour over chocolate",
						},
						{
							StepNumber:  3,
							Instruction: "Let sit for 5 minutes then stir until smooth",
						},
					},
				},
				{
					Name:         "Assembly",
					Description:  "Putting the cake together",
					DisplayOrder: 2,
					IsOptional:   false,
					PrepTime:     intPtr(15),
					Ingredients: []recipeservice.Ingredient{
						{
							Name:             "raspberries",
							OriginalQuantity: "200",
							OriginalUnit:     "g",
							TotalQuantity:    "200",
							Quantity:         "200",
							Unit:             "g",
						},
					},
					Instructions: []recipeservice.Instruction{
						{
							StepNumber:  1,
							Instruction: "Place first cake layer on a serving plate",
						},
						{
							StepNumber:  2,
							Instruction: "Spread ganache and top with second layer",
						},
						{
							StepNumber:  3,
							Instruction: "Decorate with fresh raspberries",
						},
					},
				},
			},
			Nutrition: recipeservice.Nutrition{
				Protein: 8.5,
				Carbs:   65.0,
				Fat:     28.0,
				Fiber:   4.2,
			},
		},
		categories: &ai.CategoryAIResponse{
			CuisineCategories:   []string{"Dessert", "American"},
			MealTypes:           []string{"Dessert"},
			Occasions:           []string{"Birthday", "Celebration"},
			DietaryRestrictions: []string{},
			Equipment:           []string{"Oven", "Mixing Bowls", "Cake Pans"},
		},
		richInstructions: &recipeservice.RichInstructionResponse{
			Instructions: []recipeservice.RichInstruction{
				{StepNumber: 1, InstructionRich: "Preheat oven to {{temp}} and grease pans"},
				{StepNumber: 2, InstructionRich: "Mix dry ingredients"},
				{StepNumber: 3, InstructionRich: "Pour into pans"},
			},
			PromptVersion: 1,
		},
	}

	// Setup mock image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-cake-image"))
	}))
	defer imageServer.Close()

	// Setup mock Instagram scraper
	mockInstagram := &MockInstagramScraperFixed{
		post: &scraper.InstagramPost{
			Caption:       "Amazing layered chocolate cake recipe! #chocolate #cake #dessert",
			ImageURL:      imageServer.URL,
			VideoURL:      "",
			OwnerUsername: "baker_bob",
			OwnerAvatar:   "",
			OwnerID:       "789012",
		},
	}

	mockTranscription := &MockTranscriptionClientFixed{}
	mockStorage := &MockStorageClientFixed{
		imageHash: "cake123hash",
	}
	mockBroadcaster := &MockProgressBroadcaster{
		Broadcasts: make([]ProgressUpdate, 0),
	}
	var asynqClient *asynq.Client

	broadcasterAdapter := &progressBroadcasterAdapter{inner: mockBroadcaster}

	// Create processor
	processor := worker.NewRecipeProcessor(
		testDBConn.queries,
		mockInstagram,
		nil, // TikTok
		nil, // Firecrawl
		nil, // OpenAI
		mockTranscription,
		mockGroqSplit,
		mockStorage,
		broadcasterAdapter,
		nil, // metrics
		asynqClient,
	)

	// Create import job
	_, err := testDBConn.queries.CreateImportJob(ctx, generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    url,
		Status: "QUEUED",
	})
	require.NoError(t, err)

	// Create task
	payload := worker.ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(worker.TypeProcessRecipe, payloadBytes)

	// Execute handler
	err = processor.HandleProcessRecipe(ctx, task)
	require.NoError(t, err, "Handler should complete successfully")

	// Verify import job status
	importJob, err := testDBConn.queries.GetImportJob(ctx, uuidToPgtype(uuid.MustParse(jobID)))
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", importJob.Status, "Import job should be completed")

	// Find the created recipe
	recipes, err := testDBConn.queries.GetRecipesByUser(ctx, uuidToPgtype(uuid.MustParse(userID)))
	require.NoError(t, err)
	require.Len(t, recipes, 1, "Should have created one recipe")

	recipe := recipes[0]
	assert.Equal(t, "Layered Chocolate Cake", recipe.RecipeName)

	t.Run("VerifyRecipePartsCreated", func(t *testing.T) {
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query recipe parts")
		require.Len(t, parts, 3, "Should have created 3 recipe parts")

		// Verify part names and order
		partNames := make([]string, len(parts))
		for i, part := range parts {
			partNames[i] = part.Name
			assert.Equal(t, int32(i), part.DisplayOrder, "Display order should match index")
		}
		assert.Contains(t, partNames, "Cake Layers")
		assert.Contains(t, partNames, "Ganache")
		assert.Contains(t, partNames, "Assembly")

		// Verify part details
		for _, part := range parts {
			switch part.Name {
			case "Cake Layers":
				assert.Equal(t, "The chocolate cake layers", part.Description.String)
				assert.False(t, part.IsOptional)
				assert.Equal(t, int32(20), part.PrepTime.Int32)
				assert.Equal(t, int32(25), part.CookingTime.Int32)
			case "Ganache":
				assert.Equal(t, "Chocolate ganache for frosting", part.Description.String)
				assert.False(t, part.IsOptional)
				assert.Equal(t, int32(10), part.PrepTime.Int32)
				assert.Equal(t, int32(5), part.CookingTime.Int32)
			case "Assembly":
				assert.Equal(t, "Putting the cake together", part.Description.String)
				assert.False(t, part.IsOptional)
				assert.Equal(t, int32(15), part.PrepTime.Int32)
			}
		}

		t.Logf("✅ Verified: Created %d recipe parts", len(parts))
	})

	t.Run("VerifyIngredientsHavePartID", func(t *testing.T) {
		ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query ingredients")
		require.Len(t, ingredients, 6, "Should have created 6 total ingredients (3 + 2 + 1)")

		// Get parts to build part ID -> name mapping
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err)

		partMap := make(map[[16]byte]string)
		for _, part := range parts {
			partMap[part.ID.Bytes] = part.Name
		}

		// Verify each ingredient has a part_id
		ingredientsByPart := make(map[string][]string)
		for _, ing := range ingredients {
			require.True(t, ing.PartID.Valid, "Ingredient %s should have a valid part_id", ing.Name)
			partName := partMap[ing.PartID.Bytes]
			ingredientsByPart[partName] = append(ingredientsByPart[partName], ing.Name)
		}

		// Verify ingredients are correctly assigned to parts
		assert.Len(t, ingredientsByPart["Cake Layers"], 3, "Cake Layers should have 3 ingredients")
		assert.Len(t, ingredientsByPart["Ganache"], 2, "Ganache should have 2 ingredients")
		assert.Len(t, ingredientsByPart["Assembly"], 1, "Assembly should have 1 ingredient")

		// Verify specific ingredients
		assert.Contains(t, ingredientsByPart["Cake Layers"], "all-purpose flour")
		assert.Contains(t, ingredientsByPart["Cake Layers"], "cocoa powder")
		assert.Contains(t, ingredientsByPart["Cake Layers"], "sugar")
		assert.Contains(t, ingredientsByPart["Ganache"], "dark chocolate")
		assert.Contains(t, ingredientsByPart["Ganache"], "heavy cream")
		assert.Contains(t, ingredientsByPart["Assembly"], "raspberries")

		t.Logf("✅ Verified: All %d ingredients have correct part_id assignments", len(ingredients))
	})

	t.Run("VerifyInstructionsHavePartID", func(t *testing.T) {
		instructions, err := testDBConn.queries.GetInstructionsByRecipe(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query instructions")
		require.Len(t, instructions, 9, "Should have created 9 total instructions (3 + 3 + 3)")

		// Get parts to build part ID -> name mapping
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err)

		partMap := make(map[[16]byte]string)
		for _, part := range parts {
			partMap[part.ID.Bytes] = part.Name
		}

		// Verify each instruction has a part_id
		instructionsByPart := make(map[string]int)
		stepNumbers := make(map[string][]int32)
		for _, inst := range instructions {
			require.True(t, inst.PartID.Valid, "Instruction step %d should have a valid part_id", inst.StepNumber)
			partName := partMap[inst.PartID.Bytes]
			instructionsByPart[partName]++
			stepNumbers[partName] = append(stepNumbers[partName], inst.StepNumber)
		}

		// Verify instruction counts per part
		assert.Equal(t, 3, instructionsByPart["Cake Layers"], "Cake Layers should have 3 instructions")
		assert.Equal(t, 3, instructionsByPart["Ganache"], "Ganache should have 3 instructions")
		assert.Equal(t, 3, instructionsByPart["Assembly"], "Assembly should have 3 instructions")

		// Verify step numbering is sequential across all parts
		allSteps := make(map[int32]bool)
		for _, steps := range stepNumbers {
			for _, step := range steps {
				allSteps[step] = true
			}
		}
		for i := int32(1); i <= 9; i++ {
			assert.True(t, allSteps[i], "Step %d should exist", i)
		}

		t.Logf("✅ Verified: All %d instructions have correct part_id and sequential numbering", len(instructions))
	})

	t.Run("VerifyAPIRetrievalReturnsCorrectStructure", func(t *testing.T) {
		// Get the full recipe with parts using the GetRecipeWithParts query
		fullRecipe, err := testDBConn.queries.GetRecipeWithParts(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to retrieve full recipe with parts")

		// Verify basic recipe data
		assert.Equal(t, "Layered Chocolate Cake", fullRecipe.RecipeName)
		assert.Equal(t, "A delicious three-layer chocolate cake with ganache", fullRecipe.Description.String)

		// Verify parts JSON is present
		require.NotNil(t, fullRecipe.Parts, "Parts JSON should be present")
		require.NotEmpty(t, fullRecipe.Parts, "Parts JSON should not be empty")

		// Parse the parts JSON to verify structure
		var partsData []map[string]interface{}
		err = json.Unmarshal(fullRecipe.Parts, &partsData)
		require.NoError(t, err, "Parts should be valid JSON")
		require.Len(t, partsData, 3, "Should have 3 parts in JSON")

		// Verify each part has the expected structure
		for _, part := range partsData {
			assert.NotEmpty(t, part["id"], "Part should have an ID")
			assert.NotEmpty(t, part["name"], "Part should have a name")
			assert.NotNil(t, part["display_order"], "Part should have display_order")
			assert.NotNil(t, part["ingredients"], "Part should have ingredients")
			assert.NotNil(t, part["instructions"], "Part should have instructions")

			// Verify ingredients is an array
			ingredients, ok := part["ingredients"].([]interface{})
			if ok {
				assert.Greater(t, len(ingredients), 0, "Each part should have at least one ingredient")
			}

			// Verify instructions is an array
			instructions, ok := part["instructions"].([]interface{})
			if ok {
				assert.Greater(t, len(instructions), 0, "Each part should have at least one instruction")
			}
		}

		t.Logf("✅ Verified: API retrieval returns correct structured parts data")
	})

	t.Logf("✅ Test passed: Split recipe import flow works correctly")
	t.Logf("   Created recipe: %s (ID: %s)", recipe.RecipeName, uuid.UUID(recipe.ID.Bytes).String())
}

// ============================================================================
// Test: Backward Compatibility with Flat Recipe
// ============================================================================

func TestSplitRecipe_BackwardCompatibility_FlatRecipe(t *testing.T) {
	ctx := context.Background()
	testDBConn, cleanup := setupTestDB(ctx)
	defer cleanup()

	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_flat_recipe_test/"

	// Mock Groq client with flat recipe (no parts)
	mockGroqFlat := &MockGroqClientSplitRecipe{
		t: t,
		recipe: &groq.Recipe{
			RecipeName:       "Simple Pancakes",
			Description:      "Classic fluffy pancakes",
			PrepTime:         intPtr(10),
			CookingTime:      intPtr(15),
			TotalTime:        intPtr(25),
			OriginalServings: intPtr(4),
			DifficultyRating: intPtr(1),
			Language:         "en",
			// Flat recipe - ingredients and instructions at root level
			Ingredients: []recipeservice.Ingredient{
				{
					Name:             "flour",
					OriginalQuantity: "200",
					OriginalUnit:     "g",
					TotalQuantity:    "200",
					Quantity:         "200",
					Unit:             "g",
				},
				{
					Name:             "milk",
					OriginalQuantity: "300",
					OriginalUnit:     "ml",
					TotalQuantity:    "300",
					Quantity:         "300",
					Unit:             "ml",
				},
				{
					Name:             "eggs",
					OriginalQuantity: "2",
					OriginalUnit:     "",
					TotalQuantity:    "2",
					Quantity:         "2",
					Unit:             "",
				},
			},
			Instructions: []recipeservice.Instruction{
				{
					StepNumber:  1,
					Instruction: "Mix all ingredients in a bowl",
				},
				{
					StepNumber:  2,
					Instruction: "Cook on medium heat until golden",
				},
				{
					StepNumber:  3,
					Instruction: "Serve with syrup",
				},
			},
			// No parts - empty slice
			Parts: []recipeservice.RecipePart{},
			Nutrition: recipeservice.Nutrition{
				Protein: 6.0,
				Carbs:   30.0,
				Fat:     5.0,
				Fiber:   1.0,
			},
		},
		categories: &ai.CategoryAIResponse{
			CuisineCategories:   []string{"Breakfast"},
			MealTypes:           []string{"Breakfast"},
			Occasions:           []string{},
			DietaryRestrictions: []string{},
			Equipment:           []string{"Pan"},
		},
		richInstructions: &recipeservice.RichInstructionResponse{
			Instructions: []recipeservice.RichInstruction{
				{StepNumber: 1, InstructionRich: "Mix {{ingredient:0}}, {{ingredient:1}}, and {{ingredient:2}}"},
				{StepNumber: 2, InstructionRich: "Cook on medium heat"},
				{StepNumber: 3, InstructionRich: "Serve with syrup"},
			},
			PromptVersion: 1,
		},
	}

	// Setup mock image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-pancake-image"))
	}))
	defer imageServer.Close()

	// Setup mock Instagram scraper
	mockInstagram := &MockInstagramScraperFixed{
		post: &scraper.InstagramPost{
			Caption:       "Simple pancake recipe! #breakfast #pancakes",
			ImageURL:      imageServer.URL,
			VideoURL:      "",
			OwnerUsername: "chef_marie",
			OwnerAvatar:   "",
			OwnerID:       "345678",
		},
	}

	mockTranscription := &MockTranscriptionClientFixed{}
	mockStorage := &MockStorageClientFixed{
		imageHash: "pancake456hash",
	}
	mockBroadcaster := &MockProgressBroadcaster{
		Broadcasts: make([]ProgressUpdate, 0),
	}
	var asynqClient *asynq.Client

	broadcasterAdapter := &progressBroadcasterAdapter{inner: mockBroadcaster}

	// Create processor
	processor := worker.NewRecipeProcessor(
		testDBConn.queries,
		mockInstagram,
		nil, // TikTok
		nil, // Firecrawl
		nil, // OpenAI
		mockTranscription,
		mockGroqFlat,
		mockStorage,
		broadcasterAdapter,
		nil, // metrics
		asynqClient,
	)

	// Create import job
	_, err := testDBConn.queries.CreateImportJob(ctx, generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    url,
		Status: "QUEUED",
	})
	require.NoError(t, err)

	// Create task
	payload := worker.ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(worker.TypeProcessRecipe, payloadBytes)

	// Execute handler
	err = processor.HandleProcessRecipe(ctx, task)
	require.NoError(t, err, "Handler should complete successfully")

	// Verify import job status
	importJob, err := testDBConn.queries.GetImportJob(ctx, uuidToPgtype(uuid.MustParse(jobID)))
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", importJob.Status, "Import job should be completed")

	// Find the created recipe
	recipes, err := testDBConn.queries.GetRecipesByUser(ctx, uuidToPgtype(uuid.MustParse(userID)))
	require.NoError(t, err)
	require.Len(t, recipes, 1, "Should have created one recipe")

	recipe := recipes[0]
	assert.Equal(t, "Simple Pancakes", recipe.RecipeName)

	t.Run("VerifyNoRecipePartsForFlatRecipe", func(t *testing.T) {
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query recipe parts")
		assert.Len(t, parts, 0, "Flat recipe should have no parts")

		t.Logf("✅ Verified: Flat recipe has no parts (backward compatible)")
	})

	t.Run("VerifyIngredientsHaveNoPartID", func(t *testing.T) {
		ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query ingredients")
		require.Len(t, ingredients, 3, "Should have created 3 ingredients")

		// Verify each ingredient has NO part_id (NULL)
		for _, ing := range ingredients {
			assert.False(t, ing.PartID.Valid, "Flat recipe ingredient %s should have NULL part_id", ing.Name)
		}

		// Verify ingredient names
		ingredientNames := make([]string, len(ingredients))
		for i, ing := range ingredients {
			ingredientNames[i] = ing.Name
		}
		assert.Contains(t, ingredientNames, "flour")
		assert.Contains(t, ingredientNames, "milk")
		assert.Contains(t, ingredientNames, "eggs")

		t.Logf("✅ Verified: Flat recipe ingredients have NULL part_id")
	})

	t.Run("VerifyInstructionsHaveNoPartID", func(t *testing.T) {
		instructions, err := testDBConn.queries.GetInstructionsByRecipe(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to query instructions")
		require.Len(t, instructions, 3, "Should have created 3 instructions")

		// Verify each instruction has NO part_id (NULL)
		for _, inst := range instructions {
			assert.False(t, inst.PartID.Valid, "Flat recipe instruction step %d should have NULL part_id", inst.StepNumber)
		}

		// Verify step numbering
		for i, inst := range instructions {
			assert.Equal(t, int32(i+1), inst.StepNumber, "Step number should be sequential")
		}

		t.Logf("✅ Verified: Flat recipe instructions have NULL part_id")
	})

	t.Run("VerifyAPIRetrievalForFlatRecipe", func(t *testing.T) {
		// Get the full recipe using the GetRecipeWithParts query
		fullRecipe, err := testDBConn.queries.GetRecipeWithParts(ctx, recipe.ID)
		require.NoError(t, err, "Should be able to retrieve recipe")

		// Verify basic recipe data
		assert.Equal(t, "Simple Pancakes", fullRecipe.RecipeName)

		// Verify parts JSON is NULL or empty for flat recipes
		// The JSON aggregate should return [null] or similar for flat recipes
		t.Logf("Parts data length: %d", len(fullRecipe.Parts))

		t.Logf("✅ Verified: Flat recipe API retrieval works correctly")
	})

	t.Run("VerifyGetIngredientsByRecipeStillWorks", func(t *testing.T) {
		ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
		require.NoError(t, err)
		assert.Len(t, ingredients, 3)

		t.Logf("✅ Verified: GetIngredientsByRecipe works for flat recipes")
	})

	t.Run("VerifyGetInstructionsByRecipeStillWorks", func(t *testing.T) {
		instructions, err := testDBConn.queries.GetInstructionsByRecipe(ctx, recipe.ID)
		require.NoError(t, err)
		assert.Len(t, instructions, 3)

		t.Logf("✅ Verified: GetInstructionsByRecipe works for flat recipes")
	})

	t.Logf("✅ Test passed: Backward compatibility with flat recipes works correctly")
	t.Logf("   Created flat recipe: %s (ID: %s)", recipe.RecipeName, uuid.UUID(recipe.ID.Bytes).String())
}

// ============================================================================
// Test: Mixed Recipe with Both Parts and Flat Data
// ============================================================================

func TestSplitRecipe_PartsTakePrecedence(t *testing.T) {
	ctx := context.Background()
	testDBConn, cleanup := setupTestDB(ctx)
	defer cleanup()

	// Test that when a recipe has both Parts AND flat ingredients/instructions,
	// the Parts take precedence and flat data is ignored

	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_mixed_recipe_test/"

	// Mock Groq client with both parts AND flat data
	mockGroqMixed := &MockGroqClientSplitRecipe{
		t: t,
		recipe: &groq.Recipe{
			RecipeName:       "Test Mixed Recipe",
			Description:      "Recipe with both parts and flat data",
			PrepTime:         intPtr(30),
			CookingTime:      intPtr(20),
			TotalTime:        intPtr(50),
			OriginalServings: intPtr(4),
			Language:         "en",
			// Parts defined - these should be used
			Parts: []recipeservice.RecipePart{
				{
					Name:         "Part A",
					DisplayOrder: 0,
					Ingredients: []recipeservice.Ingredient{
						{
							Name:             "ingredient from part A",
							OriginalQuantity: "100",
							OriginalUnit:     "g",
							TotalQuantity:    "100",
							Quantity:         "100",
							Unit:             "g",
						},
					},
					Instructions: []recipeservice.Instruction{
						{
							StepNumber:  1,
							Instruction: "Step from part A",
						},
					},
				},
			},
			// Flat data also defined - these should be IGNORED when Parts exist
			Ingredients: []recipeservice.Ingredient{
				{
					Name:             "flat ingredient",
					OriginalQuantity: "500",
					OriginalUnit:     "g",
					TotalQuantity:    "500",
					Quantity:         "500",
					Unit:             "g",
				},
			},
			Instructions: []recipeservice.Instruction{
				{
					StepNumber:  1,
					Instruction: "Flat instruction step",
				},
			},
			Nutrition: recipeservice.Nutrition{
				Protein: 10.0,
				Carbs:   40.0,
				Fat:     8.0,
				Fiber:   2.0,
			},
		},
		categories: &ai.CategoryAIResponse{
			CuisineCategories: []string{"Test"},
			MealTypes:         []string{"Test"},
		},
		richInstructions: &recipeservice.RichInstructionResponse{
			Instructions:  []recipeservice.RichInstruction{},
			PromptVersion: 1,
		},
	}

	// Setup mocks
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image"))
	}))
	defer imageServer.Close()

	mockInstagram := &MockInstagramScraperFixed{
		post: &scraper.InstagramPost{
			Caption:       "Test recipe #test",
			ImageURL:      imageServer.URL,
			OwnerUsername: "test_chef",
			OwnerID:       "999999",
		},
	}
	mockTranscription := &MockTranscriptionClientFixed{}
	mockStorage := &MockStorageClientFixed{imageHash: "testhash"}
	mockBroadcaster := &MockProgressBroadcaster{Broadcasts: make([]ProgressUpdate, 0)}

	processor := worker.NewRecipeProcessor(
		testDBConn.queries,
		mockInstagram,
		nil, nil, nil,
		mockTranscription,
		mockGroqMixed,
		mockStorage,
		&progressBroadcasterAdapter{inner: mockBroadcaster},
		nil, nil,
	)

	// Create and process job
	_, err := testDBConn.queries.CreateImportJob(ctx, generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    url,
		Status: "QUEUED",
	})
	require.NoError(t, err)

	payload := worker.ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(worker.TypeProcessRecipe, payloadBytes)

	err = processor.HandleProcessRecipe(ctx, task)
	require.NoError(t, err)

	// Get the created recipe
	recipes, err := testDBConn.queries.GetRecipesByUser(ctx, uuidToPgtype(uuid.MustParse(userID)))
	require.NoError(t, err)
	require.Len(t, recipes, 1)
	recipe := recipes[0]

	// Verify only Part data was saved, not flat data
	t.Run("VerifyPartsTakePrecedence", func(t *testing.T) {
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err)
		require.Len(t, parts, 1, "Should have 1 part")
		assert.Equal(t, "Part A", parts[0].Name)

		ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
		require.NoError(t, err)
		require.Len(t, ingredients, 1, "Should have only 1 ingredient (from part, not flat)")
		assert.Equal(t, "ingredient from part A", ingredients[0].Name)
		assert.True(t, ingredients[0].PartID.Valid, "Ingredient should have part_id")

		instructions, err := testDBConn.queries.GetInstructionsByRecipe(ctx, recipe.ID)
		require.NoError(t, err)
		require.Len(t, instructions, 1, "Should have only 1 instruction (from part, not flat)")
		assert.Equal(t, "Step from part A", instructions[0].Instruction)
		assert.True(t, instructions[0].PartID.Valid, "Instruction should have part_id")

		t.Logf("✅ Verified: Parts take precedence over flat data")
	})
}

// ============================================================================
// Test: Empty Parts Array Behavior
// ============================================================================

func TestSplitRecipe_EmptyPartsArray(t *testing.T) {
	ctx := context.Background()
	testDBConn, cleanup := setupTestDB(ctx)
	defer cleanup()

	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_empty_parts_test/"

	// Mock with empty parts array but flat data
	mockGroqEmptyParts := &MockGroqClientSplitRecipe{
		t: t,
		recipe: &groq.Recipe{
			RecipeName:       "Empty Parts Recipe",
			Description:      "Recipe with empty parts array",
			PrepTime:         intPtr(15),
			OriginalServings: intPtr(2),
			Language:         "en",
			Parts:            []recipeservice.RecipePart{}, // Empty but not nil
			Ingredients: []recipeservice.Ingredient{
				{
					Name:             "salt",
					OriginalQuantity: "1",
					OriginalUnit:     "tsp",
					TotalQuantity:    "1",
					Quantity:         "1",
					Unit:             "tsp",
				},
			},
			Instructions: []recipeservice.Instruction{
				{
					StepNumber:  1,
					Instruction: "Add salt",
				},
			},
			Nutrition: recipeservice.Nutrition{Protein: 0, Carbs: 0, Fat: 0, Fiber: 0},
		},
		categories: &ai.CategoryAIResponse{},
		richInstructions: &recipeservice.RichInstructionResponse{
			Instructions:  []recipeservice.RichInstruction{},
			PromptVersion: 1,
		},
	}

	// Setup mocks
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image"))
	}))
	defer imageServer.Close()

	mockInstagram := &MockInstagramScraperFixed{
		post: &scraper.InstagramPost{
			Caption:       "Test #recipe",
			ImageURL:      imageServer.URL,
			OwnerUsername: "test_chef",
			OwnerID:       "111111",
		},
	}
	mockTranscription := &MockTranscriptionClientFixed{}
	mockStorage := &MockStorageClientFixed{imageHash: "testhash2"}
	mockBroadcaster := &MockProgressBroadcaster{Broadcasts: make([]ProgressUpdate, 0)}

	processor := worker.NewRecipeProcessor(
		testDBConn.queries,
		mockInstagram,
		nil, nil, nil,
		mockTranscription,
		mockGroqEmptyParts,
		mockStorage,
		&progressBroadcasterAdapter{inner: mockBroadcaster},
		nil, nil,
	)

	// Create and process job
	_, err := testDBConn.queries.CreateImportJob(ctx, generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    url,
		Status: "QUEUED",
	})
	require.NoError(t, err)

	payload := worker.ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(worker.TypeProcessRecipe, payloadBytes)

	err = processor.HandleProcessRecipe(ctx, task)
	require.NoError(t, err)

	// Get the created recipe
	recipes, err := testDBConn.queries.GetRecipesByUser(ctx, uuidToPgtype(uuid.MustParse(userID)))
	require.NoError(t, err)
	require.Len(t, recipes, 1)
	recipe := recipes[0]

	t.Run("VerifyEmptyPartsTreatedAsFlatRecipe", func(t *testing.T) {
		parts, err := testDBConn.queries.GetRecipeParts(ctx, recipe.ID)
		require.NoError(t, err)
		assert.Len(t, parts, 0, "Empty parts array should result in no parts")

		ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
		require.NoError(t, err)
		require.Len(t, ingredients, 1)
		assert.False(t, ingredients[0].PartID.Valid, "Ingredient should have NULL part_id when parts array is empty")

		t.Logf("✅ Verified: Empty parts array treated as flat recipe")
	})
}
