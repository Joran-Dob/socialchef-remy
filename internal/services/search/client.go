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
	SearchRecipesByName(ctx context.Context, arg generated.SearchRecipesByNameParams) ([]generated.SearchRecipesByNameRow, error)
	SearchRecipesHybrid(ctx context.Context, arg generated.SearchRecipesHybridParams) ([]generated.SearchRecipesHybridRow, error)
}

// OpenAIClient defines the interface for generating embeddings
type OpenAIClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// SearchResult represents a recipe search result
type SearchResult struct {
	ID                string   `json:"id"`
	RecipeName        string   `json:"recipe_name"`
	Description       string   `json:"description,omitempty"`
	Similarity        float64  `json:"similarity,omitempty"`
	CuisineCategories []string `json:"cuisine_categories,omitempty"`
	MealTypes         []string `json:"meal_types,omitempty"`
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

// Search determines the search intent and routes to the appropriate method
func (c *Client) Search(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	intent := c.classifier.Classify(query)

	switch intent {
	case IntentByIngredient:
		return c.SearchByIngredient(ctx, query, limit)
	case IntentSimilarTo:
		return c.SearchSemantic(ctx, query, limit)
	case IntentByName:
		return c.SearchByName(ctx, query, limit)
	default:
		return c.SearchHybrid(ctx, query, limit)
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
			ID:                pgUUIDToString(r.ID),
			RecipeName:        r.RecipeName,
			Description:       r.Description.String,
			Similarity:        r.Similarity,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}

	return searchResults, nil
}

// SearchByName performs text-based search on recipe names
func (c *Client) SearchByName(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	results, err := c.db.SearchRecipesByName(ctx, generated.SearchRecipesByNameParams{
		Similarity: query,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes by name: %w", err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:                pgUUIDToString(r.ID),
			RecipeName:        r.RecipeName,
			Description:       r.Description.String,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}

	return searchResults, nil
}

// SearchTwoPhase first tries exact text matches, then falls back to vector
// search if not enough high-confidence results are found
func (c *Client) SearchTwoPhase(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	// Phase 1: High-weight text search (exact matches with zero vector contribution)
	results, err := c.db.SearchRecipesHybrid(ctx, generated.SearchRecipesHybridParams{
		Limit:          limit,
		Column2:        pgvector.NewVector(make([]float32, 1536)), // Zero vector (no vector contribution)
		PlaintoTsquery: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}

	// Check if we have good exact matches (similarity > 0.8)
	goodMatches := 0
	for _, r := range results {
		// Calculate text-only score
		textScore := 0.3 * interfaceToFloat64(r.TextRank)
		if textScore >= 0.24 { // 0.8 * 0.3
			goodMatches++
		}
	}

	// If we have good exact matches, return them
	if goodMatches >= 3 {
		return c.convertHybridRows(results), nil
	}

	// Phase 2: Full hybrid search with vector contribution
	return c.SearchHybrid(ctx, query, limit)
}

func (c *Client) SearchHybrid(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	embedding, err := c.openai.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := c.db.SearchRecipesHybrid(ctx, generated.SearchRecipesHybridParams{
		Limit:          limit,
		Column2:        pgvector.NewVector(embedding),
		PlaintoTsquery: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:                pgUUIDToString(r.ID),
			RecipeName:        r.RecipeName,
			Description:       r.Description.String,
			Similarity:        r.HybridScore,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}

	// Apply diversity - limit to 3 per cuisine
	searchResults = c.diversifyResults(searchResults, 3)

	// Trim to requested limit
	if int(limit) < len(searchResults) {
		searchResults = searchResults[:limit]
	}

	return searchResults, nil
}

// diversifyResults ensures diversity in search results by cuisine
// It limits the number of results per primary cuisine to prevent all results
// being from the same cuisine type
func (c *Client) diversifyResults(results []SearchResult, maxPerCategory int) []SearchResult {
	if maxPerCategory <= 0 {
		maxPerCategory = 3 // Default: max 3 per cuisine
	}

	categoryCount := make(map[string]int)
	diversified := make([]SearchResult, 0, len(results))

	for _, r := range results {
		// Get primary cuisine (first one)
		primaryCuisine := "Unknown"
		if len(r.CuisineCategories) > 0 {
			primaryCuisine = r.CuisineCategories[0]
		}

		// Check if we've hit the limit for this category
		if categoryCount[primaryCuisine] < maxPerCategory {
			diversified = append(diversified, r)
			categoryCount[primaryCuisine]++
		}
	}

	return diversified
}

func (c *Client) convertHybridRows(rows []generated.SearchRecipesHybridRow) []SearchResult {
	results := make([]SearchResult, len(rows))
	for i, r := range rows {
		results[i] = SearchResult{
			ID:                pgUUIDToString(r.ID),
			RecipeName:        r.RecipeName,
			Description:       r.Description.String,
			Similarity:        r.HybridScore,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}
	return results
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

func interfaceToStringSlice(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	if arr, ok := v.([]string); ok {
		return arr
	}
	if arr, ok := v.([]interface{}); ok {
		result := make([]string, len(arr))
		for i, item := range arr {
			if s, ok := item.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	return []string{}
}

func interfaceToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	if f, ok := v.(float32); ok {
		return float64(f)
	}
	return 0
}
