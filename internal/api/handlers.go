package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"
	"github.com/socialchef/remy/internal/worker"
)

type Server struct {
	cfg         *config.Config
	db          *generated.Queries
	asynqClient *asynq.Client
}

func NewServer(cfg *config.Config, db *generated.Queries, asynqClient *asynq.Client) *Server {
	return &Server{
		cfg:         cfg,
		db:          db,
		asynqClient: asynqClient,
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

	jobID := uuid.New().String()

	_, err := s.db.CreateImportJob(r.Context(), generated.CreateImportJobParams{
		ID:     parseUUID(jobID),
		UserID: parseUUID(userID),
		Url:    req.URL,
		Status: "pending",
	})
	if err != nil {
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

	job, err := s.db.GetImportJob(r.Context(), parseUUID(jobID))
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
		Error:        job.Error.String,
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
			Error:        job.Error.String,
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
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}
