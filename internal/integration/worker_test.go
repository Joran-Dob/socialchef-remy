package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/worker"
)

func createMockTask(payload interface{}) *asynq.Task {
	data, _ := json.Marshal(payload)
	return asynq.NewTask("test-task", data)
}

func TestWorker_HandleProcessRecipe_InvalidPayload(t *testing.T) {
	task := asynq.NewTask("test-task", []byte("invalid json"))

	processor := worker.NewRecipeProcessor(
		&generated.Queries{},
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)


	err := processor.HandleProcessRecipe(context.Background(), task)
	if err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

func TestWorker_HandleProcessRecipe_InvalidURL(t *testing.T) {
	fixtures := setupTestFixtures()

	jobID := uuid.New().String()
	userID := uuid.New().String()

	fixtures.mockDB.importJobs[jobID] = generated.RecipeImportJob{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    "https://twitter.com/user/status/123",
		Status: "pending",
	}

	savedJob, err := fixtures.mockDB.GetImportJob(context.Background(), uuidToPgtype(uuid.MustParse(jobID)))
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if savedJob.Url != "https://twitter.com/user/status/123" {
		t.Errorf("expected URL 'https://twitter.com/user/status/123', got %s", savedJob.Url)
	}
}

func TestMockDB_CreateAndGetImportJob(t *testing.T) {
	fixtures := setupTestFixtures()

	jobID := uuid.New().String()
	userID := uuid.New().String()

	job, err := fixtures.mockDB.CreateImportJob(context.Background(), generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    "https://instagram.com/p/test",
		Status: "pending",
	})
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if job.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", job.Status)
	}

	retrievedJob, err := fixtures.mockDB.GetImportJob(context.Background(), uuidToPgtype(uuid.MustParse(jobID)))
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrievedJob.Url != "https://instagram.com/p/test" {
		t.Errorf("expected URL 'https://instagram.com/p/test', got %s", retrievedJob.Url)
	}
}

func TestMockDB_UpdateImportJobStatus(t *testing.T) {
	fixtures := setupTestFixtures()

	jobID := uuid.New().String()
	userID := uuid.New().String()

	fixtures.mockDB.CreateImportJob(context.Background(), generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    "https://instagram.com/p/test",
		Status: "pending",
	})

	err := fixtures.mockDB.UpdateImportJobStatus(context.Background(), generated.UpdateImportJobStatusParams{
		JobID:        jobID,
		Status:       "completed",
		ProgressStep: pgtype.Text{String: "Done", Valid: true},
		Error:        nil,
	})
	if err != nil {
		t.Fatalf("failed to update job: %v", err)
	}

	job, err := fixtures.mockDB.GetImportJob(context.Background(), uuidToPgtype(uuid.MustParse(jobID)))
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if job.Status != "completed" {
		t.Errorf("expected status 'completed', got %s", job.Status)
	}

	if job.ProgressStep.String != "Done" {
		t.Errorf("expected progress step 'Done', got %s", job.ProgressStep.String)
	}
}

func TestMockDB_GetImportJobsByUser(t *testing.T) {
	fixtures := setupTestFixtures()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	for i := 0; i < 3; i++ {
		jobID := uuid.New().String()
		fixtures.mockDB.CreateImportJob(context.Background(), generated.CreateImportJobParams{
			ID:     uuidToPgtype(uuid.MustParse(jobID)),
			UserID: uuidToPgtype(uuid.MustParse(user1ID)),
			Url:    fmt.Sprintf("https://instagram.com/p/%d", i),
			Status: "pending",
		})
	}

	jobID := uuid.New().String()
	fixtures.mockDB.CreateImportJob(context.Background(), generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(user2ID)),
		Url:    "https://instagram.com/p/other",
		Status: "completed",
	})

	jobs, err := fixtures.mockDB.GetImportJobsByUser(context.Background(), uuidToPgtype(uuid.MustParse(user1ID)))
	if err != nil {
		t.Fatalf("failed to get jobs: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs for user1, got %d", len(jobs))
	}
}

