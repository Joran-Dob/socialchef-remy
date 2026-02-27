# REBUILD COMPARISON: Complete Analysis of What Was Lost in Go Migration

## Executive Summary

After analyzing both the old TypeScript/Supabase implementation and the new Go implementation, I've identified **critical functionality that was not ported** during the rebuild. This document provides a comprehensive comparison of ALL missing features, not just the images and Instagram scraping issues mentioned.

---

## 🔴 Critical Issues (Blocking/High Impact)

### 1. Complete Image Processing Pipeline (MISSING)

**Old Implementation (TypeScript):**
```typescript
// trigger/repositories/recipeRepository.ts Lines 111-126
private async downloadAndHashImage(url: string): Promise<{ buffer: Uint8Array; hash: string }> {
  const response = await fetch(url);
  const buffer = new Uint8Array(await response.arrayBuffer());
  const hash = createHash("sha256").update(buffer).digest("hex");
  return { buffer, hash };
}

// Lines 128-155: Upload to Supabase Storage
private async storeImage(imageData, imageSourcePath): Promise<string> {
  const storagePath = `${imageSourcePath}/${imageData.hash}`;
  await this.supabaseServiceRoleClient.storage
    .from("recipes")
    .upload(storagePath, imageData.buffer, {
      contentType: "image/jpeg",
      upsert: false,
    });
  return storagePath;
}

// Lines 197-257: Save recipe images with full workflow
async saveRecipeImages(recipeId: string, images: RecipeImage[]): Promise<RecipeImage[]> {
  for (const image of images) {
    const imageData = await this.downloadAndHashImage(image.url);     // 1. Download
    const storagePath = await this.storeImage(imageData, path);       // 2. Store
    const storedImageId = await this.saveStoredImage(storagePath, ...); // 3. Record
    await this.db.from("recipe_images").insert({...});                // 4. Link
  }
}
```

**Called from:** `processRecipe.ts` Lines 586-614
```typescript
// Save images
if (post.image) {
  await recipeRepository.saveRecipeImages(savedRecipe.id!, [post.image]);
}

// Save thumbnail and update recipe
if (post.thumbnail) {
  const savedImages = await recipeRepository.saveRecipeImages(savedRecipe.id!, [post.thumbnail]);
  thumbnailId = savedImages[0].id;
  savedRecipe = await recipeRepository.updateRecipeThumbnail(savedRecipe.id!, thumbnailId);
}
```

**New Implementation (Go):**
```go
// internal/worker/handlers.go Lines 74-93
if scraper.IsInstagramURL(url) {
    post, err := p.instagram.Scrape(ctx, url)
    caption = post.Caption
    // ❌ post.ImageURL NEVER USED
}

// Lines 120-133: Recipe created with empty ThumbnailID
savedRecipe, err := p.db.CreateRecipe(ctx, generated.CreateRecipeParams{
    ThumbnailID: pgtype.UUID{},  // ❌ Always empty!
})
```

**Impact:**
- `stored_images` table: Empty
- `recipe_images` table: Empty
- `recipes.thumbnail_id`: Always NULL
- All imported recipes have broken/missing images

---

### 2. Instagram Scraping Resilience (SEVERELY DEGRADED)

**Old Implementation (TypeScript) - Full Headers:**
```typescript
// instagramRepository.ts Lines 84-108
private getProxyRequestConfig(graphqlUrl: URL, proxyApiKey: string): RequestInit {
  return {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-api-key": proxyApiKey,
    },
    body: JSON.stringify({
      url: graphqlUrl.toString(),
      method: "POST",
      headers: {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)...",
        "Content-Type": "application/x-www-form-urlencoded",
        "X-IG-App-ID": this.INSTAGRAM_APP_ID,
        "X-FB-LSD": this.LSD,                    // ✅ PRESENT
        "X-ASBD-ID": "129477",                   // ✅ PRESENT
        "Sec-Fetch-Site": "same-origin",         // ✅ PRESENT
      },
    }),
  };
}
```

