package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/groq"
	recipeservice "github.com/socialchef/remy/internal/services/recipe"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/services/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

type MockDB struct {
	mock.Mock
}

func (m *MockDB) CreateImportJob(ctx context.Context, arg generated.CreateImportJobParams) (generated.RecipeImportJob, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipeImportJob), args.Error(1)
}

func (m *MockDB) GetImportJob(ctx context.Context, id pgtype.UUID) (generated.RecipeImportJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(generated.RecipeImportJob), args.Error(1)
}

func (m *MockDB) GetImportJobsByUser(ctx context.Context, userID pgtype.UUID) ([]generated.RecipeImportJob, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]generated.RecipeImportJob), args.Error(1)
}

func (m *MockDB) UpdateImportJobStatus(ctx context.Context, arg generated.UpdateImportJobStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) CreateRecipe(ctx context.Context, arg generated.CreateRecipeParams) (generated.Recipe, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.Recipe), args.Error(1)
}

func (m *MockDB) GetRecipe(ctx context.Context, id pgtype.UUID) (generated.Recipe, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(generated.Recipe), args.Error(1)
}

func (m *MockDB) UpdateRecipe(ctx context.Context, arg generated.UpdateRecipeParams) (generated.Recipe, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.Recipe), args.Error(1)
}

func (m *MockDB) CreateIngredient(ctx context.Context, arg generated.CreateIngredientParams) (generated.RecipeIngredient, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipeIngredient), args.Error(1)
}

func (m *MockDB) CreateInstruction(ctx context.Context, arg generated.CreateInstructionParams) (generated.RecipeInstruction, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipeInstruction), args.Error(1)
}

func (m *MockDB) UpdateInstructionRich(ctx context.Context, arg generated.UpdateInstructionRichParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) CreateInstructionIngredient(ctx context.Context, arg generated.CreateInstructionIngredientParams) (generated.InstructionIngredient, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.InstructionIngredient), args.Error(1)
}

func (m *MockDB) GetInstructionIngredientsByInstruction(ctx context.Context, instructionID pgtype.UUID) ([]generated.InstructionIngredient, error) {
	args := m.Called(ctx, instructionID)
	return args.Get(0).([]generated.InstructionIngredient), args.Error(1)
}

func (m *MockDB) GetInstructionIngredientsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.InstructionIngredient, error) {
	args := m.Called(ctx, recipeID)
	return args.Get(0).([]generated.InstructionIngredient), args.Error(1)
}

func (m *MockDB) DeleteInstructionIngredientsByInstruction(ctx context.Context, instructionID pgtype.UUID) error {
	args := m.Called(ctx, instructionID)
	return args.Error(0)
}

func (m *MockDB) CreateNutrition(ctx context.Context, arg generated.CreateNutritionParams) (generated.RecipeNutrition, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipeNutrition), args.Error(1)
}

func (m *MockDB) GetIngredientsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeIngredient, error) {
	args := m.Called(ctx, recipeID)
	return args.Get(0).([]generated.RecipeIngredient), args.Error(1)
}

func (m *MockDB) GetInstructionsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeInstruction, error) {
	args := m.Called(ctx, recipeID)
	return args.Get(0).([]generated.RecipeInstruction), args.Error(1)
}

func (m *MockDB) DeleteOldImportJobs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) DeleteStaleImportJobs(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) CreateRecipeImage(ctx context.Context, arg generated.CreateRecipeImageParams) (generated.RecipeImage, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipeImage), args.Error(1)
}

func (m *MockDB) UpdateRecipeThumbnail(ctx context.Context, arg generated.UpdateRecipeThumbnailParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetSocialMediaOwnerByOrigin(ctx context.Context, arg generated.GetSocialMediaOwnerByOriginParams) (generated.SocialMediaOwner, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.SocialMediaOwner), args.Error(1)
}

