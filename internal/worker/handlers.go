package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/groq"
	"github.com/socialchef/remy/internal/services/openai"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/services/storage"
)

type RecipeProcessor struct {
	db          *generated.Queries
	instagram   *scraper.InstagramScraper
	tiktok      *scraper.TikTokScraper
	openai      *openai.Client
	groq        *groq.Client
	storage     *storage.Client
	broadcaster *ProgressBroadcaster
}

func NewRecipeProcessor(
	db *generated.Queries,
	instagram *scraper.InstagramScraper,
	tiktok *scraper.TikTokScraper,
	openaiClient *openai.Client,
	groqClient *groq.Client,
	storageClient *storage.Client,
	broadcaster *ProgressBroadcaster,
) *RecipeProcessor {
	return &RecipeProcessor{
		db:          db,
		instagram:   instagram,
		tiktok:      tiktok,
		openai:      openaiClient,
		groq:        groqClient,
		storage:     storageClient,
		broadcaster: broadcaster,
	}
}

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{Valid: false}
	}
	return u
}

func (p *RecipeProcessor) HandleProcessRecipe(ctx context.Context, t *asynq.Task) error {
	var payload ProcessRecipePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	jobID := payload.JobID
	userID := payload.UserID
	url := payload.URL

	slog.Info("Processing recipe", "job_id", jobID, "url", url)

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Fetching post content...")

	var caption, platform string

	if scraper.IsInstagramURL(url) {
		platform = "instagram"
		post, err := p.instagram.Scrape(ctx, url)
		if err != nil {
			p.markFailed(ctx, jobID, userID, fmt.Sprintf("Instagram scrape failed: %v", err))
			return err
		}
		caption = post.Caption
	} else if scraper.IsTikTokURL(url) {
		platform = "tiktok"
		post, err := p.tiktok.Scrape(ctx, url)
		if err != nil {
			p.markFailed(ctx, jobID, userID, fmt.Sprintf("TikTok scrape failed: %v", err))
			return err
		}
		caption = post.Caption
	} else {
		p.markFailed(ctx, jobID, userID, "Invalid URL: must be Instagram or TikTok")
		return fmt.Errorf("invalid URL")
	}

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Generating recipe with AI...")

	recipe, err := p.groq.GenerateRecipe(ctx, caption, "", platform)
	if err != nil {
		p.markFailed(ctx, jobID, userID, fmt.Sprintf("Recipe generation failed: %v", err))
		return err
	}

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Saving recipe to database...")

	recipeUUID := parseUUID(uuid.New().String())
	userUUID := parseUUID(userID)

	var difficultyRating pgtype.Int2
	if recipe.DifficultyRating != nil {
		difficultyRating = pgtype.Int2{Int16: int16(*recipe.DifficultyRating), Valid: true}
	}

	var origin generated.RecipeOrigin
	if platform == "instagram" {
		origin = generated.RecipeOriginInstagram
	} else {
		origin = generated.RecipeOriginTiktok
	}

	savedRecipe, err := p.db.CreateRecipe(ctx, generated.CreateRecipeParams{
		ID:                  recipeUUID,
		CreatedBy:           userUUID,
		RecipeName:          recipe.RecipeName,
		Description:         pgtype.Text{String: recipe.Description, Valid: recipe.Description != ""},
		PrepTime:            pgtype.Int4{Int32: int32(ptrToInt(recipe.PrepTime)), Valid: recipe.PrepTime != nil},
		CookingTime:         pgtype.Int4{Int32: int32(ptrToInt(recipe.CookingTime)), Valid: recipe.CookingTime != nil},
		OriginalServingSize: pgtype.Int4{Int32: int32(ptrToInt(recipe.OriginalServings)), Valid: recipe.OriginalServings != nil},
		DifficultyRating:    difficultyRating,
		Origin:              origin,
		Url:                 url,
		OwnerID:             pgtype.UUID{},
		ThumbnailID:         pgtype.UUID{},
	})
	if err != nil {
		p.markFailed(ctx, jobID, userID, fmt.Sprintf("Failed to save recipe: %v", err))
		return err
	}

	for _, ing := range recipe.Ingredients {
		_, err := p.db.CreateIngredient(ctx, generated.CreateIngredientParams{
			RecipeID:         savedRecipe.ID,
			Quantity:         pgtype.Text{String: ing.OriginalQuantity, Valid: ing.OriginalQuantity != ""},
			Unit:             pgtype.Text{String: ing.Unit, Valid: ing.Unit != ""},
			OriginalQuantity: pgtype.Text{String: ing.OriginalQuantity, Valid: ing.OriginalQuantity != ""},
			OriginalUnit:     pgtype.Text{String: ing.Unit, Valid: ing.Unit != ""},
			Name:             ing.Name,
		})
		if err != nil {
			slog.Error("Failed to save ingredient", "error", err)
		}
	}

	for i, inst := range recipe.Instructions {
		_, err := p.db.CreateInstruction(ctx, generated.CreateInstructionParams{
			RecipeID:    savedRecipe.ID,
			StepNumber:  int32(i + 1),
			Instruction: inst.Instruction,
		})
		if err != nil {
			slog.Error("Failed to save instruction", "error", err)
		}
	}

	if recipe.Nutrition.Protein > 0 || recipe.Nutrition.Carbs > 0 {
		_, err := p.db.CreateNutrition(ctx, generated.CreateNutritionParams{
			RecipeID: savedRecipe.ID,
			Protein:  pgtype.Numeric{Int: big.NewInt(int64(recipe.Nutrition.Protein * 100)), Exp: -2, Valid: true},
			Carbs:    pgtype.Numeric{Int: big.NewInt(int64(recipe.Nutrition.Carbs * 100)), Exp: -2, Valid: true},
			Fat:      pgtype.Numeric{Int: big.NewInt(int64(recipe.Nutrition.Fat * 100)), Exp: -2, Valid: true},
			Fiber:    pgtype.Numeric{Int: big.NewInt(int64(recipe.Nutrition.Fiber * 100)), Exp: -2, Valid: true},
		})
		if err != nil {
			slog.Error("Failed to save nutrition", "error", err)
		}
	}

	p.updateProgress(ctx, jobID, userID, "COMPLETED", "Recipe saved successfully!")

	return nil
}