**Error Handling (Lines 110-137):**
```typescript
private async parseInstagramResponse(response: Response) {
  if (!response.ok) {
    throw new Error(`Proxy request failed with status: ${response.status}`);
  }
  const json = await response.json() as ProxyResponse;
  if (!json.data) {
    throw new Error("Invalid Instagram GraphQL response: missing data field");
  }
  try {
    const data = JSON.parse(json.data).data;
    if (!data?.xdt_shortcode_media) {
      throw new Error("Invalid Instagram GraphQL response: missing xdt_shortcode_media");
    }
    return data.xdt_shortcode_media;
  } catch (error) {
    throw new Error(`Failed to parse Instagram response: ${error.message}`);
  }
}
```

**Task Configuration with Retry (processRecipe.ts Lines 115-125):**
```typescript
export const processRecipeTask = task({
  id: "process-recipe",
  maxDuration: 300, // 5 minutes max
  retry: {
    maxAttempts: 3,
    minTimeoutInMs: 1000,
    maxTimeoutInMs: 30000,
    factor: 2,
    randomize: true,
  },
  // ...
});
```

**New Implementation (Go) - Minimal Headers:**
```go
// internal/services/scraper/instagram.go Lines 81-88
reqBody := map[string]interface{}{
    "url":    graphQLURL,
    "method": "POST",
    "headers": map[string]string{
        "Content-Type": "application/x-www-form-urlencoded",
        "X-IG-App-ID":  "936619743392459",
        // ❌ Missing: User-Agent, X-FB-LSD, X-ASBD-ID, Sec-Fetch-Site
    },
}
```

**Error Handling (Lines 98-142):**
```go
resp, err := s.httpClient.Do(req)
if err != nil {
    return nil, err  // ❌ No retry
}
if resp.StatusCode == 429 {
    return nil, ErrRateLimited  // ❌ No backoff
}
// ❌ Missing: Response validation, detailed error messages
```

**HTTP Client (Lines 29-35):**
```go
func NewInstagramScraper(proxyURL, proxyKey string) *InstagramScraper {
    return &InstagramScraper{
        httpClient: &http.Client{},  // ❌ No timeout!
    }
}
```

**Task Retry:** Not configured in Asynq task setup

**Missing Features:**
1. ❌ No HTTP timeout (TikTok has 180s, Instagram has none)
2. ❌ No retry logic for transient failures
3. ❌ Missing 4 critical Instagram headers
4. ❌ No exponential backoff for rate limits
5. ❌ Less robust error messages
6. ❌ No task-level retry configuration

---

### 3. Audio Transcript Generation (COMPLETELY MISSING)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 321-359
let audioTranscript: string | undefined;
if (post.video_url) {
  currentStep = ProcessingStep.GENERATING_AUDIO_TRANSCRIPT;
  try {
    audioTranscript = await openaiAudioRepository.getRecipeAudioTranscript(post);
    logger.log("Audio transcript generated", {
      transcriptLength: audioTranscript?.length || 0,
    });
  } catch (error) {
    logger.warn("Audio transcript generation failed, continuing without it");
    audioTranscript = undefined; // Continue without - not critical
  }
}
```

**New Implementation:**
```go
// ❌ COMPLETELY MISSING - No audio processing at all
```

**Impact:** Recipes from videos have less context for AI generation

---

### 4. Content Validation Before AI Generation (MISSING)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 362-397
logger.log("Step 2.5: Validating content quality...");
const validationConfig = ConfigService.getInstance().getValidationConfig();

const contentValidation = await validateContent(
  post.description,
  audioTranscript,
  validationConfig,
  groqClient,
  validatedPayload.origin,
);

if (!contentValidation.isValid) {
  logger.warn("Content validation failed", {
    reason: contentValidation.reason,
    confidence: contentValidation.confidence,
    missing: contentValidation.missing,
  });
  throw new AppError(
    ErrorType.VALIDATION_ERROR,
    contentValidation.reason,
    { statusCode: 400, errorCode: "INSUFFICIENT_CONTENT" },
  );
}
```

**New Implementation:**
```go
// ❌ COMPLETELY MISSING - AI generates from any content
// No validation that post has enough information for a recipe
```

