package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/socialchef/remy/internal/db/generated"
)

// DBQueries defines the database operations needed for search
type DBQueries interface {
	SearchRecipesByEmbedding(ctx context.Context, arg generated.SearchRecipesByEmbeddingParams) ([]generated.SearchRecipesByEmbeddingRow, error)
	SearchRecipesByName(ctx context.Context, arg generated.SearchRecipesByNameParams) ([]generated.Recipe, error)
}

// OpenAIClient defines the interface for generating embeddings
type OpenAIClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// SearchResult represents a recipe search result
type SearchResult struct {
	ID          string  `json:"id"`
	RecipeName  string  `json:"recipe_name"`
	Description string  `json:"description,omitempty"`
	Similarity  float64 `json:"similarity,omitempty"`
}

// Client provides search functionality
type Client struct {
	db     DBQueries
	openai OpenAIClient
}

// NewClient creates a new search client
func NewClient(db DBQueries, openai OpenAIClient) *Client {
	return &Client{
		db:     db,
		openai: openai,
	}
}

// SearchSemantic performs semantic (vector) search using embeddings
func (c *Client) SearchSemantic(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	// Generate embedding for the query
	embedding, err := c.openai.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search using the embedding
	results, err := c.db.SearchRecipesByEmbedding(ctx, generated.SearchRecipesByEmbeddingParams{
		Limit:   limit,
		Column2: pgvector.NewVector(embedding),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:          pgUUIDToString(r.ID),
			RecipeName:  r.RecipeName,
			Description: r.Description.String,
			Similarity:  r.Similarity,
		}
	}

	return searchResults, nil
}

// SearchByName performs text-based search on recipe names
func (c *Client) SearchByName(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	results, err := c.db.SearchRecipesByName(ctx, generated.SearchRecipesByNameParams{
		Column1: pgtype.Text{String: query, Valid: true},
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes by name: %w", err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:          pgUUIDToString(r.ID),
			RecipeName:  r.RecipeName,
			Description: r.Description.String,
		}
	}

	return searchResults, nil
}

// pgUUIDToString converts a pgtype.UUID to a string
func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	// Convert [16]byte to UUID string format
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