func (m *MockDB) CreateSocialMediaOwner(ctx context.Context, arg generated.CreateSocialMediaOwnerParams) (generated.SocialMediaOwner, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.SocialMediaOwner), args.Error(1)
}
func (m *MockDB) UpdateRecipeEmbedding(ctx context.Context, arg generated.UpdateRecipeEmbeddingParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// Category methods
func (m *MockDB) GetOrCreateCuisineCategory(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockDB) AddRecipeCuisineCategory(ctx context.Context, arg generated.AddRecipeCuisineCategoryParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetOrCreateMealType(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockDB) AddRecipeMealType(ctx context.Context, arg generated.AddRecipeMealTypeParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetOrCreateOccasion(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockDB) AddRecipeOccasion(ctx context.Context, arg generated.AddRecipeOccasionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetOrCreateDietaryRestriction(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockDB) AddRecipeDietaryRestriction(ctx context.Context, arg generated.AddRecipeDietaryRestrictionParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetOrCreateEquipment(ctx context.Context, name string) (pgtype.UUID, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(pgtype.UUID), args.Error(1)
}

func (m *MockDB) AddRecipeEquipment(ctx context.Context, arg generated.AddRecipeEquipmentParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) GetCuisineCategoriesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) GetMealTypesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) GetOccasionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) GetDietaryRestrictionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) GetEquipmentByUser(ctx context.Context, userID pgtype.UUID) ([]string, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDB) CreateBulkImportJob(ctx context.Context, arg generated.CreateBulkImportJobParams) (generated.BulkImportJob, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.BulkImportJob), args.Error(1)
}

func (m *MockDB) GetBulkImportJobByJobID(ctx context.Context, jobID string) (generated.BulkImportJob, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(generated.BulkImportJob), args.Error(1)
}

func (m *MockDB) UpdateBulkImportJobStatus(ctx context.Context, arg generated.UpdateBulkImportJobStatusParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) UpdateImportJobWithBulkID(ctx context.Context, arg generated.UpdateImportJobWithBulkIDParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) IncrementBulkImportCounters(ctx context.Context, arg generated.IncrementBulkImportCountersParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDB) CreateRecipePart(ctx context.Context, arg generated.CreateRecipePartParams) (generated.RecipePart, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(generated.RecipePart), args.Error(1)
}

func (m *MockDB) GetRecipeParts(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipePart, error) {
	args := m.Called(ctx, recipeID)
	return args.Get(0).([]generated.RecipePart), args.Error(1)
}

type MockInstagramScraper struct {
	mock.Mock
}

func (m *MockInstagramScraper) Scrape(ctx context.Context, postURL string) (*scraper.InstagramPost, error) {
	args := m.Called(ctx, postURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scraper.InstagramPost), args.Error(1)
}

type MockTikTokScraper struct {
	mock.Mock
}

func (m *MockTikTokScraper) Scrape(ctx context.Context, postURL string) (*scraper.TikTokPost, error) {
	args := m.Called(ctx, postURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scraper.TikTokPost), args.Error(1)
}

type MockFirecrawlScraper struct {
	mock.Mock
}

func (m *MockFirecrawlScraper) Scrape(ctx context.Context, postURL string) (*scraper.FirecrawlPost, error) {
	args := m.Called(ctx, postURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scraper.FirecrawlPost), args.Error(1)
}

type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

type MockTranscriptionClient struct {
	mock.Mock
}

func (m *MockTranscriptionClient) TranscribeVideo(ctx context.Context, videoURL string) (string, error) {
	args := m.Called(ctx, videoURL)
	return args.String(0), args.Error(1)
}

type MockGroqClient struct {
	mock.Mock
}

func (m *MockGroqClient) GenerateRecipe(ctx context.Context, caption, transcript, platform string) (*groq.Recipe, error) {
	args := m.Called(ctx, caption, transcript, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*groq.Recipe), args.Error(1)
}

func (m *MockGroqClient) GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error) {
	args := m.Called(ctx, prompt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.CategoryAIResponse), args.Error(1)
}

