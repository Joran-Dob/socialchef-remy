# Investigation Plan: Rebuild Gaps in socialchef-remy

## Overview

This document outlines the investigation and remediation plan for issues discovered after rebuilding the Go project from the TypeScript Supabase implementation. The two primary reported issues are:

1. **Images are not being added to recipes**
2. **Instagram scraping often fails when it previously worked**

Based on comprehensive analysis of the codebase, several gaps and implementation issues have been identified.

---

## 🔴 Critical Issues (Blocking)

### Issue 1: Missing Image Processing Pipeline

**Problem:** 
The worker successfully scrapes Instagram/TikTok posts and extracts image URLs, but **never downloads or saves the images** to Supabase Storage or links them to recipes.

**Evidence:**

In `internal/worker/handlers.go`, the `HandleProcessRecipe` function:

```go
// Lines 74-93: Scraper returns ImageURL but it's never used
if scraper.IsInstagramURL(url) {
    platform = "instagram"
    post, err := p.instagram.Scrape(ctx, url)
    if err != nil {
        p.markFailed(ctx, jobID, userID, fmt.Sprintf("Instagram scrape failed: %v", err))
        return err
    }
    caption = post.Caption  // ← Only caption is extracted
    // post.ImageURL is IGNORED
}

// Lines 120-133: Recipe created with empty ThumbnailID
savedRecipe, err := p.db.CreateRecipe(ctx, generated.CreateRecipeParams{
    // ... other fields ...
    ThumbnailID: pgtype.UUID{},  // ← Always empty!
    // ...
})
```

**Missing Implementation:**
- No call to download image from `post.ImageURL`
- No call to `p.storage.UploadImageWithHash()` to save to Supabase
- No call to `p.db.CreateRecipeImage()` to link image to recipe
- No update to recipe's `thumbnail_id` field

**Impact:**
- All imported recipes have no images
- Users see broken/missing thumbnails
- `stored_images` and `recipe_images` tables remain empty

**Root Cause:**
The image processing logic was likely present in the original TypeScript implementation but was not ported during the Go rebuild.

---

### Issue 2: Instagram Scraper Fragility

**Problem:**
The Instagram scraper lacks proper resilience patterns and may be using outdated API parameters.

**Evidence:**

In `internal/services/scraper/instagram.go`:

```go
// Lines 29-35: No timeout configured
func NewInstagramScraper(proxyURL, proxyKey string) *InstagramScraper {
    return &InstagramScraper{
        proxyURL:   proxyURL,
        proxyKey:   proxyKey,
        httpClient: &http.Client{},  // ← Default timeout (none!)
    }
}

// Line 79: Hardcoded doc_id - these expire/change
graphQLURL := fmt.Sprintf("https://www.instagram.com/api/graphql?variables={\"shortcode\":\"%s\"}&doc_id=10015901848480474", shortcode)

// Lines 98-106: No retry logic
resp, err := s.httpClient.Do(req)
if err != nil {
    return nil, err  // ← Immediate failure, no retry
}
```

**Comparison with TikTok Scraper** (`internal/services/scraper/tiktok.go`):
```go
// Lines 30-32: Has timeout
httpClient: &http.Client{Timeout: 180 * time.Second},
```

**Missing:**
- HTTP timeout configuration
- Retry logic with exponential backoff
- Health check for the proxy service
- Documentation about the hardcoded `doc_id` 
- Fallback mechanism if scraping fails

**Impact:**
- Requests hang indefinitely on slow networks
- Transient failures cause complete job failure
- Instagram API changes break scraping without warning
- No visibility into why requests fail

---

## 🟡 Medium Priority Issues

### Issue 3: Embedding Not Persisted

**Problem:**
The `HandleGenerateEmbedding` task generates embeddings via OpenAI but never saves them to the database.

**Evidence:**

In `internal/worker/handlers.go` (lines 182-207):

```go
func (p *RecipeProcessor) HandleGenerateEmbedding(ctx context.Context, t *asynq.Task) error {
    // ... fetch recipe ...
    
    _, err = p.openai.GenerateEmbedding(ctx, text)  // ← Generated but not used!
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }
    
    slog.Info("Embedding generated", "recipe_id", payload.RecipeID)
    // ← Never saved to database!
    return nil
}
```

The database schema (`schema.sql`) includes a `vector` extension, suggesting embeddings should be stored.

**Impact:**
- Wasted API calls to OpenAI
- Search functionality (if implemented) won't work
- Vector similarity features unavailable

---

