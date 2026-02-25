// Package integration provides integration tests for the Remy backend service.
// These tests use mocked external dependencies to avoid real API calls.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Database Interfaces (to allow mocking)
// ============================================================================

// DBQueries defines the interface for database operations used by the API and worker
type DBQueries interface {
	CreateImportJob(ctx context.Context, arg generated.CreateImportJobParams) (generated.RecipeImportJob, error)
	GetImportJob(ctx context.Context, id pgtype.UUID) (generated.RecipeImportJob, error)
	GetImportJobsByUser(ctx context.Context, userID pgtype.UUID) ([]generated.RecipeImportJob, error)
	UpdateImportJobStatus(ctx context.Context, arg generated.UpdateImportJobStatusParams) error
	CreateRecipe(ctx context.Context, arg generated.CreateRecipeParams) (generated.Recipe, error)
	GetRecipe(ctx context.Context, id pgtype.UUID) (generated.Recipe, error)
	UpdateRecipe(ctx context.Context, arg generated.UpdateRecipeParams) (generated.Recipe, error)
	CreateIngredient(ctx context.Context, arg generated.CreateIngredientParams) (generated.RecipeIngredient, error)
	CreateInstruction(ctx context.Context, arg generated.CreateInstructionParams) (generated.RecipeInstruction, error)
	CreateNutrition(ctx context.Context, arg generated.CreateNutritionParams) (generated.RecipeNutrition, error)
	GetIngredientsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeIngredient, error)
	DeleteOldImportJobs(ctx context.Context) error
	DeleteStaleImportJobs(ctx context.Context) error
	CreateRecipeImage(ctx context.Context, arg generated.CreateRecipeImageParams) (generated.RecipeImage, error)
	UpdateRecipeThumbnail(ctx context.Context, arg generated.UpdateRecipeThumbnailParams) error
}

// ============================================================================
// Mock Implementations
// ============================================================================

// MockQueries implements DBQueries for testing
type MockQueries struct {
	importJobs   map[string]generated.RecipeImportJob
	recipes      map[string]generated.Recipe
	ingredients  map[string][]generated.RecipeIngredient
	instructions map[string][]generated.RecipeInstruction
	nutrition    map[string]generated.RecipeNutrition
}

// NewMockQueries creates a new MockQueries with default behavior
func NewMockQueries() *MockQueries {
	return &MockQueries{
		importJobs:   make(map[string]generated.RecipeImportJob),
		recipes:      make(map[string]generated.Recipe),
		ingredients:  make(map[string][]generated.RecipeIngredient),
		instructions: make(map[string][]generated.RecipeInstruction),
		nutrition:    make(map[string]generated.RecipeNutrition),
	}
}

// Helper function to convert uuid.UUID to pgtype.UUID
func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(u), Valid: true}
}

// withUserID adds a user ID to the request context
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, middleware.UserIDKey, userID)
}