**Impact:** AI tries to generate recipes from insufficient content, resulting in poor quality

---

### 5. Recipe Output Validation (MISSING)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 454-483
logger.log("Step 3.5: Validating generated recipe...");

const outputValidation = validateRecipeOutput(generatedRecipe, {
  minIngredients: validationConfig.minIngredients,
  minInstructions: validationConfig.minInstructions,
  maxPlaceholderRatio: validationConfig.maxPlaceholderRatio,
});

if (!outputValidation.isValid) {
  logger.warn("Recipe output validation failed", {
    issues: outputValidation.issues,
    hasPlaceholders: outputValidation.hasPlaceholders,
    qualityScore: outputValidation.qualityScore,
  });
  throw new AppError(
    ErrorType.VALIDATION_ERROR,
    `Generated recipe contains placeholders or insufficient detail (quality score: ${outputValidation.qualityScore}/100)`,
    { statusCode: 422, errorCode: "INVALID_RECIPE_OUTPUT" },
  );
}
```

**New Implementation:**
```go
// ❌ COMPLETELY MISSING - No validation of AI output
// Recipes with placeholders or insufficient detail are saved
```

**Impact:** Poor quality recipes with placeholders (e.g., "[ingredient]", "[time]") are saved to database

---

### 6. Embedding Persistence (INCOMPLETE)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 486-500
const embedding = await openaiRepository.generateRecipeEmbedding(generatedRecipe);

logger.log("Embedding generated", { dimensions: embedding.length });

// Lines 524-532: Saved with recipe
let savedRecipe = await recipeRepository.saveRecipe(
  {
    ...generatedRecipe.recipe,
    embedding: embedding,  // ✅ Saved to database
  },
  savedOwner.id!,
);
```

**New Implementation:**
```go
// internal/worker/handlers.go Lines 182-207
func (p *RecipeProcessor) HandleGenerateEmbedding(ctx context.Context, t *asynq.Task) error {
    // ... fetch recipe ...
    
    _, err = p.openai.GenerateEmbedding(ctx, text)  // ← Generated but discarded!
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }
    
    slog.Info("Embedding generated", "recipe_id", payload.RecipeID)
    // ❌ Never saved to database!
    return nil
}
```

**Impact:** 
- Wasted OpenAI API calls
- Vector search won't work (embeddings not stored)
- No semantic search capability

---

## 🟡 High Priority Issues

### 7. Comprehensive Error Classification (MISSING)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 646-731
} catch (error) {
  const classifiedError = classifyError(error, currentStep);
  
  await captureError(error, {
    tags: {
      taskId: "process-recipe",
      errorType: classifiedError.errorType,
      errorCode: classifiedError.errorCode,
      step: classifiedError.step,
    },
    extra: {
      userMessage: classifiedError.message,
      recoverySuggestion: classifiedError.recoverySuggestion,
      isRetryable: classifiedError.isRetryable,
    },
  });

  return {
    success: false,
    error: classifiedError,
  };
}
```

**New Implementation:**
```go
// Basic error logging only
slog.Error("Recipe processing failed", "error", err)
p.markFailed(ctx, jobID, userID, fmt.Sprintf("Instagram scrape failed: %v", err))
```

**Missing:**
- ❌ Error classification (VALIDATION_ERROR, API_ERROR, etc.)
- ❌ User-friendly error messages
- ❌ Recovery suggestions
- ❌ Retryable flag
- ❌ Error codes

---

### 8. Sentry Integration (MISSING)

**Old Implementation:**
```typescript
// Comprehensive Sentry integration throughout
import { captureError, initSentry, withTransaction } from "../../utils/sentryService.ts";

// In task:
initSentry("process-recipe");