### Issue 4: Missing Environment Variable Loading

**Problem:**
The application doesn't load `.env` files automatically, requiring manual environment setup.

**Evidence:**

Both `cmd/server/main.go` and `cmd/worker/main.go` lack:
```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()  // ← Missing!
    // ...
}
```

The `godotenv` package is in `go.mod` but unused in main entry points.

**Impact:**
- Developer friction during local development
- Easy to forget setting env vars
- Scripts like `generate-jwt.go` have to manually load it

---

## 🟢 Low Priority Issues

### Issue 5: Placeholder Cleanup Handler

**Problem:**
The `HandleCleanupJobs` handler is a no-op placeholder.

**Evidence:**

```go
func (p *RecipeProcessor) HandleCleanupJobs(ctx context.Context, t *asynq.Task) error {
    slog.Info("Running cleanup job")
    return nil  // ← Does nothing
}
```

**Impact:**
- Stalled/failed jobs may accumulate in the database
- No automatic cleanup of old import jobs

---

### Issue 6: Incomplete Image Metadata

**Problem:**
The storage client doesn't populate all `stored_images` fields when creating records.

**Evidence:**

In `internal/services/storage/client.go` (lines 142-169):

```go
func (c *Client) UploadImageWithHash(ctx context.Context, bucket, path string, data []byte) (string, error) {
    // ...
    
    imageID := generateUUID()
    if err := c.CreateStoredImageRecord(ctx, imageID, hash, path); err != nil {
        return "", err
    }
    // ↑ Only sets: id, content_hash, storage_path
    // ↓ Missing: source_url, mime_type, width, height, file_size
}
```

