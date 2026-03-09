package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/errors"
	"github.com/socialchef/remy/internal/services/ai"
	"github.com/socialchef/remy/internal/services/groq"
	"github.com/socialchef/remy/internal/services/recipe"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/services/storage"
	sentrylib "github.com/socialchef/remy/internal/sentry"
	"github.com/socialchef/remy/internal/utils"
	"github.com/socialchef/remy/internal/validation"
)

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
	UpdateInstructionRich(ctx context.Context, arg generated.UpdateInstructionRichParams) error
	CreateNutrition(ctx context.Context, arg generated.CreateNutritionParams) (generated.RecipeNutrition, error)
	GetIngredientsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeIngredient, error)
	GetInstructionsByRecipe(ctx context.Context, recipeID pgtype.UUID) ([]generated.RecipeInstruction, error)
	DeleteOldImportJobs(ctx context.Context) error
	DeleteStaleImportJobs(ctx context.Context) error
	CreateRecipeImage(ctx context.Context, arg generated.CreateRecipeImageParams) (generated.RecipeImage, error)
	UpdateRecipeThumbnail(ctx context.Context, arg generated.UpdateRecipeThumbnailParams) error
	UpdateRecipeEmbedding(ctx context.Context, arg generated.UpdateRecipeEmbeddingParams) error
	GetSocialMediaOwnerByOrigin(ctx context.Context, arg generated.GetSocialMediaOwnerByOriginParams) (generated.SocialMediaOwner, error)
	CreateSocialMediaOwner(ctx context.Context, arg generated.CreateSocialMediaOwnerParams) (generated.SocialMediaOwner, error)
	// Category methods
	GetOrCreateCuisineCategory(ctx context.Context, name string) (pgtype.UUID, error)
	AddRecipeCuisineCategory(ctx context.Context, arg generated.AddRecipeCuisineCategoryParams) error
	GetOrCreateMealType(ctx context.Context, name string) (pgtype.UUID, error)
	AddRecipeMealType(ctx context.Context, arg generated.AddRecipeMealTypeParams) error
	GetOrCreateOccasion(ctx context.Context, name string) (pgtype.UUID, error)
	AddRecipeOccasion(ctx context.Context, arg generated.AddRecipeOccasionParams) error
	GetOrCreateDietaryRestriction(ctx context.Context, name string) (pgtype.UUID, error)
	AddRecipeDietaryRestriction(ctx context.Context, arg generated.AddRecipeDietaryRestrictionParams) error
	GetOrCreateEquipment(ctx context.Context, name string) (pgtype.UUID, error)
	AddRecipeEquipment(ctx context.Context, arg generated.AddRecipeEquipmentParams) error
	GetCuisineCategoriesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetMealTypesByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetOccasionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetDietaryRestrictionsByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
	GetEquipmentByUser(ctx context.Context, userID pgtype.UUID) ([]string, error)
}

type InstagramScraper interface {
	Scrape(ctx context.Context, postURL string) (*scraper.InstagramPost, error)
}

type TikTokScraper interface {
	Scrape(ctx context.Context, postURL string) (*scraper.TikTokPost, error)
}
type FirecrawlScraper interface {
	Scrape(ctx context.Context, postURL string) (*scraper.FirecrawlPost, error)
}

type OpenAIClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

type TranscriptionClient interface {
	TranscribeVideo(ctx context.Context, videoURL string) (string, error)
}

type GroqClient interface {
	GenerateRecipe(ctx context.Context, caption, transcript, platform string) (*groq.Recipe, error)
	GenerateCategories(ctx context.Context, prompt string) (*ai.CategoryAIResponse, error)
	GenerateRichInstructions(ctx context.Context, recipe *groq.Recipe) (*recipe.RichInstructionResponse, error)
}

type StorageClient interface {
	UploadImageWithHash(ctx context.Context, bucket, path, sourceURL string, data []byte) (string, error)
	GetImageByHash(ctx context.Context, hash string) (*storage.ExistingImageResponse, error)
}