// Implement database query methods
func (m *MockQueries) CreateImportJob(ctx context.Context, arg generated.CreateImportJobParams) (generated.RecipeImportJob, error) {
	jobID := uuid.UUID(arg.ID.Bytes).String()
	job := generated.RecipeImportJob{
		ID:        arg.ID,
		UserID:    arg.UserID,
		Url:       arg.Url,
		Status:    arg.Status,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	m.importJobs[jobID] = job
	return job, nil
}

func (m *MockQueries) GetImportJob(ctx context.Context, id pgtype.UUID) (generated.RecipeImportJob, error) {
	jobID := uuid.UUID(id.Bytes).String()
	if job, ok := m.importJobs[jobID]; ok {
		return job, nil
	}
	return generated.RecipeImportJob{}, nil
}

func (m *MockQueries) GetImportJobsByUser(ctx context.Context, userID pgtype.UUID) ([]generated.RecipeImportJob, error) {
	var jobs []generated.RecipeImportJob
	userUUID := uuid.UUID(userID.Bytes).String()
	for _, job := range m.importJobs {
		if uuid.UUID(job.UserID.Bytes).String() == userUUID {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (m *MockQueries) UpdateImportJobStatus(ctx context.Context, arg generated.UpdateImportJobStatusParams) error {
	jobID := arg.JobID
	if job, ok := m.importJobs[jobID]; ok {
		job.Status = arg.Status
		job.ProgressStep = arg.ProgressStep
		job.Error = arg.Error
		job.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		m.importJobs[jobID] = job
	}
	return nil
}

func (m *MockQueries) CreateRecipe(ctx context.Context, arg generated.CreateRecipeParams) (generated.Recipe, error) {
	recipe := generated.Recipe{
		ID:                  arg.ID,
		CreatedBy:           arg.CreatedBy,
		RecipeName:          arg.RecipeName,
		Description:         arg.Description,
		PrepTime:            arg.PrepTime,
		CookingTime:         arg.CookingTime,
		OriginalServingSize: arg.OriginalServingSize,
		DifficultyRating:    arg.DifficultyRating,
		Origin:              arg.Origin,
		Url:                 arg.Url,
		OwnerID:             arg.OwnerID,
		ThumbnailID:         arg.ThumbnailID,
		CreatedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	recipeID := uuid.UUID(arg.ID.Bytes).String()
	m.recipes[recipeID] = recipe
	return recipe, nil
}

func (m *MockQueries) GetRecipe(ctx context.Context, id pgtype.UUID) (generated.Recipe, error) {
	recipeID := uuid.UUID(id.Bytes).String()
	if recipe, ok := m.recipes[recipeID]; ok {
		return recipe, nil
	}
	return generated.Recipe{}, nil
}

func (m *MockQueries) UpdateRecipe(ctx context.Context, arg generated.UpdateRecipeParams) (generated.Recipe, error) {
	recipeID := uuid.UUID(arg.ID.Bytes).String()
	if recipe, ok := m.recipes[recipeID]; ok {
		recipe.RecipeName = arg.RecipeName
		recipe.Description = arg.Description
		recipe.PrepTime = arg.PrepTime
		recipe.CookingTime = arg.CookingTime
		recipe.OriginalServingSize = arg.OriginalServingSize
		recipe.DifficultyRating = arg.DifficultyRating
		recipe.Origin = arg.Origin
		recipe.Url = arg.Url
		recipe.OwnerID = arg.OwnerID
		recipe.ThumbnailID = arg.ThumbnailID
		recipe.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		m.recipes[recipeID] = recipe
		return recipe, nil
	}
	return generated.Recipe{}, nil
}

func (m *MockQueries) CreateIngredient(ctx context.Context, arg generated.CreateIngredientParams) (generated.RecipeIngredient, error) {
	ingredient := generated.RecipeIngredient{
		ID:               uuidToPgtype(uuid.New()),
		RecipeID:         arg.RecipeID,
		Quantity:         arg.Quantity,
		Unit:             arg.Unit,
		OriginalQuantity: arg.OriginalQuantity,
		OriginalUnit:     arg.OriginalUnit,
		Name:             arg.Name,
		CreatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	recipeID := uuid.UUID(arg.RecipeID.Bytes).String()
	m.ingredients[recipeID] = append(m.ingredients[recipeID], ingredient)
	return ingredient, nil
}

func (m *MockQueries) CreateInstruction(ctx context.Context, arg generated.CreateInstructionParams) (generated.RecipeInstruction, error) {
	instruction := generated.RecipeInstruction{
		ID:          uuidToPgtype(uuid.New()),
		RecipeID:    arg.RecipeID,
		StepNumber:  arg.StepNumber,
		Instruction: arg.Instruction,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	recipeID := uuid.UUID(arg.RecipeID.Bytes).String()
	m.instructions[recipeID] = append(m.instructions[recipeID], instruction)
	return instruction, nil
}

func (m *MockQueries) CreateNutrition(ctx context.Context, arg generated.CreateNutritionParams) (generated.RecipeNutrition, error) {
	nutrition := generated.RecipeNutrition{
		ID:        uuidToPgtype(uuid.New()),
		RecipeID:  arg.RecipeID,
		Protein:   arg.Protein,
		Carbs:     arg.Carbs,
		Fat:       arg.Fat,
		Fiber:     arg.Fiber,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	recipeID := uuid.UUID(arg.RecipeID.Bytes).String()
	m.nutrition[recipeID] = nutrition
	return nutrition, nil
}

func (m *MockQueries) GetIngredientsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeIngredient, error) {
	id := uuid.UUID(recipeID.Bytes).String()
	return m.ingredients[id], nil
}
func (m *MockQueries) DeleteOldImportJobs(ctx context.Context) error {
	return nil
}

func (m *MockQueries) DeleteStaleImportJobs(ctx context.Context) error {
	return nil
}

func (m *MockQueries) CreateRecipeImage(ctx context.Context, arg generated.CreateRecipeImageParams) (generated.RecipeImage, error) {
	return generated.RecipeImage{ID: pgtype.UUID{Valid: true, Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}}, nil
}


func (m *MockQueries) UpdateRecipeThumbnail(ctx context.Context, arg generated.UpdateRecipeThumbnailParams) error {
	return nil
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// float64Ptr returns a pointer to a float64
func float64Ptr(f float64) *float64 {
	return &f
}

// ============================================================================
// Service Mock Types (simplified for testing)
// ============================================================================

// ScrapedPost represents a generic scraped post
type ScrapedPost struct {
	ID       string
	Caption  string
	Platform string
}

// ProgressUpdate represents a progress update broadcast
type ProgressUpdate struct {
	JobID   string
	Status  string
	Message string
}

// MockProgressBroadcaster for testing
type MockProgressBroadcaster struct {
	BroadcastFunc func(userID string, update ProgressUpdate) error
	Broadcasts    []ProgressUpdate
}

func (m *MockProgressBroadcaster) Broadcast(userID string, update ProgressUpdate) error {
	m.Broadcasts = append(m.Broadcasts, update)
	if m.BroadcastFunc != nil {
		return m.BroadcastFunc(userID, update)
	}
	return nil
}

// MockAsynqClient for testing
type MockAsynqClient struct {
	EnqueueFunc func(task interface{}) (interface{}, error)
}

func (m *MockAsynqClient) Enqueue(task interface{}) (interface{}, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task)
	}
	return nil, nil
}

// ============================================================================
// Test Fixtures
// ============================================================================

type testFixtures struct {
	cfg             *config.Config
	mockDB          *MockQueries
	mockBroadcaster *MockProgressBroadcaster
	mockAsynq       *MockAsynqClient
}

func setupTestFixtures() *testFixtures {
	return &testFixtures{
		cfg: &config.Config{
			SupabaseURL:       "https://test.supabase.co",
			SupabaseJWTSecret: "test-secret",
		},
		mockDB:          NewMockQueries(),
		mockBroadcaster: &MockProgressBroadcaster{Broadcasts: make([]ProgressUpdate, 0)},
		mockAsynq:       &MockAsynqClient{},
	}
}

// Ensure MockQueries implements DBQueries interface
var _ DBQueries = (*MockQueries)(nil)

// ============================================================================
// Mock Tests
// ============================================================================

func TestMockQueries_CreateImportJob(t *testing.T) {
	m := NewMockQueries()
	userID := uuid.New()
	jobID := uuid.New()

	arg := generated.CreateImportJobParams{
		ID:     uuidToPgtype(jobID),
		UserID: uuidToPgtype(userID),
		Url:    "https://example.com/recipe",
		Status: "pending",
	}

	job, err := m.CreateImportJob(context.Background(), arg)
	require.NoError(t, err)
	assert.Equal(t, arg.Url, job.Url)
	assert.Equal(t, arg.Status, job.Status)
	assert.True(t, job.CreatedAt.Valid)
	assert.True(t, job.UpdatedAt.Valid)

	// Verify job was stored
	stored, err := m.GetImportJob(context.Background(), arg.ID)
	require.NoError(t, err)
	assert.Equal(t, job.Url, stored.Url)
}

func TestMockQueries_GetImportJob(t *testing.T) {
	m := NewMockQueries()
	userID := uuid.New()
	jobID := uuid.New()

	// Create a job first
	arg := generated.CreateImportJobParams{
		ID:     uuidToPgtype(jobID),
		UserID: uuidToPgtype(userID),
		Url:    "https://example.com/recipe",
		Status: "pending",
	}
	created, _ := m.CreateImportJob(context.Background(), arg)

	// Test getting existing job
	job, err := m.GetImportJob(context.Background(), arg.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Url, job.Url)
	assert.Equal(t, created.Status, job.Status)

	// Test getting non-existent job
	nonExistentID := uuidToPgtype(uuid.New())
	job, err = m.GetImportJob(context.Background(), nonExistentID)
	require.NoError(t, err)
	assert.Empty(t, job.Url)
}

func TestMockQueries_GetImportJobsByUser(t *testing.T) {
	m := NewMockQueries()
	userID := uuid.New()
	otherUserID := uuid.New()

	// Create jobs for user1
	for i := 0; i < 3; i++ {
		arg := generated.CreateImportJobParams{
			ID:     uuidToPgtype(uuid.New()),
			UserID: uuidToPgtype(userID),
			Url:    "https://example.com/recipe/" + string(rune('0'+i)),
			Status: "pending",
		}
		_, err := m.CreateImportJob(context.Background(), arg)
		require.NoError(t, err)
	}

	// Create job for user2
	arg := generated.CreateImportJobParams{
		ID:     uuidToPgtype(uuid.New()),
		UserID: uuidToPgtype(otherUserID),
		Url:    "https://example.com/other",
		Status: "pending",
	}
	_, err := m.CreateImportJob(context.Background(), arg)
	require.NoError(t, err)

	// Get jobs for user1
	jobs, err := m.GetImportJobsByUser(context.Background(), uuidToPgtype(userID))
	require.NoError(t, err)
	assert.Len(t, jobs, 3)

	// Get jobs for user2
	jobs, err = m.GetImportJobsByUser(context.Background(), uuidToPgtype(otherUserID))
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "https://example.com/other", jobs[0].Url)
}

func TestMockQueries_CreateRecipe(t *testing.T) {
	m := NewMockQueries()
	userID := uuid.New()
	recipeID := uuid.New()

	description := pgtype.Text{String: "A delicious test recipe", Valid: true}

	arg := generated.CreateRecipeParams{
		ID:                  uuidToPgtype(recipeID),
		CreatedBy:           uuidToPgtype(userID),
		RecipeName:          "Test Recipe",
		Description:         description,
		PrepTime:            pgtype.Int4{Int32: 30, Valid: true},
		CookingTime:         pgtype.Int4{Int32: 45, Valid: true},
		OriginalServingSize: pgtype.Int4{Int32: 4, Valid: true},
		DifficultyRating:    pgtype.Int2{Int16: 3, Valid: true},
		Origin:              generated.RecipeOriginInstagram,
		Url:                 "https://example.com/recipe",
	}

	recipe, err := m.CreateRecipe(context.Background(), arg)
	require.NoError(t, err)
	assert.Equal(t, "Test Recipe", recipe.RecipeName)
	assert.Equal(t, description.String, recipe.Description.String)
	assert.Equal(t, int32(30), recipe.PrepTime.Int32)
	assert.True(t, recipe.CreatedAt.Valid)
	assert.True(t, recipe.UpdatedAt.Valid)

	// Verify recipe was stored
	stored, err := m.GetRecipe(context.Background(), arg.ID)
	require.NoError(t, err)
	assert.Equal(t, recipe.RecipeName, stored.RecipeName)
}
