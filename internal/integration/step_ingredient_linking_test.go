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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/socialchef/remy/internal/db"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/groq"
	recipeservice "github.com/socialchef/remy/internal/services/recipe"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/services/storage"
	"github.com/socialchef/remy/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStepLevelIngredientLinking tests the full flow from AI response to database
// for step-level ingredient linking functionality
func TestStepLevelIngredientLinking(t *testing.T) {
	// Setup test database connection
	ctx := context.Background()
	testDBConn, cleanup := setupTestDB(ctx)
	defer cleanup()

	// Create test fixtures
	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_test123/"

	// Mock Groq client with realistic recipe response
	mockGroq = &MockGroqClientWithInstructionIngredients{
		t: t,
		recipe: &groq.Recipe{
			RecipeName:       "Pancake Recipe",
			Description:      "Delicious homemade pancakes",
			PrepTime:         intPtr(10),
			CookingTime:      intPtr(15),
			TotalTime:        intPtr(25),
			OriginalServings: intPtr(4),
			DifficultyRating: intPtr(2),
			Language:         "en",
			Ingredients: []groq.Ingredient{
				{
					Name:             "milk",
					OriginalQuantity: "500",
					OriginalUnit:     "ml",
					TotalQuantity:    "500",
					Quantity:         "500",
					Unit:             "ml",
				},
				{
					Name:             "flour",
					OriginalQuantity: "200",
					OriginalUnit:     "g",
					TotalQuantity:    "200",
					Quantity:         "200",
					Unit:             "g",
				},
			},
			Instructions: []groq.Instruction{
				{
					StepNumber:  1,
					Instruction: "Mix 300ml milk with flour",
					IngredientsUsed: []recipeservice.StepIngredient{
						{
							IngredientName: "milk",
							QuantityUsed:   "300ml",
						},
						{
							IngredientName: "flour",
							QuantityUsed:   "200g",
						},
					},
				},
				{
					StepNumber:  2,
					Instruction: "Add remaining milk",
					IngredientsUsed: []recipeservice.StepIngredient{
						{
							IngredientName: "milk",
							QuantityUsed:   "200ml",
						},
					},
				},
			},
			Nutrition: groq.Nutrition{
				Protein: 12.0,
				Carbs:   45.0,
				Fat:     8.0,
				Fiber:   2.0,
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
				{
					StepNumber:      1,
					InstructionRich: "Mix {{ingredient:0}} with {{ingredient:1}}",
				},
				{
					StepNumber:      2,
					InstructionRich: "Add remaining {{ingredient:0}}",
				},
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
	mockInstagram = &MockInstagramScraperFixed{
		post: &scraper.InstagramPost{
			Caption:       "Delicious pancake recipe! #breakfast #pancakes #homemade",
			ImageURL:      imageServer.URL,
			VideoURL:      "",
			OwnerUsername: "chef_alex",
			OwnerAvatar:   "",
			OwnerID:       "123456",
		},
	}

	// Create mock dependencies
	mockTranscription = &MockTranscriptionClientFixed{}
	mockStorage = &MockStorageClientFixed{
		imageHash: "abc123hash",
	}
	mockBroadcaster = &MockProgressBroadcaster{
		Broadcasts: make([]ProgressUpdate, 0),
	}
	// Use nil asynq client for this test
	var asynqClient *asynq.Client

	// Create processor with proper adapter for broadcaster
	broadcasterAdapter := &progressBroadcasterAdapter{inner: mockBroadcaster}

	// Create processor
	processor := worker.NewRecipeProcessor(
		testDBConn.queries,
		mockInstagram,
		nil, // TikTok
		nil, // Firecrawl
		nil, // OpenAI
		mockTranscription,
		mockGroq,
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
	assert.Equal(t, "Pancake Recipe", recipe.RecipeName)
	assert.Equal(t, "Delicious homemade pancakes", recipe.Description.String)

	// Verify ingredients
	ingredients, err := testDBConn.queries.GetIngredientsByRecipe(ctx, recipe.ID)
	require.NoError(t, err)
	assert.Len(t, ingredients, 2, "Should have created 2 ingredients")

	// Build ingredient name -> ID map
	ingredientMap := make(map[string]pgtype.UUID)
	for _, ing := range ingredients {
		ingredientMap[ing.Name] = ing.ID
	}

	// Verify ingredient names
	assert.Contains(t, ingredientMap, "milk")
	assert.Contains(t, ingredientMap, "flour")

	// Verify instructions
	instructions, err := testDBConn.queries.GetInstructionsByRecipe(ctx, recipe.ID)
	require.NoError(t, err)
	assert.Len(t, instructions, 2, "Should have created 2 instructions")

	// Build instruction step number -> ID map
	instructionMap := make(map[int32]pgtype.UUID)
	for _, inst := range instructions {
		instructionMap[inst.StepNumber] = inst.ID
	}

	// Verify instruction text
	step1 := instructions[0]
	step2 := instructions[1]
	if step1.StepNumber != 1 {
		step1, step2 = step2, step1
	}
	assert.Equal(t, int32(1), step1.StepNumber)
	assert.Equal(t, "Mix 300ml milk with flour", step1.Instruction)
	assert.Equal(t, int32(2), step2.StepNumber)
	assert.Equal(t, "Add remaining milk", step2.Instruction)

	// Verify junction entries - this is the core test
	junctions, err := testDBConn.queries.GetInstructionIngredientsByRecipe(ctx, recipe.ID)
	require.NoError(t, err, "Should be able to query instruction ingredients")
	assert.Len(t, junctions, 3, "Should have created 3 junction entries")

	// Verify each junction entry
	// Expected entries:
	// 1. (step1, milk, 300ml)
	// 2. (step1, flour, 200g)
	// 3. (step2, milk, 200ml)

	step1MilkFound := false
	step1FlourFound := false
	step2MilkFound := false

	milkID := ingredientMap["milk"]
	flourID := ingredientMap["flour"]
	step1ID := instructionMap[1]
	step2ID := instructionMap[2]

	for _, junction := range junctions {
		if junction.InstructionID.Bytes == step1ID.Bytes {
			if junction.IngredientID.Bytes == milkID.Bytes {
				assert.Equal(t, "300ml", junction.StepQuantity.String, "Step 1 should use 300ml of milk")
				step1MilkFound = true
			} else if junction.IngredientID.Bytes == flourID.Bytes {
				assert.Equal(t, "200g", junction.StepQuantity.String, "Step 1 should use 200g of flour")
				step1FlourFound = true
			}
		} else if junction.InstructionID.Bytes == step2ID.Bytes {
			if junction.IngredientID.Bytes == milkID.Bytes {
				assert.Equal(t, "200ml", junction.StepQuantity.String, "Step 2 should use 200ml of milk")
				step2MilkFound = true
			}
		}
	}

	assert.True(t, step1MilkFound, "Step 1 -> Milk junction should exist")
	assert.True(t, step1FlourFound, "Step 1 -> Flour junction should exist")
	assert.True(t, step2MilkFound, "Step 2 -> Milk junction should exist")

	t.Logf("✅ Test passed: Step-level ingredient linking works correctly")
	t.Logf("   Created recipe: %s (ID: %s)", recipe.RecipeName, uuid.UUID(recipe.ID.Bytes).String())
	t.Logf("   Created 2 ingredients: milk, flour")
	t.Logf("   Created 2 instructions")
	t.Logf("   Created 3 junction entries:")
	t.Logf("     - Step 1 -> Milk (300ml)")
	t.Logf("     - Step 1 -> Flour (200g)")
	t.Logf("     - Step 2 -> Milk (200ml)")
}

// Mock implementations for testing (global variables for mock package compatibility)

var mockGroq *MockGroqClientWithInstructionIngredients
var mockInstagram *MockInstagramScraperFixed
var mockTranscription *MockTranscriptionClientFixed
var mockStorage *MockStorageClientFixed
var mockBroadcaster *MockProgressBroadcaster

type MockGroqClientWithInstructionIngredients struct {
	t                *testing.T
	recipe           *groq.Recipe
	categories       *ai.CategoryAIResponse
	richInstructions *recipeservice.RichInstructionResponse
}

func (m *MockGroqClientWithInstructionIngredients) GenerateRecipe(ctx context.Context, caption, transcript, platform string) (*groq.Recipe, error) {
	return m.recipe, nil
}

func (m *MockGroqClientWithInstructionIngredients) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	return m.categories, nil
}

func (m *MockGroqClientWithInstructionIngredients) GenerateRichInstructions(ctx context.Context, recipe *groq.Recipe) (*recipeservice.RichInstructionResponse, error) {
	return m.richInstructions, nil
}

type MockInstagramScraperFixed struct {
	post *scraper.InstagramPost
}

func (m *MockInstagramScraperFixed) Scrape(ctx context.Context, postURL string) (*scraper.InstagramPost, error) {
	return m.post, nil
}

type MockTranscriptionClientFixed struct{}

func (m *MockTranscriptionClientFixed) TranscribeVideo(ctx context.Context, videoURL string) (string, error) {
	return "", nil
}

type MockStorageClientFixed struct {
	imageHash string
}

func (m *MockStorageClientFixed) UploadImageWithHash(ctx context.Context, bucket, path, sourceURL string, data []byte) (string, error) {
	return "https://storage.example.com/" + path, nil
}

func (m *MockStorageClientFixed) GetImageByHash(ctx context.Context, hash string) (*storage.ExistingImageResponse, error) {
	return &storage.ExistingImageResponse{
		ID:          uuid.New().String(),
		ContentHash: hash,
		StoragePath: "pancakes/" + hash,
	}, nil
}

// Test database setup

type testDBConnection struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

func setupTestDB(ctx context.Context) (*testDBConnection, func()) {
	// Get test database URL from environment or use default
	databaseURL := "postgres://postgres:postgres@localhost:5432/remy_test?sslmode=disable"

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		pool.Close()
	}

	return &testDBConnection{
		pool:    pool,
		queries: generated.New(pool),
	}, cleanup
}

// progressBroadcasterAdapter adapts integration.ProgressUpdate to worker.ProgressUpdate
type progressBroadcasterAdapter struct {
	inner *MockProgressBroadcaster
}

func (a *progressBroadcasterAdapter) Broadcast(userID string, update worker.ProgressUpdate) error {
	// Convert worker.ProgressUpdate to integration.ProgressUpdate
	integrationUpdate := ProgressUpdate{
		JobID:   update.JobID,
		Status:  update.Status,
		Message: update.Message,
	}
	return a.inner.Broadcast(userID, integrationUpdate)
}