func (m *MockGroqClient) GenerateRichInstructions(ctx context.Context, recipe *groq.Recipe) (*recipeservice.RichInstructionResponse, error) {
	args := m.Called(ctx, recipe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*recipeservice.RichInstructionResponse), args.Error(1)
}

type MockStorageClient struct {
	mock.Mock
}

func (m *MockStorageClient) UploadImageWithHash(ctx context.Context, bucket, path, sourceURL string, data []byte) (string, error) {
	args := m.Called(ctx, bucket, path, sourceURL, data)
	return args.String(0), args.Error(1)
}

func (m *MockStorageClient) GetImageByHash(ctx context.Context, hash string) (*storage.ExistingImageResponse, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ExistingImageResponse), args.Error(1)
}

type MockBroadcaster struct {
	mock.Mock
}

func (m *MockBroadcaster) Broadcast(userID string, update ProgressUpdate) error {
	args := m.Called(userID, update)
	return args.Error(0)
}

// Tests

func TestHandleProcessRecipe_ValidRecipe(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_abc123/"

	payload := ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeProcessRecipe, payloadBytes)

	mockDB := new(MockDB)
	mockInsta := new(MockInstagramScraper)
	mockTikTok := new(MockTikTokScraper)
	mockOpenAI := new(MockOpenAIClient)
	mockTranscription := new(MockTranscriptionClient)
	mockGroq := new(MockGroqClient)
	mockStorage := new(MockStorageClient)
	mockBroadcaster := new(MockBroadcaster)

	processor := NewRecipeProcessor(
		mockDB, mockInsta, mockTikTok, nil, mockOpenAI, mockTranscription, mockGroq, mockStorage, mockBroadcaster, nil, nil,
	)

	// Expectations
	mockDB.On("UpdateImportJobStatus", ctx, mock.MatchedBy(func(arg generated.UpdateImportJobStatusParams) bool {
		return arg.JobID == jobID && arg.Status == "EXECUTING"
	})).Return(nil)

	// Set up mock image server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image-data"))
	}))
	defer ts.Close()

	mockInsta.On("Scrape", ctx, url).Return(&scraper.InstagramPost{
		Caption:  "Delicious Chocolate Cake Recipe! #baking #cake #delicious #homemade #recipe",
		ImageURL: ts.URL,
		VideoURL: "https://example.com/video.mp4",
	}, nil)

	mockTranscription.On("TranscribeVideo", mock.Anything, "https://example.com/video.mp4").Return("Mix flour and sugar, then add eggs.", nil)

	expectedRecipe := &groq.Recipe{
		RecipeName:  "Chocolate Cake",
		Description: "A delicious chocolate cake",
		Ingredients: []groq.Ingredient{
			{Name: "Flour", OriginalQuantity: "2", Quantity: "2", Unit: "cups"},
			{Name: "Sugar", OriginalQuantity: "1", Quantity: "1", Unit: "cup"},
		},
		Instructions: []groq.Instruction{
			{StepNumber: 1, Instruction: "Preheat oven to 350F"},
			{StepNumber: 2, Instruction: "Mix dry ingredients"},
		},
		Nutrition: groq.Nutrition{
			Protein: 10,
			Carbs:   20,
			Fat:     5,
			Fiber:   2,
		},
	}
	mockGroq.On("GenerateRecipe", ctx, mock.Anything, "Mix flour and sugar, then add eggs.", "instagram").Return(expectedRecipe, nil)

	mockDB.On("GetCuisineCategoriesByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetMealTypesByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetOccasionsByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetDietaryRestrictionsByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetEquipmentByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockGroq.On("GenerateCategories", mock.Anything, mock.Anything).Return(&ai.CategoryAIResponse{
		CuisineCategories:   []string{"Dessert"},
		MealTypes:           []string{"Snack"},
		Occasions:           []string{"Party"},
		DietaryRestrictions: []string{},
		Equipment:           []string{"Oven"},
	}, nil)

	mockGroq.On("GenerateRichInstructions", ctx, mock.Anything).Return(&recipeservice.RichInstructionResponse{
		Instructions: []recipeservice.RichInstruction{
			{StepNumber: 1, InstructionRich: "Preheat oven to {{ingredient:0}}"},
			{StepNumber: 2, InstructionRich: "Mix {{ingredient:1}} with {{ingredient:0}}"},
		},
		PromptVersion: 1,
	}, nil)

	recipeUUID := pgtype.UUID{Valid: true} // Simplified for mock
	mockDB.On("UpdateImportJobStatus", ctx, mock.Anything).Return(nil)
	mockDB.On("CreateRecipe", ctx, mock.Anything).Return(generated.Recipe{ID: recipeUUID, RecipeName: "Chocolate Cake"}, nil)
	mockDB.On("CreateIngredient", ctx, mock.Anything).Return(generated.RecipeIngredient{ID: pgtype.UUID{Valid: true}}, nil)
	mockDB.On("CreateInstruction", ctx, mock.Anything).Return(generated.RecipeInstruction{ID: pgtype.UUID{Valid: true}}, nil)
	mockDB.On("CreateNutrition", ctx, mock.Anything).Return(generated.RecipeNutrition{}, nil)
	mockDB.On("UpdateInstructionRich", ctx, mock.Anything).Return(nil)

	// Mock Image Processing
	mockStorage.On("UploadImageWithHash", ctx, "recipes", mock.Anything, ts.URL, mock.Anything).Return("https://public.com/image.jpg", nil)
	mockStorage.On("GetImageByHash", ctx, mock.Anything).Return(&storage.ExistingImageResponse{ID: uuid.New().String(), StoragePath: "path/to/img"}, nil)
	mockDB.On("CreateRecipeImage", ctx, mock.Anything).Return(generated.RecipeImage{ID: pgtype.UUID{Valid: true}}, nil)
	mockDB.On("UpdateRecipeThumbnail", ctx, mock.Anything).Return(nil)

	mockBroadcaster.On("Broadcast", userID, mock.Anything).Return(nil)

	mockDB.On("GetOrCreateCuisineCategory", ctx, mock.Anything).Return(pgtype.UUID{Valid: true}, nil)
	mockDB.On("AddRecipeCuisineCategory", ctx, mock.Anything).Return(nil)
	mockDB.On("GetOrCreateMealType", ctx, mock.Anything).Return(pgtype.UUID{Valid: true}, nil)
	mockDB.On("AddRecipeMealType", ctx, mock.Anything).Return(nil)
	mockDB.On("GetOrCreateOccasion", ctx, mock.Anything).Return(pgtype.UUID{Valid: true}, nil)
	mockDB.On("AddRecipeOccasion", ctx, mock.Anything).Return(nil)
	mockDB.On("GetOrCreateEquipment", ctx, mock.Anything).Return(pgtype.UUID{Valid: true}, nil)
	mockDB.On("AddRecipeEquipment", ctx, mock.Anything).Return(nil)

	err := processor.HandleProcessRecipe(ctx, task)

	// Assert
	assert.NoError(t, err)
	mockDB.AssertExpectations(t)
	mockInsta.AssertExpectations(t)
	mockTranscription.AssertExpectations(t)
	mockGroq.AssertExpectations(t)
}