**Impact:**
- `source_url` field (original Instagram URL) is empty
- Image dimensions unknown (can't optimize display)
- File size unknown (can't enforce quotas)

---

## Investigation Verification Steps

### Step 1: Confirm Image Processing Gap

**To verify the issue:**

1. Check if any code downloads images from scraped URLs:
   ```bash
   grep -r "http.Get\|Download\|FetchImage" --include="*.go" internal/
   ```

2. Check if `UploadImageWithHash` is called anywhere:
   ```bash
   grep -r "UploadImageWithHash" --include="*.go" internal/
   ```

3. Check if `CreateRecipeImage` is called:
   ```bash
   grep -r "CreateRecipeImage" --include="*.go" internal/
   ```

**Expected Result:** No matches found - confirming images are never processed.

---

### Step 2: Test Instagram Scraper Resilience

**To verify timeout issue:**

1. Check current HTTP client configuration:
   ```bash
   grep -A 5 "http.Client" internal/services/scraper/instagram.go
   ```

2. Compare with TikTok implementation:
   ```bash
   grep -A 5 "http.Client" internal/services/scraper/tiktok.go
   ```

**Expected Result:** Instagram has no timeout, TikTok has 180s timeout.

---

### Step 3: Verify Database State

**SQL queries to run:**

```sql
-- Check if any recipe images exist
SELECT COUNT(*) FROM recipe_images;

-- Check if any stored images exist  
SELECT COUNT(*) FROM stored_images;

-- Check recipes without thumbnails
SELECT COUNT(*) FROM recipes WHERE thumbnail_id IS NULL;

-- Check import job status distribution
SELECT status, COUNT(*) FROM recipe_import_jobs GROUP BY status;
```

**Expected Results:**
- `recipe_images`: 0 or very low count
- `stored_images`: 0 or very low count
- `recipes` with NULL `thumbnail_id`: High percentage
- Failed jobs: Elevated count for "Instagram" origin

---

### Step 4: Review Test Coverage

**To check what's tested:**

1. List all test files:
   ```bash
   find internal -name "*_test.go" -type f
   ```

2. Check if image processing is tested:
   ```bash
   grep -r "ImageURL\|Thumbnail\|UploadImage" --include="*_test.go" internal/
   ```

3. Check if Instagram scraper has tests:
   ```bash
   ls -la internal/services/scraper/*_test.go 2>/dev/null || echo "No scraper tests found"
   ```

**Expected Results:** Limited or no test coverage for image processing and scraping.

---

## Remediation Plan

### Phase 1: Fix Image Processing (Critical)

**Task 1.1: Implement Image Download in Worker**

Add image download logic to `HandleProcessRecipe`:

```go
// After scraping, download and store image
if post.ImageURL != "" {
    imageData, err := downloadImage(ctx, post.ImageURL)
    if err != nil {
        slog.Error("Failed to download image", "error", err)
        // Continue without image rather than failing entire job
    } else {
        publicURL, err := p.storage.UploadImageWithHash(ctx, "recipe-images", 
            fmt.Sprintf("%s/%s.jpg", recipeUUID, imageID), imageData)
        if err != nil {
            slog.Error("Failed to upload image", "error", err)
        } else {
            // Create stored_images record and link to recipe
            // Update recipe.thumbnail_id
        }
    }
}
```

**Task 1.2: Enhance UploadImageWithHash**

Modify to accept and store additional metadata:
- Source URL
- MIME type
- Dimensions (if available)
- File size

---

### Phase 2: Harden Instagram Scraper (Critical)

**Task 2.1: Add Timeout and Retry Logic**

```go
func NewInstagramScraper(proxyURL, proxyKey string) *InstagramScraper {
    return &InstagramScraper{
        proxyURL:   proxyURL,
        proxyKey:   proxyKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,  // Add timeout
        },
    }
}

// Add retry logic in Scrape method
for attempt := 0; attempt < 3; attempt++ {
    resp, err := s.httpClient.Do(req)
    if err == nil && resp.StatusCode == 200 {
        break
    }
    if resp != nil && resp.StatusCode == 429 {
        time.Sleep(time.Duration(attempt+1) * time.Second)
        continue
    }
    // ...
}
```

**Task 2.2: Document Proxy Requirements**

Add documentation about:
- What proxy service is expected
- How to obtain `doc_id` when it changes
- How to verify proxy health

---

### Phase 3: Fix Embeddings (Medium)

**Task 3.1: Persist Embeddings**

Add database column for embedding vector and update handler:

```go
embedding, err := p.openai.GenerateEmbedding(ctx, text)
if err != nil {
    return fmt.Errorf("failed to generate embedding: %w", err)
}

err = p.db.UpdateRecipeEmbedding(ctx, generated.UpdateRecipeEmbeddingParams{
    ID:        recipeUUID,
    Embedding: embedding,  // []float32 or pgvector.Vector
})
```

---

### Phase 4: Developer Experience (Low)

**Task 4.1: Add godotenv.Load()**

```go
// In both cmd/server/main.go and cmd/worker/main.go
import _ "github.com/joho/godotenv/autoload"
// Or explicitly: godotenv.Load()
```

**Task 4.2: Implement Cleanup Handler**

Add logic to remove old failed/completed jobs from `recipe_import_jobs` table.

---

## Files Requiring Changes

| File | Issue | Priority |
|------|-------|----------|
| `internal/worker/handlers.go` | Missing image download, embedding persistence | 🔴 Critical |
| `internal/services/storage/client.go` | Incomplete image metadata | 🔴 Critical |
| `internal/services/scraper/instagram.go` | No timeout, no retry | 🔴 Critical |
| `cmd/server/main.go` | Missing godotenv.Load() | 🟡 Medium |
| `cmd/worker/main.go` | Missing godotenv.Load() | 🟡 Medium |
| `internal/db/queries/*.sql` | Add embedding update query | 🟡 Medium |

---

## Testing Strategy

1. **Unit Tests:**
   - Mock Instagram/TikTok API responses
   - Test image download with test server
   - Test storage client with mock Supabase

2. **Integration Tests:**
   - End-to-end recipe import flow
   - Image upload and retrieval
   - Error handling (network failures, rate limits)

3. **Manual Verification:**
   - Import recipe from Instagram
   - Verify image appears in Supabase Storage
   - Verify `recipe_images` table populated
   - Verify recipe displays thumbnail

---

## Success Criteria

- [x] Instagram recipe imports include images
- [x] TikTok recipe imports include images  
- [x] Failed scrapes retry automatically
- [x] No hanging requests (all have timeouts)
- [x] Embeddings are persisted to database
- [x] `.env` file loads automatically in development
- [x] All new code has test coverage

---

## Questions for Clarification

1. **Image Priority:** Should we fail the entire import if image download fails, or continue with text-only recipe?

2. **Multiple Images:** Instagram posts can have multiple images - should we save all or just the first?

3. **Proxy Service:** What proxy service is being used? Is it a custom solution or third-party?

4. **Embedding Storage:** Should embeddings be stored in the `recipes` table or a separate table?

5. **Image Dimensions:** Do we need to process images to get dimensions, or trust metadata?

---

*Plan created: 2026-02-24*
*Next step: Review plan and begin implementation phase*

---

## Status: ✅ COMPLETE

Completed: 2026-02-27
