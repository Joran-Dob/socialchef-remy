package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"
	"github.com/socialchef/remy/internal/services/search"
	"github.com/socialchef/remy/internal/worker"
)

type Server struct {
	cfg         *config.Config
	db          *generated.Queries
	asynqClient *asynq.Client
	search      *search.Client
}

func NewServer(cfg *config.Config, db *generated.Queries, asynqClient *asynq.Client, searchClient *search.Client) *Server {
	return &Server{
		cfg:         cfg,
		db:          db,
		asynqClient: asynqClient,
		search:      searchClient,
	}
}

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

type ImportRecipeRequest struct {
	URL string `json:"url"`
}

type ImportRecipeResponse struct {
	JobID string `json:"job_id"`
	URL   string `json:"url"`
}

func (s *Server) HandleImportRecipe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ImportRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Detect origin from URL
	origin := "instagram"
	if strings.Contains(req.URL, "tiktok") {
		origin = "tiktok"
	}

	// Generate separate IDs: database ID and job/task ID
	id := uuid.New().String()
	jobID := uuid.New().String()

	_, err := s.db.CreateImportJob(r.Context(), generated.CreateImportJobParams{
		ID:     parseUUID(id),
		JobID:  jobID,
		UserID: parseUUID(userID),
		Url:    req.URL,
		Origin: origin,
		Status: "QUEUED",
	})
	if err != nil {
		slog.Error("Failed to create import job", "error", err, "user_id", userID, "job_id", jobID)
		http.Error(w, "Failed to create import job", http.StatusInternalServerError)
		return
	}

	task, err := worker.NewProcessRecipeTask(worker.ProcessRecipePayload{
		JobID:  jobID,
		URL:    req.URL,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	if _, err := s.asynqClient.Enqueue(task); err != nil {
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(ImportRecipeResponse{
		JobID: jobID,
		URL:   req.URL,
	})
}

type JobStatusResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ProgressStep string `json:"progress_step,omitempty"`
	Error        string `json:"error,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (s *Server) HandleJobStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	job, err := s.db.GetImportJobByJobID(r.Context(), jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if job.UserID.Bytes != parseUUID(userID).Bytes {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JobStatusResponse{
		ID:           uuid.UUID(job.ID.Bytes).String(),
		Status:       job.Status,
		ProgressStep: job.ProgressStep.String,
		Error:        string(job.Error),
		CreatedAt:    job.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    job.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type UserImportStatusResponse struct {
	Jobs []JobStatusResponse `json:"jobs"`
}

func (s *Server) HandleUserImportStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobs, err := s.db.GetImportJobsByUser(r.Context(), parseUUID(userID))
	if err != nil {
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	response := UserImportStatusResponse{
		Jobs: make([]JobStatusResponse, len(jobs)),
	}

	for i, job := range jobs {
		response.Jobs[i] = JobStatusResponse{
			ID:           uuid.UUID(job.ID.Bytes).String(),
			Status:       job.Status,
			ProgressStep: job.ProgressStep.String,
			Error:        string(job.Error),
			CreatedAt:    job.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    job.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type GenerateEmbeddingRequest struct {
	RecipeID string `json:"recipe_id"`
}

func (s *Server) HandleGenerateEmbedding(w http.ResponseWriter, r *http.Request) {
	var req GenerateEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RecipeID == "" {
		http.Error(w, "recipe_id is required", http.StatusBadRequest)
		return
	}

	task, err := worker.NewGenerateEmbeddingTask(worker.GenerateEmbeddingPayload{
		RecipeID: req.RecipeID,
	})
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	if _, err := s.asynqClient.Enqueue(task); err != nil {
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

type InstructionIngredientsCountResponse struct {
	Count int `json:"count"`
}

func (s *Server) HandleGetInstructionIngredientsCount(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	recipeID := r.URL.Query().Get("recipe_id")
	if recipeID == "" {
		http.Error(w, "recipe_id is required", http.StatusBadRequest)
		return
	}

	ingredients, err := s.db.GetInstructionIngredientsByRecipe(r.Context(), parseUUID(recipeID))
	if err != nil {
		slog.Error("Failed to get instruction ingredients", "error", err, "recipe_id", recipeID)
		http.Error(w, "Failed to get instruction ingredients", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(InstructionIngredientsCountResponse{
		Count: len(ingredients),
	})
}

type Timer struct {
	DurationSeconds int    `json:"duration_seconds"`
	DurationText    string `json:"duration_text"`
	Label           string `json:"label"`
	Type            string `json:"type"`
	Category        string `json:"category"`
}

type StepIngredientDetail struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StepQuantity  string `json:"step_quantity"`  // from instruction_ingredients
	TotalQuantity string `json:"total_quantity"` // from recipe_ingredients
	Unit          string `json:"unit"`           // from recipe_ingredients
}

type StepDetail struct {
	StepNumber      int32                  `json:"step_number"`
	InstructionRich string                 `json:"instruction_rich"`
	Ingredients     []StepIngredientDetail `json:"ingredients"`
	Timers          []Timer                `json:"timers"`
}

type RecipeStepsResponse struct {
	RecipeID   string       `json:"recipe_id"`
	TotalSteps int          `json:"total_steps"`
	Steps      []StepDetail `json:"steps"`
}

func (s *Server) HandleGetRecipeSteps(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	recipeID := chi.URLParam(r, "recipeID")
	if recipeID == "" {
		http.Error(w, "recipe_id is required", http.StatusBadRequest)
		return
	}

	recipe, err := s.db.GetRecipe(r.Context(), parseUUID(recipeID))
	if err != nil {
		slog.Error("Failed to get recipe", "error", err, "recipe_id", recipeID)
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}

	instructions, err := s.db.GetInstructionsByRecipe(r.Context(), parseUUID(recipeID))
	if err != nil {
		slog.Error("Failed to get instructions", "error", err, "recipe_id", recipeID)
		http.Error(w, "Failed to get instructions", http.StatusInternalServerError)
		return
	}

	recipeIngredients, err := s.db.GetIngredientsByRecipe(r.Context(), parseUUID(recipeID))
	if err != nil {
		slog.Error("Failed to get recipe ingredients", "error", err, "recipe_id", recipeID)
		http.Error(w, "Failed to get recipe ingredients", http.StatusInternalServerError)
		return
	}

	ingredientMap := make(map[string]generated.RecipeIngredient)
	for _, ing := range recipeIngredients {
		ingID := uuid.UUID(ing.ID.Bytes).String()
		ingredientMap[ingID] = ing
	}

	steps := make([]StepDetail, 0, len(instructions))
	for _, inst := range instructions {
		instIngredients, err := s.db.GetInstructionIngredientsByInstruction(r.Context(), inst.ID)
		if err != nil {
			slog.Error("Failed to get instruction ingredients", "error", err, "instruction_id", inst.ID)
			http.Error(w, "Failed to get instruction ingredients", http.StatusInternalServerError)
			return
		}

		ingredients := make([]StepIngredientDetail, 0, len(instIngredients))
		for _, ii := range instIngredients {
			ingredientID := uuid.UUID(ii.IngredientID.Bytes).String()
			recipeIng, ok := ingredientMap[ingredientID]
			if !ok {
				slog.Warn("Ingredient not found in recipe ingredients", "ingredient_id", ingredientID)
				continue
			}

			ingredients = append(ingredients, StepIngredientDetail{
				ID:            ingredientID,
				Name:          recipeIng.Name,
				StepQuantity:  ii.StepQuantity.String,
				TotalQuantity: recipeIng.TotalQuantity.String,
				Unit:          recipeIng.Unit.String,
			})
		}

		var timers []Timer
		if len(inst.TimerData) > 0 {
			if err := json.Unmarshal(inst.TimerData, &timers); err != nil {
				slog.Error("Failed to parse timer data", "error", err, "instruction_id", inst.ID)
			}
		}

		steps = append(steps, StepDetail{
			StepNumber:      inst.StepNumber,
			InstructionRich: inst.InstructionRich.String,
			Ingredients:     ingredients,
			Timers:          timers,
		})
	}

	response := RecipeStepsResponse{
		RecipeID:   uuid.UUID(recipe.ID.Bytes).String(),
		TotalSteps: len(steps),
		Steps:      steps,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
