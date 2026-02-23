package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Task type constants
const (
	TypeProcessRecipe     = "process:recipe"
	TypeGenerateEmbedding = "generate:embedding"
	TypeCleanupJobs       = "cleanup:jobs"
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