func TestHandleProcessRecipe_ContentValidationFails(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_abc123/"

	payload := ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeProcessRecipe, payloadBytes)

	mockDB := new(MockDB)
	mockInsta := new(MockInstagramScraper)
	mockBroadcaster := new(MockBroadcaster)

	processor := NewRecipeProcessor(
		mockDB, mockInsta, nil, nil, nil, nil, nil, nil, mockBroadcaster, nil, nil,
	)

	mockDB.On("UpdateImportJobStatus", ctx, mock.Anything).Return(nil)
	mockInsta.On("Scrape", ctx, url).Return(&scraper.InstagramPost{
		Caption: "Just a photo of my cat #cats",
	}, nil)

	mockDB.On("UpdateImportJobStatus", ctx, mock.MatchedBy(func(arg generated.UpdateImportJobStatusParams) bool {
		return arg.Status == "FAILED"
	})).Return(nil)
	mockBroadcaster.On("Broadcast", userID, mock.Anything).Return(nil)

	err := processor.HandleProcessRecipe(ctx, task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Content validation failed")
}

func TestHandleProcessRecipe_TranscriptionFails(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_abc123/"

	payload := ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeProcessRecipe, payloadBytes)

	mockDB := new(MockDB)
	mockInsta := new(MockInstagramScraper)
	mockTranscription := new(MockTranscriptionClient)
	mockBroadcaster := new(MockBroadcaster)

	processor := NewRecipeProcessor(
		mockDB, mockInsta, nil, nil, nil, mockTranscription, nil, nil, mockBroadcaster, nil, nil,
	)

	mockDB.On("UpdateImportJobStatus", ctx, mock.Anything).Return(nil)
	mockInsta.On("Scrape", ctx, url).Return(&scraper.InstagramPost{
		Caption:  "Recipe in video! This is a very long caption to pass the content validation check that requires at least 30 characters. #cooking #recipe",
		VideoURL: "https://example.com/video.mp4",
	}, nil)

	mockTranscription.On("TranscribeVideo", mock.Anything, "https://example.com/video.mp4").Return("", fmt.Errorf("api error"))

	mockDB.On("UpdateImportJobStatus", ctx, mock.MatchedBy(func(arg generated.UpdateImportJobStatusParams) bool {
		return arg.Status == "FAILED"
	})).Return(nil)
	mockBroadcaster.On("Broadcast", userID, mock.Anything).Return(nil)

	err := processor.HandleProcessRecipe(ctx, task)

	assert.Error(t, err)
	assert.Equal(t, "api error", err.Error())
}