return await withTransaction("processRecipeTask", "process-recipe", async () => {
  Sentry.setTag("taskId", "process-recipe");
  Sentry.setTag("origin", validatedPayload.origin);
  Sentry.setUser({ id: validatedPayload.userId });
  
  Sentry.addBreadcrumb({
    category: "task",
    message: "Starting recipe processing",
    level: "info",
    data: { url: validatedPayload.url },
  });
  
  // ...
  
  await captureError(error, { tags: {...}, extra: {...} });
});
```

**New Implementation:**
```go
// ❌ No Sentry integration at all
// Only basic slog logging
slog.Info("Processing recipe", "job_id", jobID, "url", url)
```

---

### 9. Progress Metadata (DEGRADED)

**Old Implementation:**
```typescript
metadata.set("progress", {
  step: currentStep,
  message: "Fetching social media post",
});
```

**New Implementation:**
```go
p.updateProgress(ctx, jobID, userID, "EXECUTING", "Fetching post content...")
```

**Difference:** Old had structured metadata for UI, new has simple strings

---

### 10. Owner Profile Picture Handling (MISSING)

**Old Implementation:**
```typescript
// recipeRepository.ts Lines 74-90
if (postOwner.profile_pic_url) {
  const imageData = await this.downloadAndHashImage(postOwner.profile_pic_url);
  const storagePath = await this.storeImage(imageData, ImageSourcePath.USER);
  const storedImageId = await this.saveStoredImage(storagePath, ...);
  owner.profile_pic_stored_image_id = storedImageId;
}
```

**New Implementation:**
```go
// ❌ Owner profile pictures not processed
// social_media_owners.profile_pic_stored_image_id will be empty
```

---

### 11. Recipe Categories Support (MISSING)

**Old Implementation:**
```typescript
// processRecipe.ts Lines 537-553
if (generatedRecipe.cuisine_categories?.length > 0 ||
    generatedRecipe.meal_types?.length > 0 ||
    generatedRecipe.occasions?.length > 0 ||
    generatedRecipe.dietary_restrictions?.length > 0 ||
    generatedRecipe.equipment?.length > 0) {
  await recipeRepository.saveRecipeCategories(
    savedRecipe.id!,
    generatedRecipe.cuisine_categories,
    generatedRecipe.meal_types,
    generatedRecipe.occasions,
    generatedRecipe.dietary_restrictions,
    generatedRecipe.equipment,
  );
}
```

**New Implementation:**
```go
// ❌ No category support in Go implementation
// Database has category tables but they're not populated
```

---

### 12. HTTP Timeout Configuration (INCONSISTENT)

**Old:** All HTTP clients had explicit timeouts
**New:** Only TikTok has timeout (180s), Instagram has none

---

## 🟢 Medium Priority Issues

### 13. Environment Variable Loading (MISSING)

**Old:** Used dotenv automatically
**New:** Missing `godotenv.Load()` in main entry points

**Fix needed:**
```go
import _ "github.com/joho/godotenv/autoload"
```

---

### 14. Cleanup Jobs Handler (PLACEHOLDER)

**New Implementation:**
```go
func (p *RecipeProcessor) HandleCleanupJobs(ctx context.Context, t *asynq.Task) error {
    slog.Info("Running cleanup job")
    return nil  // ❌ Does nothing
}
```

**Should:** Clean up old failed/completed jobs from database

---

### 15. TikTok Video Download (MISSING)

**Old Implementation:**
```typescript
// Used getPostWithVideo for TikTok
const post: Post = validatedPayload.origin === "instagram"
  ? await instagramRepository.getPost(validatedPayload.url)
  : await tiktokRepository.getPostWithVideo(validatedPayload.url);