func (p *RecipeProcessor) HandleGenerateEmbedding(ctx context.Context, t *asynq.Task) error {
	var payload GenerateEmbeddingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	recipeUUID := parseUUID(payload.RecipeID)
	recipe, err := p.db.GetRecipe(ctx, recipeUUID)
	if err != nil {
		return fmt.Errorf("recipe not found: %w", err)
	}

	text := recipe.RecipeName + " " + recipe.Description.String
	ingredients, _ := p.db.GetIngredientsByRecipe(ctx, recipe.ID)
	for _, ing := range ingredients {
		text += " " + ing.Name
	}

	_, err = p.openai.GenerateEmbedding(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	slog.Info("Embedding generated", "recipe_id", payload.RecipeID)
	return nil
}

func (p *RecipeProcessor) HandleCleanupJobs(ctx context.Context, t *asynq.Task) error {
	slog.Info("Running cleanup job")
	return nil
}

func (p *RecipeProcessor) updateProgress(ctx context.Context, jobID, userID, status, message string) {
	slog.Info("Progress update", "job_id", jobID, "status", status, "message", message)

	err := p.db.UpdateImportJobStatus(ctx, generated.UpdateImportJobStatusParams{
		JobID:        jobID,
		Status:       status,
		ProgressStep: pgtype.Text{String: message, Valid: true},
	})
	if err != nil {
		slog.Error("Failed to update job status in database", "error", err, "job_id", jobID)
	}

	if p.broadcaster != nil {
		p.broadcaster.Broadcast(userID, ProgressUpdate{
			JobID:   jobID,
			Status:  status,
			Message: message,
		})
	}
}

func (p *RecipeProcessor) markFailed(ctx context.Context, jobID, userID, errorMsg string) {
	slog.Error("Job failed", "job_id", jobID, "error", errorMsg)

	p.db.UpdateImportJobStatus(ctx, generated.UpdateImportJobStatusParams{
		JobID:        jobID,
		Status:       "FAILED",
		ProgressStep: pgtype.Text{String: "Failed", Valid: true},
		Error:        []byte(errorMsg),
	})

	if p.broadcaster != nil {
		p.broadcaster.Broadcast(userID, ProgressUpdate{
			JobID:   jobID,
			Status:  "failed",
			Message: errorMsg,
		})
	}
}

func ptrToInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