func TestHandleProcessRecipe_OutputValidationFails(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New().String()
	userID := uuid.New().String()
	url := "https://www.instagram.com/p/C_abc123/"

	payload := ProcessRecipePayload{
		JobID:  jobID,
		UserID: userID,
		URL:    url,
	}
	payloadBytes, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeProcessRecipe, payloadBytes)

	mockDB := new(MockDB)
	mockInsta := new(MockInstagramScraper)
	mockGroq := new(MockGroqClient)
	mockBroadcaster := new(MockBroadcaster)

	processor := NewRecipeProcessor(
		mockDB, mockInsta, nil, nil, nil, nil, mockGroq, nil, mockBroadcaster, nil, nil,
	)

	mockDB.On("UpdateImportJobStatus", ctx, mock.Anything).Return(nil)
	mockInsta.On("Scrape", ctx, url).Return(&scraper.InstagramPost{
		Caption: "Ingredients: Water, Salt. Step 1: Mix. Step 2: Done.",
	}, nil)

	// Low quality recipe
	mockGroq.On("GenerateRecipe", ctx, mock.Anything, mock.Anything, "instagram").Return(&groq.Recipe{
		RecipeName: "Water",
		Ingredients: []groq.Ingredient{
			{Name: "N/A", OriginalQuantity: "some"},
		},
		Instructions: []groq.Instruction{
			{StepNumber: 1, Instruction: "Placeholder"},
		},
	}, nil)

	mockDB.On("GetCuisineCategoriesByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetMealTypesByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetOccasionsByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetDietaryRestrictionsByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockDB.On("GetEquipmentByUser", mock.Anything, mock.Anything).Return([]string{}, nil)
	mockGroq.On("GenerateCategories", mock.Anything, mock.Anything).Return(&ai.CategoryAIResponse{}, nil)

	mockGroq.On("GenerateRichInstructions", ctx, mock.Anything).Return(&recipeservice.RichInstructionResponse{
		Instructions: []recipeservice.RichInstruction{
			{StepNumber: 1, InstructionRich: ""},
		},
		PromptVersion: 1,
	}, nil)

	mockDB.On("UpdateImportJobStatus", ctx, mock.Anything).Return(nil)
	mockDB.On("UpdateImportJobStatus", ctx, mock.MatchedBy(func(arg generated.UpdateImportJobStatusParams) bool {
		return arg.Status == "FAILED"
	})).Return(nil)
	mockBroadcaster.On("Broadcast", userID, mock.Anything).Return(nil)

	err := processor.HandleProcessRecipe(ctx, task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Recipe validation failed")
}

func TestSaveInstructionIngredients(t *testing.T) {
	ctx := context.Background()

	instructionID1 := parseUUID("00000000-0000-0000-0000-000000000001")
	instructionID2 := parseUUID("00000000-0000-0000-0000-000000000002")
	ingredientID1 := parseUUID("00000000-0000-0000-0000-000000000001")
	ingredientID2 := parseUUID("00000000-0000-0000-0000-000000000002")
	ingredientID3 := parseUUID("00000000-0000-0000-0000-000000000003")

	tests := []struct {
		name               string
		savedInstructions  []generated.RecipeInstruction
		savedIngredientIDs []string
		ingredients        []recipeservice.Ingredient
		recipeInstructions []recipeservice.Instruction
		expectCreateCall   bool
		expectCreateParams []generated.CreateInstructionIngredientParams
		expectError        bool
		description        string
	}{
		{
			name: "HappyPath",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
				{ID: instructionID2, StepNumber: 2},
			},
			savedIngredientIDs: []string{ingredientID1.String(), ingredientID2.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
				{Name: "eggs"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "300ml"},
					},
				},
				{
					StepNumber: 2,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "eggs", QuantityUsed: "2"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "300ml", Valid: true},
				},
				{
					InstructionID: instructionID2,
					IngredientID:  ingredientID2,
					StepQuantity:  pgtype.Text{String: "2", Valid: true},
				},
			},
			expectError: false,
			description: "Valid step-ingredient mappings, all ingredients found",
		},
		{
			name: "UnknownIngredient",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "300ml"},
						{IngredientName: "unknown_sauce", QuantityUsed: "50ml"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "300ml", Valid: true},
				},
			},
			expectError: false,
			description: "AI returns ingredient not in list, should skip and log",
		},
		{
			name: "DuplicateIngredients",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "200ml"},
						{IngredientName: "milk", QuantityUsed: "100ml"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "200ml + 100ml", Valid: true},
				},
			},
			expectError: false,
			description: "Same ingredient appears twice in same step, should combine quantities",
		},
		{
			name: "EmptyIngredientsUsed",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
				{ID: instructionID2, StepNumber: 2},
			},
			savedIngredientIDs: []string{ingredientID1.String(), ingredientID2.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
				{Name: "eggs"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "300ml"},
					},
				},
				{
					StepNumber:      2,
					IngredientsUsed: []recipeservice.StepIngredient{},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "300ml", Valid: true},
				},
			},
			expectError: false,
			description: "Step with no ingredients, should handle gracefully",
		},
		{
			name: "QuantityMismatch",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "100ml"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "100ml", Valid: true},
				},
			},
			expectError: false,
			description: "Step quantities don't sum to total, should log warning",
		},
		{
			name: "CaseInsensitiveMatching",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "Milk", QuantityUsed: "300ml"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "300ml", Valid: true},
				},
			},
			expectError: false,
			description: "Milk should match ingredient milk",
		},
		{
			name: "EmptyQuantity",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "salt"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "salt", QuantityUsed: ""},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{Valid: false},
				},
			},
			expectError: false,
			description: "QuantityUsed is empty string, should store as NULL",
		},
		{
			name:               "EmptyInputs",
			savedInstructions:  []generated.RecipeInstruction{},
			savedIngredientIDs: []string{},
			ingredients:        []recipeservice.Ingredient{},
			recipeInstructions: []recipeservice.Instruction{},
			expectCreateCall:   false,
			expectCreateParams: []generated.CreateInstructionIngredientParams{},
			expectError:        false,
			description:        "All empty inputs, should return early with no error",
		},
		{
			name: "DuplicateWithEmptyQuantity",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
			},
			savedIngredientIDs: []string{ingredientID1.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "pepper"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "pepper", QuantityUsed: ""},
						{IngredientName: "pepper", QuantityUsed: "1 tsp"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "1 tsp", Valid: true},
				},
			},
			expectError: false,
			description: "Duplicate ingredients where first has empty quantity, should use non-empty quantity",
		},
		{
			name: "MultipleStepsWithMixedIngredients",
			savedInstructions: []generated.RecipeInstruction{
				{ID: instructionID1, StepNumber: 1},
				{ID: instructionID2, StepNumber: 2},
			},
			savedIngredientIDs: []string{ingredientID1.String(), ingredientID2.String(), ingredientID3.String()},
			ingredients: []recipeservice.Ingredient{
				{Name: "milk"},
				{Name: "flour"},
				{Name: "eggs"},
			},
			recipeInstructions: []recipeservice.Instruction{
				{
					StepNumber: 1,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "milk", QuantityUsed: "200ml"},
						{IngredientName: "flour", QuantityUsed: "100g"},
					},
				},
				{
					StepNumber: 2,
					IngredientsUsed: []recipeservice.StepIngredient{
						{IngredientName: "eggs", QuantityUsed: "2"},
						{IngredientName: "milk", QuantityUsed: "50ml"},
					},
				},
			},
			expectCreateCall: true,
			expectCreateParams: []generated.CreateInstructionIngredientParams{
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "200ml", Valid: true},
				},
				{
					InstructionID: instructionID1,
					IngredientID:  ingredientID2,
					StepQuantity:  pgtype.Text{String: "100g", Valid: true},
				},
				{
					InstructionID: instructionID2,
					IngredientID:  ingredientID3,
					StepQuantity:  pgtype.Text{String: "2", Valid: true},
				},
				{
					InstructionID: instructionID2,
					IngredientID:  ingredientID1,
					StepQuantity:  pgtype.Text{String: "50ml", Valid: true},
				},
			},
			expectError: false,
			description: "Multiple steps with shared and unique ingredients",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			p := &RecipeProcessor{db: mockDB}

			if tt.expectCreateCall && len(tt.expectCreateParams) > 0 {
				mockDB.On("CreateInstructionIngredient", ctx, mock.Anything).Return(generated.InstructionIngredient{}, nil).Times(len(tt.expectCreateParams))
			}

			err := p.saveInstructionIngredients(ctx, tt.savedInstructions, tt.savedIngredientIDs, tt.ingredients, tt.recipeInstructions)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.description)
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.description)
			}

			if tt.expectCreateCall {
				mockDB.AssertExpectations(t)

				if len(tt.expectCreateParams) > 0 {
					mockDB.AssertCalled(t, "CreateInstructionIngredient", ctx, mock.Anything)
					for _, call := range mockDB.Calls {
						if call.Method == "CreateInstructionIngredient" {
							params := call.Arguments[1].(generated.CreateInstructionIngredientParams)

							found := false
							for _, expected := range tt.expectCreateParams {
								if params.InstructionID == expected.InstructionID &&
									params.IngredientID == expected.IngredientID &&
									params.StepQuantity.Valid == expected.StepQuantity.Valid {
									if !expected.StepQuantity.Valid {
										found = true
										break
									}
									if params.StepQuantity.String == expected.StepQuantity.String {
										found = true
										break
									}
								}
							}
							assert.True(t, found, "CreateInstructionIngredient called with unexpected params: %+v", params)
						}
					}
				}
			} else {
				mockDB.AssertNotCalled(t, "CreateInstructionIngredient")
			}
		})
	}
}
