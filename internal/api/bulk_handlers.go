package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"
	"github.com/socialchef/remy/internal/worker"
)

const (
	MaxURLsPerBulkImport  = 50
	MaxConcurrentBulkJobs = 3
	BulkImportQueue       = "bulk_import"
)

type BulkImportRecipeRequest struct {
	URLs []string `json:"urls"`
}

type BulkImportRecipeResponse struct {
	BulkJobID string `json:"bulk_job_id"`
	TotalURLs int    `json:"total_urls"`
	Status    string `json:"status"`
}

type BulkImportResultItem struct {
	URL      string `json:"url"`
	Status   string `json:"status"`
	RecipeID string `json:"recipe_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

type BulkImportStatusResponse struct {
	BulkJobID string                 `json:"bulk_job_id"`
	Status    string                 `json:"status"`
	Progress  BulkImportProgress     `json:"progress"`
	Results   []BulkImportResultItem `json:"results,omitempty"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

type BulkImportProgress struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Success   int `json:"success"`
	Failed    int `json:"failed"`
}

type UserBulkImportSummary struct {
	BulkJobID    string `json:"bulk_job_id"`
	Status       string `json:"status"`
	TotalURLs    int    `json:"total_urls"`
	SuccessCount int    `json:"success_count"`
	FailedCount  int    `json:"failed_count"`
	CreatedAt    string `json:"created_at"`
}

type UserBulkImportsResponse struct {
	Jobs []UserBulkImportSummary `json:"jobs"`
}

func (s *Server) HandleBulkImportRecipe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req BulkImportRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(w, "At least one URL is required", http.StatusBadRequest)
		return
	}

	if len(req.URLs) > MaxURLsPerBulkImport {
		http.Error(w, fmt.Sprintf("Maximum %d URLs allowed per bulk import", MaxURLsPerBulkImport), http.StatusBadRequest)
		return
	}

	activeCount, err := s.db.GetUserActiveBulkImportCount(r.Context(), parseUUID(userID))
	if err != nil {
		slog.Error("Failed to check active bulk import count", "error", err, "user_id", userID)
		http.Error(w, "Failed to check rate limit", http.StatusInternalServerError)
		return
	}

	if activeCount >= MaxConcurrentBulkJobs {
		http.Error(w, fmt.Sprintf("Maximum %d concurrent bulk imports allowed. Please wait for existing imports to complete.", MaxConcurrentBulkJobs), http.StatusTooManyRequests)
		return
	}

	dedupedURLs := deduplicateURLs(req.URLs)
	if len(dedupedURLs) == 0 {
		http.Error(w, "No valid URLs provided", http.StatusBadRequest)
		return
	}

	bulkJobID := uuid.New().String()
	id := uuid.New().String()

	_, err = s.db.CreateBulkImportJob(r.Context(), generated.CreateBulkImportJobParams{
		ID:        parseUUID(id),
		JobID:     bulkJobID,
		UserID:    parseUUID(userID),
		TotalUrls: int32(len(dedupedURLs)),
		Status:    "QUEUED",
	})
	if err != nil {
		slog.Error("Failed to create bulk import job", "error", err, "user_id", userID, "bulk_job_id", bulkJobID)
		http.Error(w, "Failed to create bulk import job", http.StatusInternalServerError)
		return
	}

	task, err := worker.NewProcessBulkImportTask(worker.ProcessBulkImportPayload{
		BulkJobID: bulkJobID,
		URLs:      dedupedURLs,
		UserID:    userID,
	})
	if err != nil {
		slog.Error("Failed to create bulk import task", "error", err, "bulk_job_id", bulkJobID)
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	if _, err := s.asynqClient.Enqueue(task, worker.Queue(BulkImportQueue)); err != nil {
		slog.Error("Failed to enqueue bulk import task", "error", err, "bulk_job_id", bulkJobID)
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(BulkImportRecipeResponse{
		BulkJobID: bulkJobID,
		TotalURLs: len(dedupedURLs),
		Status:    "QUEUED",
	})
}

func (s *Server) HandleBulkImportStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bulkJobID := chi.URLParam(r, "bulkJobID")
	if bulkJobID == "" {
		http.Error(w, "bulkJobID is required", http.StatusBadRequest)
		return
	}

	job, err := s.db.GetBulkImportJobByJobID(r.Context(), bulkJobID)
	if err != nil {
		http.Error(w, "Bulk import job not found", http.StatusNotFound)
		return
	}

	if job.UserID.Bytes != parseUUID(userID).Bytes {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := BulkImportStatusResponse{
		BulkJobID: job.JobID,
		Status:    job.Status,
		Progress: BulkImportProgress{
			Total:     int(job.TotalUrls),
			Processed: int(job.ProcessedCount.Int32),
			Success:   int(job.SuccessCount.Int32),
			Failed:    int(job.FailedCount.Int32),
		},
		CreatedAt: job.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: job.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}

	if job.Status == "COMPLETED" || job.Status == "FAILED" || job.ProcessedCount.Int32 > 0 {
		importJobs, err := s.db.GetImportJobsByBulkJobID(r.Context(), pgtype.Text{String: bulkJobID, Valid: true})
		if err == nil && len(importJobs) > 0 {
			response.Results = make([]BulkImportResultItem, len(importJobs))
			for i, ij := range importJobs {
				item := BulkImportResultItem{
					URL:    ij.Url,
					Status: ij.Status,
				}
				if ij.Result != nil {
					var result map[string]interface{}
					if err := json.Unmarshal(ij.Result, &result); err == nil {
						if recipeID, ok := result["recipe_id"].(string); ok {
							item.RecipeID = recipeID
						}
					}
				}
				if ij.Error != nil {
					item.Error = string(ij.Error)
				}
				response.Results[i] = item
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleListUserBulkImports(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobs, err := s.db.GetBulkImportJobsByUser(r.Context(), parseUUID(userID))
	if err != nil {
		slog.Error("Failed to fetch bulk import jobs", "error", err, "user_id", userID)
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	response := UserBulkImportsResponse{
		Jobs: make([]UserBulkImportSummary, len(jobs)),
	}

	for i, job := range jobs {
		response.Jobs[i] = UserBulkImportSummary{
			BulkJobID:    job.JobID,
			Status:       job.Status,
			TotalURLs:    int(job.TotalUrls),
			SuccessCount: int(job.SuccessCount.Int32),
			FailedCount:  int(job.FailedCount.Int32),
			CreatedAt:    job.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleCancelBulkImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bulkJobID := chi.URLParam(r, "bulkJobID")
	if bulkJobID == "" {
		http.Error(w, "bulkJobID is required", http.StatusBadRequest)
		return
	}

	job, err := s.db.GetBulkImportJobByJobID(r.Context(), bulkJobID)
	if err != nil {
		http.Error(w, "Bulk import job not found", http.StatusNotFound)
		return
	}

	if job.UserID.Bytes != parseUUID(userID).Bytes {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if job.Status != "QUEUED" && job.Status != "EXECUTING" {
		http.Error(w, fmt.Sprintf("Cannot cancel job with status: %s", job.Status), http.StatusBadRequest)
		return
	}

	if err := s.db.CancelBulkImportJob(r.Context(), bulkJobID); err != nil {
		slog.Error("Failed to cancel bulk import job", "error", err, "bulk_job_id", bulkJobID)
		http.Error(w, "Failed to cancel job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"bulk_job_id": bulkJobID,
		"status":      "CANCELED",
	})
}

func deduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(urls))

	for _, url := range urls {
		trimmed := strings.TrimSpace(url)
		if trimmed == "" {
			continue
		}
		if !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}

	return result
}