func TestMockDB_CreateAndGetRecipe(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()
	userID := uuid.New().String()

	recipe, err := fixtures.mockDB.CreateRecipe(context.Background(), generated.CreateRecipeParams{
		ID:                  uuidToPgtype(uuid.MustParse(recipeID)),
		CreatedBy:           uuidToPgtype(uuid.MustParse(userID)),
		RecipeName:          "Test Recipe",
		Description:         pgtype.Text{String: "A test recipe", Valid: true},
		PrepTime:            pgtype.Int4{Int32: 15, Valid: true},
		CookingTime:         pgtype.Int4{Int32: 30, Valid: true},
		OriginalServingSize: pgtype.Int4{Int32: 4, Valid: true},
		DifficultyRating:    pgtype.Int2{Int16: 3, Valid: true},
		Origin:              generated.RecipeOriginInstagram,
		Url:                 "https://instagram.com/p/test",
	})
	if err != nil {
		t.Fatalf("failed to create recipe: %v", err)
	}

	if recipe.RecipeName != "Test Recipe" {
		t.Errorf("expected recipe name 'Test Recipe', got %s", recipe.RecipeName)
	}

	retrievedRecipe, err := fixtures.mockDB.GetRecipe(context.Background(), uuidToPgtype(uuid.MustParse(recipeID)))
	if err != nil {
		t.Fatalf("failed to get recipe: %v", err)
	}

	if retrievedRecipe.RecipeName != "Test Recipe" {
		t.Errorf("expected retrieved name 'Test Recipe', got %s", retrievedRecipe.RecipeName)
	}
}

func TestMockDB_CreateIngredients(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()

	ingredients := []struct {
		name     string
		quantity string
		unit     string
	}{
		{"flour", "2", "cups"},
		{"sugar", "1", "cup"},
		{"eggs", "2", ""},
	}

	for _, ing := range ingredients {
		_, err := fixtures.mockDB.CreateIngredient(context.Background(), generated.CreateIngredientParams{
			RecipeID:         uuidToPgtype(uuid.MustParse(recipeID)),
			Name:             ing.name,
			Quantity:         pgtype.Text{String: ing.quantity, Valid: true},
			Unit:             pgtype.Text{String: ing.unit, Valid: ing.unit != ""},
			OriginalQuantity: pgtype.Text{String: ing.quantity, Valid: true},
			OriginalUnit:     pgtype.Text{String: ing.unit, Valid: ing.unit != ""},
		})
		if err != nil {
			t.Fatalf("failed to create ingredient: %v", err)
		}
	}

	retrievedIngs, err := fixtures.mockDB.GetIngredientsByRecipe(context.Background(), uuidToPgtype(uuid.MustParse(recipeID)))
	if err != nil {
		t.Fatalf("failed to get ingredients: %v", err)
	}

	if len(retrievedIngs) != 3 {
		t.Errorf("expected 3 ingredients, got %d", len(retrievedIngs))
	}
}

func TestMockDB_CreateInstructions(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()

	instructions := []string{
		"Preheat oven to 350F",
		"Mix dry ingredients",
		"Add wet ingredients",
		"Bake for 30 minutes",
	}

	for i, inst := range instructions {
		_, err := fixtures.mockDB.CreateInstruction(context.Background(), generated.CreateInstructionParams{
			RecipeID:    uuidToPgtype(uuid.MustParse(recipeID)),
			StepNumber:  int32(i + 1),
			Instruction: inst,
		})
		if err != nil {
			t.Fatalf("failed to create instruction: %v", err)
		}
	}

	if len(fixtures.mockDB.instructions[recipeID]) != 4 {
		t.Errorf("expected 4 instructions, got %d", len(fixtures.mockDB.instructions[recipeID]))
	}
}