```

**New Implementation:**
```go
// Both platforms just get caption
// No video download for transcript generation
```

---

### 16. LSD Token in Instagram Headers (MISSING)

**Old:** Had `X-FB-LSD` header
**New:** Missing - may cause scraping failures

---

## Summary Table: Complete Feature Comparison

| Feature | Old (TypeScript) | New (Go) | Priority | Impact |
|---------|------------------|----------|----------|--------|
| **Image Download** | ✅ Full pipeline | ❌ Missing | 🔴 Critical | No images in recipes |
| **Image Upload** | ✅ Supabase Storage | ❌ Missing | 🔴 Critical | No images in recipes |
| **Image Deduplication** | ✅ SHA256 hash | ❌ Missing | 🔴 Critical | Duplicate storage |
| **Recipe-Image Linking** | ✅ recipe_images | ❌ Missing | 🔴 Critical | No images in recipes |
| **Thumbnail Assignment** | ✅ updateRecipeThumbnail | ❌ Missing | 🔴 Critical | No thumbnails |
| **HTTP Timeout (Instagram)** | ✅ Configured | ❌ Missing | 🔴 Critical | Hanging requests |
| **Retry Logic** | ✅ 3 attempts + backoff | ❌ Missing | 🔴 Critical | Transient failures |
| **Audio Transcript** | ✅ Whisper integration | ❌ Missing | 🟡 High | Less context for AI |
| **Content Validation** | ✅ Pre-AI validation | ❌ Missing | 🟡 High | Poor quality recipes |
| **Output Validation** | ✅ Post-AI validation | ❌ Missing | 🟡 High | Placeholders in recipes |
| **Embedding Persistence** | ✅ Saved to DB | ❌ Generated only | 🟡 High | Search broken |
| **Instagram Headers** | ✅ 7 headers | ❌ 2 headers | 🟡 High | Scraping failures |
| **Error Classification** | ✅ Rich types | ❌ Basic | 🟡 Medium | Poor error UX |
| **Sentry Integration** | ✅ Full tracing | ❌ Missing | 🟡 Medium | No error tracking |
| **Owner Profile Picture** | ✅ Downloaded | ❌ Missing | 🟡 Medium | No owner avatars |
| **Recipe Categories** | ✅ Saved | ❌ Missing | 🟢 Low | No categorization |
| **Progress Metadata** | ✅ Structured | ❌ Strings | 🟢 Low | Less UI info |
| **TikTok Video Download** | ✅ For transcript | ❌ Missing | 🟢 Low | No video audio |
| **Environment Loading** | ✅ dotenv | ❌ Missing | 🟢 Low | Dev friction |
| **Cleanup Jobs** | ✅ Implemented | ❌ Placeholder | 🟢 Low | DB bloat |
| **LSD Token** | ✅ In headers | ❌ Missing | 🟢 Low | Scraping fragility |

---

## Root Cause Analysis

### Why Were These Features Lost?

1. **Image Processing:** Likely the most complex feature - requires downloading, hashing, storage upload, and database linking. May have been deprioritized during rebuild.

2. **Resilience Features:** Timeout, retry, and comprehensive error handling are often considered "nice to have" in initial implementations but are critical for production.

3. **Validation:** Content and output validation require additional AI calls (Groq) and complex logic - may have been skipped to simplify initial implementation.

4. **Audio Transcripts:** Requires video download + Whisper API integration - complex and may have been deferred.

5. **Observability:** Sentry and structured logging are often added later but are critical for debugging production issues.

### Critical Path Issues

The two features you mentioned (images and Instagram scraping) are indeed the most critical:

1. **Images not saving:** This is a complete feature gap - the code structure exists (storage client, DB tables) but the orchestration in the worker handler is missing.

2. **Instagram scraping failures:** The scraper is missing 4 headers, has no timeout, and no retry logic. This makes it fragile and prone to failure.

---

## Recommended Fix Priority

### Week 1: Critical (Fix Immediately)
1. ✅ Implement image download in worker handler
2. ✅ Add image upload using storage client
3. ✅ Link images to recipes in database
4. ✅ Add HTTP timeout to Instagram scraper (30s)
5. ✅ Add retry logic with exponential backoff

### Week 2: High Priority
6. ✅ Add missing Instagram headers
7. ✅ Test Instagram scraping thoroughly
8. ✅ Implement content validation
9. ✅ Implement output validation

### Week 3: Medium Priority  
10. ✅ Fix embedding persistence
11. ✅ Add audio transcript generation
12. ✅ Add Sentry integration

### Week 4: Polish
13. ✅ Add owner profile picture handling
14. ✅ Add recipe categories
15. ✅ Implement cleanup jobs
16. ✅ Add godotenv loading

---

## Next Steps

Run `/start-work` to begin implementing these fixes, starting with the critical image processing pipeline.

---

## Status: ✅ COMPLETE

Completed: 2026-02-27
