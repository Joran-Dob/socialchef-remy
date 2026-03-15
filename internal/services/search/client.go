package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db/generated"
)

type DBQueries interface {
	SearchRecipesByEmbedding(ctx context.Context, arg generated.SearchRecipesByEmbeddingParams) ([]generated.SearchRecipesByEmbeddingRow, error)
	SearchRecipesByName(ctx context.Context, arg generated.SearchRecipesByNameParams) ([]generated.SearchRecipesByNameRow, error)
	SearchRecipesHybrid(ctx context.Context, arg generated.SearchRecipesHybridParams) ([]generated.SearchRecipesHybridRow, error)
	SearchRecipesByIngredient(ctx context.Context, arg generated.SearchRecipesByIngredientParams) ([]generated.SearchRecipesByIngredientRow, error)
}

type OpenAIClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	Complete(ctx context.Context, prompt string) (string, error)
}

type SearchResult struct {
	ID                string   `json:"id"`
	RecipeName        string   `json:"recipe_name"`
	Description       string   `json:"description,omitempty"`
	ThumbnailID       string   `json:"thumbnail_id,omitempty"`
	ThumbnailURL      string   `json:"thumbnail_url,omitempty"`
	OwnerID           string   `json:"owner_id,omitempty"`
	OwnerUsername     string   `json:"owner_username,omitempty"`
	VectorSimilarity  float64  `json:"vector_similarity,omitempty"`
	TextSimilarity    float64  `json:"text_similarity,omitempty"`
	HybridScore       float64  `json:"hybrid_score,omitempty"`
	CuisineCategories []string `json:"cuisine_categories,omitempty"`
	MealTypes         []string `json:"meal_types,omitempty"`
}

type Client struct {
	db         DBQueries
	openai     OpenAIClient
	cfg        *config.Config
	classifier *QueryClassifier
	reranker   *CrossEncoderReranker
	expander   *QueryExpander
}

func NewClient(db DBQueries, openai OpenAIClient, cfg *config.Config) *Client {
	return &Client{
		db:         db,
		openai:     openai,
		cfg:        cfg,
		classifier: NewQueryClassifier(),
		reranker:   NewCrossEncoderReranker(openai),
		expander:   NewQueryExpander(openai),
	}
}

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

func (c *Client) SearchSemantic(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	embedding, err := c.openai.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := c.db.SearchRecipesByEmbedding(ctx, generated.SearchRecipesByEmbeddingParams{
		Limit:   limit,
		Column2: pgvector.NewVector(embedding),
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
			VectorSimilarity:  r.Similarity,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}

	return searchResults, nil
}

func (c *Client) SearchByName(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	results, err := c.db.SearchRecipesByName(ctx, generated.SearchRecipesByNameParams{
		Similarity: query,
		Limit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes by name: %w", err)
	}

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

func (c *Client) SearchByIngredient(ctx context.Context, ingredient string, limit int32) ([]SearchResult, error) {
	results, err := c.db.SearchRecipesByIngredient(ctx, generated.SearchRecipesByIngredientParams{
		IngredientNames: []string{ingredient},
		Limit:           limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes by ingredient: %w", err)
	}

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

func (c *Client) SearchTwoPhase(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	results, err := c.db.SearchRecipesHybrid(ctx, generated.SearchRecipesHybridParams{
		Limit:          limit,
		Column2:        pgvector.NewVector(make([]float32, 1536)),
		PlaintoTsquery: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}

	goodMatches := 0
	for _, r := range results {
		textScore := 0.3 * r.TextSimilarity
		if textScore >= 0.24 {
			goodMatches++
		}
	}

	if goodMatches >= 3 {
		return c.convertHybridRows(results), nil
	}

	return c.SearchHybrid(ctx, query, limit)
}

func (c *Client) SearchHybrid(ctx context.Context, query string, limit int32) ([]SearchResult, error) {
	// Expand query
	expandedQuery, err := c.expander.ExpandQuery(ctx, query)
	if err != nil {
		expandedQuery = query // Fallback to original
	}

	embedding, err := c.openai.GenerateEmbedding(ctx, expandedQuery)
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
			ThumbnailURL:      c.buildThumbnailURL(r.ThumbnailStoragePath.String),
			OwnerID:           pgUUIDToString(r.OwnerID),
			OwnerUsername:     r.OwnerUsername.String,
			VectorSimilarity:  r.VectorSimilarity,
			TextSimilarity:    r.TextSimilarity,
			HybridScore:       r.HybridScore,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}

	searchResults = c.diversifyResults(searchResults, 3)

	rerankedResults, err := c.reranker.Rerank(ctx, query, searchResults, int(limit))
	if err != nil {
		return searchResults, nil
	}

	if int(limit) < len(rerankedResults) {
		rerankedResults = rerankedResults[:limit]
	}

	return rerankedResults, nil
}

func (c *Client) diversifyResults(results []SearchResult, maxPerCategory int) []SearchResult {
	if maxPerCategory <= 0 {
		maxPerCategory = 3
	}

	categoryCount := make(map[string]int)
	diversified := make([]SearchResult, 0, len(results))

	for _, r := range results {
		primaryCuisine := "Unknown"
		if len(r.CuisineCategories) > 0 {
			primaryCuisine = r.CuisineCategories[0]
		}

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
			ThumbnailURL:      c.buildThumbnailURL(r.ThumbnailStoragePath.String),
			OwnerID:           pgUUIDToString(r.OwnerID),
			OwnerUsername:     r.OwnerUsername.String,
			VectorSimilarity:  r.VectorSimilarity,
			TextSimilarity:    r.TextSimilarity,
			HybridScore:       r.HybridScore,
			CuisineCategories: interfaceToStringSlice(r.CuisineCategories),
			MealTypes:         interfaceToStringSlice(r.MealTypes),
		}
	}
	return results
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
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

func (c *Client) buildThumbnailURL(storagePath string) string {
	if storagePath == "" || c.cfg == nil {
		return ""
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.cfg.SupabaseURL, c.cfg.RecipeStorageBucket, storagePath)
}