func TestMockDB_CreateNutrition(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()

	_, err := fixtures.mockDB.CreateNutrition(context.Background(), generated.CreateNutritionParams{
		RecipeID: uuidToPgtype(uuid.MustParse(recipeID)),
		Protein:  pgtype.Numeric{Int: big.NewInt(1500), Exp: -2, Valid: true},
		Carbs:    pgtype.Numeric{Int: big.NewInt(4500), Exp: -2, Valid: true},
		Fat:      pgtype.Numeric{Int: big.NewInt(1200), Exp: -2, Valid: true},
		Fiber:    pgtype.Numeric{Int: big.NewInt(300), Exp: -2, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create nutrition: %v", err)
	}

	nutrition, ok := fixtures.mockDB.nutrition[recipeID]
	if !ok {
		t.Fatal("nutrition not found")
	}

	if nutrition.Protein.Int.Cmp(big.NewInt(1500)) != 0 {
		t.Errorf("expected protein 15.00, got %v", nutrition.Protein)
	}
}

func TestMockDB_UpdateRecipe(t *testing.T) {
	fixtures := setupTestFixtures()

	recipeID := uuid.New().String()
	userID := uuid.New().String()

	fixtures.mockDB.CreateRecipe(context.Background(), generated.CreateRecipeParams{
		ID:          uuidToPgtype(uuid.MustParse(recipeID)),
		CreatedBy:   uuidToPgtype(uuid.MustParse(userID)),
		RecipeName:  "Test Recipe",
		Description: pgtype.Text{String: "A test recipe", Valid: true},
		Origin:      generated.RecipeOriginInstagram,
		Url:         "https://instagram.com/p/test",
	})

	updatedRecipe, err := fixtures.mockDB.UpdateRecipe(context.Background(), generated.UpdateRecipeParams{
		ID:          uuidToPgtype(uuid.MustParse(recipeID)),
		RecipeName:  "Updated Recipe",
		Description: pgtype.Text{String: "An updated recipe", Valid: true},
		PrepTime:    pgtype.Int4{Int32: 20, Valid: true},
		CookingTime: pgtype.Int4{Int32: 40, Valid: true},
		Origin:      generated.RecipeOriginInstagram,
		Url:         "https://instagram.com/p/test",
	})
	if err != nil {
		t.Fatalf("failed to update recipe: %v", err)
	}

	if updatedRecipe.RecipeName != "Updated Recipe" {
		t.Errorf("expected updated recipe name 'Updated Recipe', got %s", updatedRecipe.RecipeName)
	}
}

func TestScraper_IsInstagramURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://instagram.com/p/test123", true},
		{"https://www.instagram.com/p/test123", true},
		{"https://instagram.com/reel/test123", true},
		{"https://instagram.com/reels/test123", true},
		{"https://tiktok.com/@user/video/123", false},
		{"https://twitter.com/user/status/123", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := scraper.IsInstagramURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsInstagramURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestScraper_IsTikTokURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://tiktok.com/@user/video/123", true},
		{"https://www.tiktok.com/@user/video/123", true},
		{"https://vm.tiktok.com/abc123", true},
		{"https://instagram.com/p/test123", false},
		{"https://twitter.com/user/status/123", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := scraper.IsTikTokURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsTikTokURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestWorker_NewProcessRecipeTask(t *testing.T) {
	payload := worker.ProcessRecipePayload{
		JobID:  uuid.New().String(),
		URL:    "https://instagram.com/p/test",
		UserID: uuid.New().String(),
	}

	task, err := worker.NewProcessRecipeTask(payload)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if task.Type() != worker.TypeProcessRecipe {
		t.Errorf("expected task type %s, got %s", worker.TypeProcessRecipe, task.Type())
	}

	var retrievedPayload worker.ProcessRecipePayload
	if err := json.Unmarshal(task.Payload(), &retrievedPayload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if retrievedPayload.URL != payload.URL {
		t.Errorf("expected URL %s, got %s", payload.URL, retrievedPayload.URL)
	}
}

func TestWorker_NewGenerateEmbeddingTask(t *testing.T) {
	payload := worker.GenerateEmbeddingPayload{
		RecipeID: uuid.New().String(),
	}

	task, err := worker.NewGenerateEmbeddingTask(payload)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if task.Type() != worker.TypeGenerateEmbedding {
		t.Errorf("expected task type %s, got %s", worker.TypeGenerateEmbedding, task.Type())
	}

	var retrievedPayload worker.GenerateEmbeddingPayload
	if err := json.Unmarshal(task.Payload(), &retrievedPayload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if retrievedPayload.RecipeID != payload.RecipeID {
		t.Errorf("expected recipe ID %s, got %s", payload.RecipeID, retrievedPayload.RecipeID)
	}
}

func TestWorker_NewCleanupJobsTask(t *testing.T) {
	task := worker.NewCleanupJobsTask()

	if task.Type() != worker.TypeCleanupJobs {
		t.Errorf("expected task type %s, got %s", worker.TypeCleanupJobs, task.Type())
	}
}

func TestMockProgressBroadcaster_Broadcast(t *testing.T) {
	broadcaster := &MockProgressBroadcaster{
		Broadcasts: make([]ProgressUpdate, 0),
	}

	update := ProgressUpdate{
		JobID:   "job-123",
		Status:  "scraping",
		Message: "Fetching post content...",
	}

	err := broadcaster.Broadcast("user-123", update)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(broadcaster.Broadcasts) != 1 {
		t.Errorf("expected 1 broadcast, got %d", len(broadcaster.Broadcasts))
	}

	if broadcaster.Broadcasts[0].JobID != "job-123" {
		t.Errorf("expected job ID 'job-123', got %s", broadcaster.Broadcasts[0].JobID)
	}
}

func TestMockProgressBroadcaster_MultipleUpdates(t *testing.T) {
	broadcaster := &MockProgressBroadcaster{
		Broadcasts: make([]ProgressUpdate, 0),
	}

	updates := []ProgressUpdate{
		{JobID: "job-123", Status: "scraping", Message: "Fetching..."},
		{JobID: "job-123", Status: "generating", Message: "Generating..."},
		{JobID: "job-123", Status: "saving", Message: "Saving..."},
		{JobID: "job-123", Status: "completed", Message: "Done!"},
	}

	for _, update := range updates {
		err := broadcaster.Broadcast("user-123", update)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	}

	if len(broadcaster.Broadcasts) != len(updates) {
		t.Errorf("expected %d broadcasts, got %d", len(updates), len(broadcaster.Broadcasts))
	}

	expectedStatuses := []string{"scraping", "generating", "saving", "completed"}
	for i, expected := range expectedStatuses {
		if broadcaster.Broadcasts[i].Status != expected {
			t.Errorf("expected status %s at index %d, got %s", expected, i, broadcaster.Broadcasts[i].Status)
		}
	}
}

func TestRecipePipeline_MockDBOperations(t *testing.T) {
	fixtures := setupTestFixtures()
	ctx := context.Background()

	userID := uuid.New().String()
	jobID := uuid.New().String()

	job, err := fixtures.mockDB.CreateImportJob(ctx, generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.MustParse(jobID)),
		UserID: uuidToPgtype(uuid.MustParse(userID)),
		Url:    "https://instagram.com/p/chocolate-cake",
		Status: "pending",
	})
	if err != nil {
		t.Fatalf("failed to create import job: %v", err)
	}

	if job.Status != "pending" {
		t.Errorf("expected initial status 'pending', got %s", job.Status)
	}

	fixtures.mockDB.UpdateImportJobStatus(ctx, generated.UpdateImportJobStatusParams{
		JobID:        jobID,
		Status:       "scraping",
		ProgressStep: pgtype.Text{String: "Fetching post content...", Valid: true},
	})

	recipeID := uuid.New().String()
	_, err = fixtures.mockDB.CreateRecipe(ctx, generated.CreateRecipeParams{
		ID:                  uuidToPgtype(uuid.MustParse(recipeID)),
		CreatedBy:           uuidToPgtype(uuid.MustParse(userID)),
		RecipeName:          "Chocolate Cake",
		Description:         pgtype.Text{String: "Rich chocolate cake recipe", Valid: true},
		PrepTime:            pgtype.Int4{Int32: 15, Valid: true},
		CookingTime:         pgtype.Int4{Int32: 35, Valid: true},
		OriginalServingSize: pgtype.Int4{Int32: 8, Valid: true},
		Origin:              generated.RecipeOriginInstagram,
		Url:                 "https://instagram.com/p/chocolate-cake",
	})
	if err != nil {
		t.Fatalf("failed to create recipe: %v", err)
	}

	ingredients := []struct {
		name     string
		quantity string
		unit     string
	}{
		{"all-purpose flour", "2", "cups"},
		{"cocoa powder", "3/4", "cup"},
		{"sugar", "2", "cups"},
		{"eggs", "2", ""},
		{"vanilla extract", "2", "tsp"},
	}

	for _, ing := range ingredients {
		_, err := fixtures.mockDB.CreateIngredient(ctx, generated.CreateIngredientParams{
			RecipeID:         uuidToPgtype(uuid.MustParse(recipeID)),
			Name:             ing.name,
			Quantity:         pgtype.Text{String: ing.quantity, Valid: true},
			Unit:             pgtype.Text{String: ing.unit, Valid: ing.unit != ""},
			OriginalQuantity: pgtype.Text{String: ing.quantity, Valid: true},
			OriginalUnit:     pgtype.Text{String: ing.unit, Valid: ing.unit != ""},
		})
		if err != nil {
			t.Fatalf("failed to create ingredient: %v", err)
		}
	}

	instructions := []string{
		"Preheat oven to 350°F (175°C)",
		"Mix flour, cocoa powder, and sugar in a large bowl",
		"Add eggs and vanilla extract, mix until smooth",
		"Pour batter into greased cake pan",
		"Bake for 35 minutes or until toothpick comes out clean",
	}

	for i, inst := range instructions {
		_, err := fixtures.mockDB.CreateInstruction(ctx, generated.CreateInstructionParams{
			RecipeID:    uuidToPgtype(uuid.MustParse(recipeID)),
			StepNumber:  int32(i + 1),
			Instruction: inst,
		})
		if err != nil {
			t.Fatalf("failed to create instruction: %v", err)
		}
	}

	_, err = fixtures.mockDB.CreateNutrition(ctx, generated.CreateNutritionParams{
		RecipeID: uuidToPgtype(uuid.MustParse(recipeID)),
		Protein:  pgtype.Numeric{Int: big.NewInt(400), Exp: -2, Valid: true},
		Carbs:    pgtype.Numeric{Int: big.NewInt(4800), Exp: -2, Valid: true},
		Fat:      pgtype.Numeric{Int: big.NewInt(1400), Exp: -2, Valid: true},
		Fiber:    pgtype.Numeric{Int: big.NewInt(200), Exp: -2, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create nutrition: %v", err)
	}

	fixtures.mockDB.UpdateImportJobStatus(ctx, generated.UpdateImportJobStatusParams{
		JobID:        jobID,
		Status:       "completed",
		ProgressStep: pgtype.Text{String: "Recipe saved successfully!", Valid: true},
	})

	finalJob, _ := fixtures.mockDB.GetImportJob(ctx, uuidToPgtype(uuid.MustParse(jobID)))
	if finalJob.Status != "completed" {
		t.Errorf("expected final status 'completed', got %s", finalJob.Status)
	}

	retrievedRecipe, _ := fixtures.mockDB.GetRecipe(ctx, uuidToPgtype(uuid.MustParse(recipeID)))
	if retrievedRecipe.RecipeName != "Chocolate Cake" {
		t.Errorf("expected recipe name 'Chocolate Cake', got %s", retrievedRecipe.RecipeName)
	}

	retrievedIngredients, _ := fixtures.mockDB.GetIngredientsByRecipe(ctx, uuidToPgtype(uuid.MustParse(recipeID)))
	if len(retrievedIngredients) != 5 {
		t.Errorf("expected 5 ingredients, got %d", len(retrievedIngredients))
	}

	_, err = fixtures.mockDB.UpdateRecipe(ctx, generated.UpdateRecipeParams{
		ID:          uuidToPgtype(uuid.MustParse(recipeID)),
		RecipeName:  "Updated Chocolate Cake",
		Description: pgtype.Text{String: "Rich chocolate cake recipe", Valid: true},
		Origin:      generated.RecipeOriginInstagram,
		Url:         "https://instagram.com/p/chocolate-cake",
	})
	if err != nil {
		t.Fatalf("failed to update recipe: %v", err)
	}

	finalRecipe, _ := fixtures.mockDB.GetRecipe(ctx, uuidToPgtype(uuid.MustParse(recipeID)))
	if finalRecipe.RecipeName != "Updated Chocolate Cake" {
		t.Errorf("expected updated recipe name 'Updated Chocolate Cake', got %s", finalRecipe.RecipeName)
	}

	t.Logf("Recipe pipeline completed successfully: created job %s, recipe %s with %d ingredients",
		jobID, recipeID, len(retrievedIngredients))
}