type ProgressBroadcasterInterface interface {
	Broadcast(userID string, update ProgressUpdate) error
}

type RecipeProcessor struct {
	db        DBQueries
	instagram InstagramScraper
	tiktok    TikTokScraper
	firecrawl FirecrawlScraper

	openai        OpenAIClient
	transcription TranscriptionClient
	groq          GroqClient
	storage       StorageClient
	broadcaster   ProgressBroadcasterInterface
	metrics       *WorkerMetrics
	asynqClient   *asynq.Client
}

func NewRecipeProcessor(
	db DBQueries,
	instagram InstagramScraper,
	tiktok TikTokScraper,
	firecrawl FirecrawlScraper,

	openaiClient OpenAIClient,
	transcriptionClient TranscriptionClient,
	groqClient GroqClient,
	storageClient StorageClient,
	broadcaster ProgressBroadcasterInterface,
	metrics *WorkerMetrics,
	asynqClient *asynq.Client,
) *RecipeProcessor {
	return &RecipeProcessor{
		db:        db,
		instagram: instagram,
		tiktok:    tiktok,
		firecrawl: firecrawl,

		openai:        openaiClient,
		transcription: transcriptionClient,
		groq:          groqClient,
		storage:       storageClient,
		broadcaster:   broadcaster,
		metrics:       metrics,
		asynqClient:   asynqClient,
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
	start := time.Now()
	var status = "success"
	defer func() {
		duration := time.Since(start).Seconds()
		p.metrics.RecordJob(ctx, "process_recipe", status, duration)
	}()

	var payload ProcessRecipePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		status = "failure"
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	jobID := payload.JobID
	userID := payload.UserID
	url := payload.URL

	slog.Info("Processing recipe", "job_id", jobID, "url", url)

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Fetching post content...")

	var caption, platform, imageURL, videoURL string
	var ownerUsername, ownerAvatar, ownerID string

	if scraper.IsInstagramURL(url) {
		platform = "instagram"
		post, err := p.instagram.Scrape(ctx, url)
		if err != nil {
			status = "failure"
			p.markFailed(ctx, jobID, userID, fmt.Sprintf("Instagram scrape failed: %v", err))
			return err
		}
		caption = post.Caption
		imageURL = post.ImageURL
		videoURL = post.VideoURL
		ownerUsername = post.OwnerUsername
		ownerAvatar = post.OwnerAvatar
		ownerID = post.OwnerID

	} else if scraper.IsTikTokURL(url) {
		platform = "tiktok"
		post, err := p.tiktok.Scrape(ctx, url)
		if err != nil {
			status = "failure"
			p.markFailed(ctx, jobID, userID, fmt.Sprintf("TikTok scrape failed: %v", err))
			return err
		}
		caption = post.Caption
		imageURL = post.ThumbnailURL
		videoURL = post.VideoURL
		ownerUsername = post.OwnerUsername
		ownerAvatar = post.OwnerAvatar
		ownerID = post.OwnerID

	} else if p.firecrawl != nil {
		// Firecrawl handles all other URLs (only if enabled/configured)
		platform = "firecrawl"
		post, err := p.firecrawl.Scrape(ctx, url)
		if err != nil {
			status = "failure"
			p.markFailed(ctx, jobID, userID, fmt.Sprintf("Firecrawl scrape failed: %v", err))
			return err
		}
		caption = post.Caption
		imageURL = post.ImageURL
		videoURL = post.VideoURL
		ownerUsername = post.OwnerUsername
		ownerAvatar = post.OwnerAvatar
		ownerID = post.OwnerID
	} else {
		// Firecrawl not enabled
		status = "failure"
		p.markFailed(ctx, jobID, userID, "Invalid URL: must be Instagram or TikTok (Firecrawl not enabled)")
		return fmt.Errorf("invalid URL: Firecrawl not enabled")
	}

	validationResult := validation.QuickValidate(caption, "")
	if !validationResult.IsValid {
		status = "failure"
		errMsg := fmt.Sprintf("Content validation failed: %s", validationResult.Reason)
		p.markFailed(ctx, jobID, userID, errMsg)
		return errors.NewValidationError(errMsg, "CONTENT_NOT_RECIPE", "")
	}
	slog.Info("Content validation passed", "confidence", string(validationResult.Confidence), "reason", validationResult.Reason)

	var transcript string
	var imageData []byte

	// Run transcription and image download in parallel
	if videoURL != "" || imageURL != "" {
		p.updateProgress(ctx, jobID, userID, "EXECUTING", "Processing video and image content...")

		funcs := []ParallelFunc{}

		// Add transcription function if videoURL exists
		if videoURL != "" {
			funcs = append(funcs, func(ctx context.Context) error {
				transcriptResult, err := p.transcription.TranscribeVideo(ctx, videoURL)
				if err != nil {
					return err
				}
				transcript = transcriptResult
				return nil
			})
		}

		// Add image download function if imageURL exists
		if imageURL != "" {
			funcs = append(funcs, func(ctx context.Context) error {
				data, err := downloadImage(ctx, imageURL)
				if err != nil {
					return err
				}
				imageData = data
				return nil
			})
		}

		// Execute parallel functions
		result := RunParallel(ctx, funcs)

		// Check for transcription errors (fail job)
		for _, err := range result.Errors {
			// Check if this error is from transcription (videoURL != "" and we have a transcript error)
			if videoURL != "" && err != nil && transcript == "" {
				status = "failure"
				p.markFailed(ctx, jobID, userID, fmt.Sprintf("Transcription failed: %v", err))
				return err
			}
		}

		// Image download errors are logged but don't fail the job
		// (error is already logged by RunParallel)
	}
	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Generating recipe with AI...")

	recipe, err := p.groq.GenerateRecipe(ctx, caption, transcript, platform)
	if err != nil {
		status = "failure"
		p.markFailed(ctx, jobID, userID, fmt.Sprintf("Recipe generation failed: %v", err))
		return err
	}

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Generating categories with AI...")

	categoryService := ai.NewCategoryService(p.db, p.groq)
	categories, err := utils.WithRetry(ctx, func(ctx context.Context) (*ai.CategorySuggestions, error) {
		return categoryService.SuggestCategories(ctx, ai.RecipeInfo{
			Name:        recipe.RecipeName,
			Description: recipe.Description,
			Ingredients: extractIngredientNames(recipe.Ingredients),
		}, userID)
	}, utils.DefaultRetryConfig())
	if err == nil {
		recipe.CuisineCategories = categories.CuisineCategories
		recipe.MealTypes = categories.MealTypes
		recipe.Occasions = categories.Occasions
		recipe.DietaryRestrictions = categories.DietaryRestrictions
		recipe.Equipment = categories.Equipment
	} else {
		slog.Error("Category generation failed after retries", "error", err, "recipe_name", recipe.RecipeName)
		sentrylib.CaptureError(err, map[string]string{
			"recipe_name": recipe.RecipeName,
			"component":   "category_generation",
		})
	}

	validationConfig := validation.RecipeOutputValidationConfig{
		MinIngredients:      2,
		MinInstructions:     2,
		MaxPlaceholderRatio: 0.2,
	}

	vRecipe := validation.Recipe{
		RecipeName:          recipe.RecipeName,
		Description:         recipe.Description,
		PrepTime:            recipe.PrepTime,
		CookingTime:         recipe.CookingTime,
		TotalTime:           recipe.TotalTime,
		OriginalServings:    recipe.OriginalServings,
		DifficultyRating:    recipe.DifficultyRating,
		FocusedDiet:         recipe.FocusedDiet,
		EstimatedCalories:   recipe.EstimatedCalories,
		Ingredients:         convertIngredients(recipe.Ingredients),
		Instructions:        convertInstructions(recipe.Instructions),
		Nutrition:           convertNutrition(recipe.Nutrition),
		CuisineCategories:   recipe.CuisineCategories,
		MealTypes:           recipe.MealTypes,
		Occasions:           recipe.Occasions,
		DietaryRestrictions: recipe.DietaryRestrictions,
		Equipment:           recipe.Equipment,
		Language:            recipe.Language,
	}

	result := validation.ValidateRecipe(vRecipe, validationConfig)
	if !result.IsValid {
		status = "failure"
		errMsg := fmt.Sprintf("Recipe validation failed (quality score: %d): %s", result.QualityScore, strings.Join(result.Issues, ", "))
		p.markFailed(ctx, jobID, userID, errMsg)
		return errors.NewValidationError(errMsg, "LOW_QUALITY_RECIPE", "Try providing a more detailed video or transcript.")
	}
	slog.Info("Recipe validation passed", "quality_score", result.QualityScore, "has_placeholders", result.HasPlaceholders)

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Saving recipe to database...")

	var ownerUUID pgtype.UUID
	if ownerID != "" {
		p.updateProgress(ctx, jobID, userID, "EXECUTING", "Saving recipe owner...")

		owner, err := p.db.GetSocialMediaOwnerByOrigin(ctx, generated.GetSocialMediaOwnerByOriginParams{
			OriginID: ownerID,
			Platform: generated.SocialMediaPlatform(platform),
		})

		if err != nil {
			var storedImageID pgtype.Text

			if ownerAvatar != "" {
				avatarData, err := downloadImage(ctx, ownerAvatar)
				if err == nil {
					hash := storage.HashContent(avatarData)
					path := fmt.Sprintf("user_avatars/%s", hash)
					_, err := p.storage.UploadImageWithHash(ctx, "recipes", path, ownerAvatar, avatarData)
					if err == nil {
						if existing, err := p.storage.GetImageByHash(ctx, hash); err == nil {
							storedImageID = pgtype.Text{String: existing.ID, Valid: true}
						}
					}
				}
			}

			newOwner, err := p.db.CreateSocialMediaOwner(ctx, generated.CreateSocialMediaOwnerParams{
				Username:                ownerUsername,
				ProfilePicStoredImageID: storedImageID,
				OriginID:                ownerID,
				Platform:                generated.SocialMediaPlatform(platform),
			})
			if err == nil {
				ownerUUID = newOwner.ID
			} else {
				slog.Error("Failed to create social media owner", "error", err)
			}
		} else {
			ownerUUID = owner.ID
		}
	}

	recipeUUID := parseUUID(uuid.New().String())
	userUUID := parseUUID(userID)

	var difficultyRating pgtype.Int2
	if recipe.DifficultyRating != nil {
		difficultyRating = pgtype.Int2{Int16: int16(*recipe.DifficultyRating), Valid: true}
	}

	var origin generated.RecipeOrigin
	if platform == "instagram" {
		origin = generated.RecipeOriginInstagram
	} else if platform == "tiktok" {
		origin = generated.RecipeOriginTiktok
	} else {
		origin = generated.RecipeOriginFirecrawl
	}

	savedRecipe, err := p.db.CreateRecipe(ctx, generated.CreateRecipeParams{
		ID:                  recipeUUID,
		CreatedBy:           userUUID,
		RecipeName:          recipe.RecipeName,
		Description:         pgtype.Text{String: recipe.Description, Valid: recipe.Description != ""},
		PrepTime:            pgtype.Int4{Int32: int32(ptrToInt(recipe.PrepTime)), Valid: recipe.PrepTime != nil},
		CookingTime:         pgtype.Int4{Int32: int32(ptrToInt(recipe.CookingTime)), Valid: recipe.CookingTime != nil},
		TotalTime:           pgtype.Int4{Int32: int32(ptrToInt(recipe.TotalTime)), Valid: recipe.TotalTime != nil},
		OriginalServingSize: pgtype.Int4{Int32: int32(ptrToInt(recipe.OriginalServings)), Valid: recipe.OriginalServings != nil},
		DifficultyRating:    difficultyRating,
		FocusedDiet:         pgtype.Text{String: recipe.FocusedDiet, Valid: recipe.FocusedDiet != ""},
		EstimatedCalories:   pgtype.Int4{Int32: int32(ptrToInt(recipe.EstimatedCalories)), Valid: recipe.EstimatedCalories != nil},
		Origin:              origin,
		Url:                 url,
		OwnerID:             ownerUUID,
		ThumbnailID:         pgtype.UUID{},
		Language:            pgtype.Text{String: recipe.Language, Valid: recipe.Language != ""},
	})
	if err != nil {
		status = "failure"
		p.markFailed(ctx, jobID, userID, fmt.Sprintf("Failed to save recipe: %v", err))
		return err
	}

	// Save categories
	cuisineSaved := 0
	for _, cat := range recipe.CuisineCategories {
		catID, err := p.db.GetOrCreateCuisineCategory(ctx, cat)
		if err == nil {
			err = p.db.AddRecipeCuisineCategory(ctx, generated.AddRecipeCuisineCategoryParams{
				RecipeID:          savedRecipe.ID,
				CuisineCategoryID: catID,
			})
			if err == nil {
				cuisineSaved++
			}
		}
	}
	if cuisineSaved == 0 {
		slog.Error("No cuisine categories persisted for recipe", "recipe_id", savedRecipe.ID, "recipe_name", recipe.RecipeName)
		sentrylib.CaptureError(fmt.Errorf("no cuisine categories persisted for recipe %s", recipe.RecipeName), map[string]string{
			"recipe_name": recipe.RecipeName,
			"component":   "category_persistence",
		})
	}

	for _, mt := range recipe.MealTypes {
		mtID, err := p.db.GetOrCreateMealType(ctx, mt)
		if err == nil {
			p.db.AddRecipeMealType(ctx, generated.AddRecipeMealTypeParams{
				RecipeID:   savedRecipe.ID,
				MealTypeID: mtID,
			})
		}
	}

	for _, occ := range recipe.Occasions {
		occID, err := p.db.GetOrCreateOccasion(ctx, occ)
		if err == nil {
			p.db.AddRecipeOccasion(ctx, generated.AddRecipeOccasionParams{
				RecipeID:   savedRecipe.ID,
				OccasionID: occID,
			})
		}
	}

	for _, dr := range recipe.DietaryRestrictions {
		drID, err := p.db.GetOrCreateDietaryRestriction(ctx, dr)
		if err == nil {
			p.db.AddRecipeDietaryRestriction(ctx, generated.AddRecipeDietaryRestrictionParams{
				RecipeID:             savedRecipe.ID,
				DietaryRestrictionID: drID,
			})
		}
	}

	for _, eq := range recipe.Equipment {
		eqID, err := p.db.GetOrCreateEquipment(ctx, eq)
		if err == nil {
			p.db.AddRecipeEquipment(ctx, generated.AddRecipeEquipmentParams{
				RecipeID:    savedRecipe.ID,
				EquipmentID: eqID,
			})
		}
	}

	savedIngredientIDs := make([]string, len(recipe.Ingredients))
	for i, ing := range recipe.Ingredients {
		totalQty := string(ing.Quantity)

		var perServingQty string
		if recipe.OriginalServings != nil && *recipe.OriginalServings > 0 {
			if totalNum, err := strconv.ParseFloat(totalQty, 64); err == nil {
				perServingNum := totalNum / float64(*recipe.OriginalServings)
				perServingQty = strconv.FormatFloat(perServingNum, 'f', -1, 64)
			} else {
				perServingQty = totalQty
			}
		} else {
			perServingQty = totalQty
		}

		savedIng, err := p.db.CreateIngredient(ctx, generated.CreateIngredientParams{
			RecipeID:         savedRecipe.ID,
			Quantity:         pgtype.Text{String: perServingQty, Valid: true},
			TotalQuantity:    pgtype.Text{String: totalQty, Valid: true},
			Unit:             pgtype.Text{String: ing.Unit, Valid: ing.Unit != ""},
			OriginalQuantity: pgtype.Text{String: string(ing.OriginalQuantity), Valid: ing.OriginalQuantity != ""},
			OriginalUnit:     pgtype.Text{String: ing.OriginalUnit, Valid: ing.OriginalUnit != ""},
			Name:             ing.Name,
		})
		if err != nil {
			slog.Error("Failed to save ingredient", "error", err)
			continue
		}
		savedIngredientIDs[i] = pgUUIDToString(savedIng.ID)
	}

	savedInstructions := make([]generated.RecipeInstruction, len(recipe.Instructions))
	for i, inst := range recipe.Instructions {
		var timerData []byte
		if len(inst.TimerData) > 0 {
			var err error
			timerData, err = json.Marshal(inst.TimerData)
			if err != nil {
				slog.Error("Failed to marshal timer data", "error", err)
			}
		}

		savedInst, err := p.db.CreateInstruction(ctx, generated.CreateInstructionParams{
			RecipeID:               savedRecipe.ID,
			StepNumber:             int32(i + 1),
			Instruction:            inst.Instruction,
			TimerData:              timerData,
			InstructionRich:        pgtype.Text{},
			InstructionRichVersion: pgtype.Int4{},
		})
		if err != nil {
			slog.Error("Failed to save instruction", "error", err)
			continue
		}
		savedInstructions[i] = savedInst
	}

	p.updateProgress(ctx, jobID, userID, "EXECUTING", "Generating rich instruction formatting...")

	for i := range recipe.Ingredients {
		if i < len(savedIngredientIDs) && savedIngredientIDs[i] != "" {
			recipe.Ingredients[i].ID = savedIngredientIDs[i]
		}
	}

	richResp, err := p.groq.GenerateRichInstructions(ctx, recipe)
	if err != nil {
		slog.Warn("Failed to generate rich instructions, enqueueing retry", "error", err, "recipe_name", recipe.RecipeName)
		p.enqueueRichInstructionsRetry(ctx, pgUUIDToString(savedRecipe.ID))
	} else if richResp != nil {
		for i, inst := range richResp.Instructions {
			if i < len(savedInstructions) {
				err := p.db.UpdateInstructionRich(ctx, generated.UpdateInstructionRichParams{
					InstructionRich:        pgtype.Text{String: inst.InstructionRich, Valid: inst.InstructionRich != ""},
					InstructionRichVersion: pgtype.Int4{Int32: int32(richResp.PromptVersion), Valid: richResp.PromptVersion > 0},
					ID:                     savedInstructions[i].ID,
				})
				if err != nil {
					slog.Error("Failed to update instruction with rich text", "error", err, "step", i+1)
				}
			}
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

	if imageURL != "" && imageData != nil {
		p.updateProgress(ctx, jobID, userID, "EXECUTING", "Processing recipe image...")

		hash := storage.HashContent(imageData)
		path := fmt.Sprintf("post_images/%s", hash)
		_, err := p.storage.UploadImageWithHash(ctx, "recipes", path, imageURL, imageData)
		if err != nil {
			slog.Error("Failed to upload image", "error", err)
		} else {
			existing, err := p.storage.GetImageByHash(ctx, hash)
			if err != nil || existing == nil {
				slog.Error("Failed to get stored image after upload", "error", err)
			} else {
				storedImageUUID := parseUUID(existing.ID)

				recipeImage, err := p.db.CreateRecipeImage(ctx, generated.CreateRecipeImageParams{
					RecipeID:      savedRecipe.ID,
					StoredImageID: storedImageUUID,
					ImageType:     "full",
				})
				if err != nil {
					slog.Error("Failed to create recipe image record", "error", err)
				} else {
					err = p.db.UpdateRecipeThumbnail(ctx, generated.UpdateRecipeThumbnailParams{
						ID:          savedRecipe.ID,
						ThumbnailID: recipeImage.ID,
					})
					if err != nil {
						slog.Error("Failed to update recipe thumbnail", "error", err)
					}
				}
			}
		}
	}
	// Enqueue embedding generation task
	if p.asynqClient != nil {
		embedTask, err := NewGenerateEmbeddingTask(GenerateEmbeddingPayload{
			RecipeID: pgUUIDToString(savedRecipe.ID),
		})
		if err == nil {
			_, err = p.asynqClient.Enqueue(embedTask)
			if err != nil {
				slog.Error("Failed to enqueue embedding task", "error", err)
			} else {
				slog.Info("Enqueued embedding task", "recipe_id", pgUUIDToString(savedRecipe.ID))
			}
		}
	}

	p.updateProgress(ctx, jobID, userID, "COMPLETED", "Recipe saved successfully!")

	return nil
}

func (p *RecipeProcessor) HandleGenerateEmbedding(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	var status = "success"
	defer func() {
		duration := time.Since(start).Seconds()
		p.metrics.RecordJob(ctx, "generate_embedding", status, duration)
	}()

	var payload GenerateEmbeddingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		status = "failure"
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	recipeUUID := parseUUID(payload.RecipeID)
	recipe, err := p.db.GetRecipe(ctx, recipeUUID)
	if err != nil {
		status = "failure"
		return fmt.Errorf("recipe not found: %w", err)
	}

	text := recipe.RecipeName + " " + recipe.Description.String
	ingredients, _ := p.db.GetIngredientsByRecipe(ctx, recipe.ID)
	for _, ing := range ingredients {
		text += " " + ing.Name
	}

	embedding, err := p.openai.GenerateEmbedding(ctx, text)
	if err != nil {
		status = "failure"
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	err = p.db.UpdateRecipeEmbedding(ctx, generated.UpdateRecipeEmbeddingParams{
		ID:        recipe.ID,
		Embedding: ptrVector(pgvector.NewVector(embedding)),
	})
	if err != nil {
		status = "failure"
		return fmt.Errorf("failed to save embedding: %w", err)
	}

	slog.Info("Embedding generated and saved", "recipe_id", payload.RecipeID)
	return nil
}

func (p *RecipeProcessor) HandleCleanupJobs(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	var status = "success"
	defer func() {
		duration := time.Since(start).Seconds()
		p.metrics.RecordJob(ctx, "cleanup_jobs", status, duration)
	}()

	slog.Info("Running cleanup job")

	err := p.db.DeleteOldImportJobs(ctx)
	if err != nil {
		status = "failure"
		slog.Error("Failed to delete old import jobs", "error", err)
		return err
	}

	err = p.db.DeleteStaleImportJobs(ctx)
	if err != nil {
		status = "failure"
		slog.Error("Failed to delete stale import jobs", "error", err)
		return err
	}

	slog.Info("Cleanup job completed successfully")
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

func downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func convertIngredients(ings []groq.Ingredient) []validation.Ingredient {
	result := make([]validation.Ingredient, len(ings))
	for i, ing := range ings {
		result[i] = validation.Ingredient{
			OriginalQuantity: string(ing.OriginalQuantity),
			OriginalUnit:     ing.OriginalUnit,
			Quantity:         string(ing.Quantity),
			Unit:             ing.Unit,
			Name:             ing.Name,
		}
	}
	return result
}

func convertInstructions(insts []groq.Instruction) []validation.Instruction {
	result := make([]validation.Instruction, len(insts))
	for i, inst := range insts {
		result[i] = validation.Instruction{
			StepNumber:  inst.StepNumber,
			Instruction: inst.Instruction,
		}
	}
	return result
}

func convertNutrition(n groq.Nutrition) validation.Nutrition {
	return validation.Nutrition{
		Protein: n.Protein,
		Carbs:   n.Carbs,
		Fat:     n.Fat,
		Fiber:   n.Fiber,
	}
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
func ptrVector(v pgvector.Vector) *pgvector.Vector {
	return &v
}

func extractIngredientNames(ingredients []groq.Ingredient) []string {
	names := make([]string, len(ingredients))
	for i, ing := range ingredients {
		names[i] = ing.Name
	}
	return names
}

func formatQuantity(q string) string {
	if q == "" {
		return ""
	}
	return q
}

func (p *RecipeProcessor) enqueueRichInstructionsRetry(ctx context.Context, recipeID string) {
	if p.asynqClient == nil {
		return
	}
	task, err := NewGenerateRichInstructionsTask(GenerateRichInstructionsPayload{RecipeID: recipeID})
	if err == nil {
		_, err = p.asynqClient.Enqueue(task)
		if err != nil {
			slog.Error("Failed to enqueue rich instructions retry", "error", err, "recipe_id", recipeID)
		} else {
			slog.Info("Enqueued rich instructions retry", "recipe_id", recipeID)
		}
	}
}

// HandleGenerateRichInstructions retries rich instruction generation for a saved recipe.
func (p *RecipeProcessor) HandleGenerateRichInstructions(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	var status = "success"
	defer func() {
		duration := time.Since(start).Seconds()
		p.metrics.RecordJob(ctx, "generate_rich_instructions", status, duration)
	}()

	var payload GenerateRichInstructionsPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		status = "failure"
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	recipeUUID := parseUUID(payload.RecipeID)
	dbRecipe, err := p.db.GetRecipe(ctx, recipeUUID)
	if err != nil {
		status = "failure"
		return fmt.Errorf("recipe not found: %w", err)
	}

	ingredients, err := p.db.GetIngredientsByRecipe(ctx, dbRecipe.ID)
	if err != nil {
		status = "failure"
		return fmt.Errorf("failed to get ingredients: %w", err)
	}

	instructions, err := p.db.GetInstructionsByRecipe(ctx, dbRecipe.ID)
	if err != nil {
		status = "failure"
		return fmt.Errorf("failed to get instructions: %w", err)
	}

	// Build recipe struct from DB data
	r := &groq.Recipe{
		RecipeName: dbRecipe.RecipeName,
	}

	for _, ing := range ingredients {
		r.Ingredients = append(r.Ingredients, groq.Ingredient{
			ID:   pgUUIDToString(ing.ID),
			Name: ing.Name,
		})
	}

	for _, inst := range instructions {
		instruction := groq.Instruction{
			StepNumber:  int(inst.StepNumber),
			Instruction: inst.Instruction,
		}
		if len(inst.TimerData) > 0 {
			var timers []recipe.Timer
			if err := json.Unmarshal(inst.TimerData, &timers); err == nil {
				instruction.TimerData = timers
			}
		}
		r.Instructions = append(r.Instructions, instruction)
	}

	richResp, err := p.groq.GenerateRichInstructions(ctx, r)
	if err != nil {
		status = "failure"
		return fmt.Errorf("failed to generate rich instructions: %w", err)
	}

	if richResp != nil {
		for i, inst := range richResp.Instructions {
			if i < len(instructions) {
				err := p.db.UpdateInstructionRich(ctx, generated.UpdateInstructionRichParams{
					InstructionRich:        pgtype.Text{String: inst.InstructionRich, Valid: inst.InstructionRich != ""},
					InstructionRichVersion: pgtype.Int4{Int32: int32(richResp.PromptVersion), Valid: richResp.PromptVersion > 0},
					ID:                     instructions[i].ID,
				})
				if err != nil {
					slog.Error("Failed to update instruction with rich text", "error", err, "step", i+1)
				}
			}
		}
	}

	slog.Info("Rich instructions generated and saved", "recipe_id", payload.RecipeID)
	return nil
}

// HandleInstagramRetry handles retry attempts for failed Instagram scrapes.
// It uses cached data if available and applies fast retry logic.
func (p *RecipeProcessor) HandleInstagramRetry(ctx context.Context, t *asynq.Task) error {
	start := time.Now()
	var status = "success"
	defer func() {
		duration := time.Since(start).Seconds()
		p.metrics.RecordJob(ctx, "instagram_retry", status, duration)
	}()

	var payload InstagramRetryPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		status = "failure"
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	jobID := payload.JobID
	url := payload.URL

	slog.Info("Processing Instagram retry", "job_id", jobID, "url", url)

	// For now, just log the retry - full implementation would re-scrape
	// This is a placeholder for the retry logic
	slog.Info("Instagram retry processed", "job_id", jobID, "url", url)

	return nil
}
