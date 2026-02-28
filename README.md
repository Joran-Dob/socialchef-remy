# SocialChef Remy

Remy is an AI-powered recipe extractor that scrapes social media posts from Instagram and TikTok, transcribes video content, and generates structured recipes.

## Workflow

The processing pipeline follows these steps:

1.  **Scrape**: Extracts raw post data (caption, images, video URL) from the social media source.
2.  **Content Validate**: Runs a quick heuristic check to verify if the content is likely a recipe before further processing.
3.  **Transcribe**: If a video is available, the audio is transcribed using OpenAI to capture spoken instructions and ingredients.
4.  **Generate**: Uses AI (Groq, Cerebras, or OpenAI) to synthesize the caption and transcript into a structured recipe format.
5.  **Output Validate**: Performs a quality check on the generated recipe to ensure accuracy and completeness.
6.  **Save**: Persists the validated recipe and associated media to the database and storage.

## Features

- **Video Transcription**: Leverages OpenAI `gpt-4o-mini-transcribe` to extract recipe details from video audio.
- **Multi-Provider AI**: Supports Groq, Cerebras, and OpenAI for recipe generation with automatic fallback.
- **Content Validation**: Multi-stage validation (heuristic length/keyword check + optional AI classification) to reject non-recipe posts.
- **Output Quality Check**: Automated scoring based on placeholder detection and minimum content requirements.
- **Structured Error Handling**: Categorized error types with specific codes for robust debugging and recovery.
- **Automatic Retry**: Built-in exponential backoff for transient failures during scraping or AI processing.

## Recipe Generation

Recipe generation supports multiple AI providers with automatic fallback:

### Providers
- **Groq** (default): Uses `llama-3.3-70b-versatile` model
- **Cerebras**: Uses `gpt-oss-120b` model
- **OpenAI**: Uses `gpt-4o` model

### Configuration
Configure in `config.yaml`:
```yaml
recipe_generation:
  provider: cerebras           # groq | cerebras | openai
  fallback_enabled: true
  fallback_provider: groq      # secondary provider if primary fails
```

### Fallback Behavior
When `fallback_enabled` is true, if the primary provider fails with a retryable error (rate limit, server error, credit exhaustion), the system automatically tries the fallback provider.

## Error Handling

Remy uses structured errors with categories and specific codes for better error management.

| Error Type | Status Code | Description |
| :--- | :--- | :--- |
| `VALIDATION_ERROR` | 400 | Input content or generated recipe failed validation. |
| `TRANSCRIPTION_ERROR` | 500 | Errors occurring during video audio transcription. |
| `SCRAPER_ERROR` | 500 | Failures when fetching data from social media platforms. |
| `RECIPE_GENERATION_ERROR` | 500 | Failures during AI recipe synthesis. |
| `RATE_LIMIT_ERROR` | 429 | Service provider rate limits reached (OpenAI, Groq, etc.). |
| `NOT_FOUND_ERROR` | 404 | The requested post or resource could not be found. |
| `INTERNAL_ERROR` | 500 | Unexpected internal application errors. |

### Common Error Codes

- `CONTENT_NOT_RECIPE`: Initial content failed validation (e.g., too short, no keywords).
- `LOW_QUALITY_RECIPE`: Generated recipe failed quality checks (e.g., too many placeholders).
- `VIDEO_FETCH_ERROR`: Failed to download video content for transcription.
- `OPENAI_API_ERROR`: Issue communicating with the transcription service.
- `PROVIDER_FALLBACK_FAILED`: Both primary and fallback providers failed.

## Validation

### Content Validation
Every post is checked before AI processing:
- **Heuristics**: Minimum character counts and presence of recipe-related keywords.
- **AI Check**: For ambiguous content, an LLM determines if a recipe is present.

### Output Validation
Generated recipes receive a quality score (0-100) based on:
- **Placeholder Detection**: Checks for generic terms like "N/A", "TBD", or "Not Specified".
- **Minimum Requirements**: Must have at least 2 ingredients and 2 detailed instructions.
- **Threshold**: Recipes with a score below 50 or exceeding 20% placeholders are rejected.

## Retry Configuration

Transient operations use an automatic retry mechanism with exponential backoff.

- **Max Attempts**: 3
- **Initial Delay**: 1s
- **Max Delay**: 5s
- **Backoff Factor**: 2.0
- **Jitter**: Enabled (up to 10%)

## Environment Variables

| Variable | Required | Description |
| :--- | :--- | :--- |
| `DATABASE_URL` | Yes | PostgreSQL connection string. |
| `REDIS_URL` | Yes | Redis connection for the task queue. |
| `OPENAI_API_KEY` | Yes | For transcription and embedding services. |
| `GROQ_API_KEY` | Yes | For AI recipe generation. |
| `CEREBRAS_API_KEY` | Yes | For Cerebras AI recipe generation (alternative to Groq). |
| `APIFY_API_KEY` | Yes | For TikTok scraping services. |
| `PROXY_SERVER_URL` | Yes | For Instagram scraping proxy. |
| `PROXY_API_KEY` | Yes | For Instagram scraping proxy authentication. |
| `SUPABASE_URL` | Yes | Supabase project URL for storage and auth. |
| `SUPABASE_SERVICE_ROLE_KEY` | Yes | Admin key for Supabase operations. |
