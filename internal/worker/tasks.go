package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Priority constants for task queues
const (
	PriorityHigh   = 10
	PriorityNormal = 5
	PriorityLow    = 1
)

// Task type constants
const (
	TypeProcessRecipe     = "process:recipe"
	TypeGenerateEmbedding = "generate:embedding"
	TypeCleanupJobs       = "cleanup:jobs"
	TypeInstagramRetry    = "instagram:retry"
)

// ProcessRecipePayload is the payload for recipe processing tasks
type ProcessRecipePayload struct {
	JobID  string `json:"job_id"`
	URL    string `json:"url"`
	UserID string `json:"user_id"`
}

// GenerateEmbeddingPayload is the payload for embedding tasks
type GenerateEmbeddingPayload struct {
	RecipeID string `json:"recipe_id"`
}

// InstagramRetryPayload is the payload for Instagram retry tasks
type InstagramRetryPayload struct {
	JobID string `json:"job_id"`
	URL   string `json:"url"`
}

// NewProcessRecipeTask creates a new process recipe task
func NewProcessRecipeTask(payload ProcessRecipePayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeProcessRecipe, data), nil
}

// NewGenerateEmbeddingTask creates a new embedding task
func NewGenerateEmbeddingTask(payload GenerateEmbeddingPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGenerateEmbedding, data), nil
}

// NewCleanupJobsTask creates a new cleanup task
func NewCleanupJobsTask() *asynq.Task {
	return asynq.NewTask(TypeCleanupJobs, nil)
}

// NewInstagramRetryTask creates a new Instagram retry task with high priority
func NewInstagramRetryTask(payload InstagramRetryPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeInstagramRetry, data), nil
}
