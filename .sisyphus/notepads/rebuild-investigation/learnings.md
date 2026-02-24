
## Image Processing Pipeline Implementation
- Implemented a complete image processing pipeline in the worker handler.
- Added `downloadImage` helper to fetch images from Instagram/TikTok URLs.
- Fixed `storage/client.go` to generate real UUIDs using `github.com/google/uuid` instead of a constant placeholder.
- Integrated image download, upload, and database linking in `HandleProcessRecipe`.
- Images are stored in the `recipes` bucket under `post_images/{hash}`.
- The recipe's `thumbnail_id` is updated with the `StoredImageID` after successful processing.
- Error handling ensures that image processing failures do not fail the entire recipe import job.

## Instagram Scraper Enhancements (Phase 2)
- Added 30s timeout to Instagram HTTP client to prevent hanging requests.
- Implemented retry logic with linear/exponential backoff (1s, 2s, 3s) for transient failures (429 Rate Limited).
- Added missing headers to the proxy request body to better mimic browser behavior: User-Agent, X-FB-LSD, X-ASBD-ID, Sec-Fetch-Site.
- Used `log/slog` for structured logging of failures and retries.
- Verified that headers must be placed inside the nested "headers" map of the proxy request body, as the scraper uses an external proxy service.


## Environment Management (Phase 4.1)
- Added `github.com/joho/godotenv/autoload` to `cmd/server/main.go` and `cmd/worker/main.go`.
- This ensures `.env` files are loaded automatically on startup without manual calls to `godotenv.Load()`.
- Autoload is imported at the top of the import block to ensure environment variables are available for subsequent package initializations.