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
- **Split Recipes**: Complex recipes with multiple components (like "Main Dish + Sauce") are automatically split into parts, each with their own ingredients and instructions.

## Recipe Generation

Recipe generation supports multiple AI providers with automatic fallback:

### Providers
- **Groq** (default): Uses `openai/gpt-oss-120b` model
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

### Feature Support Matrix

All providers support the core recipe generation features:

| Feature | Groq | Cerebras | OpenAI | Description |
| :--- | :--- | :--- | :--- | :--- |
| **Recipe Generation** | ✅ | ✅ | ✅ | Generate structured recipes from captions and transcripts |
| **Category Generation** | ✅ | ✅ | ❌ | AI-powered category suggestions (cuisine, meal type, dietary restrictions) |
| **Rich Instructions** | ✅ | ✅ | ❌ | Enhanced instructions with ingredient/timer placeholders |

**Note**: When using Cerebras or OpenAI as primary providers, enable `fallback_enabled` to ensure category and rich instruction features work via fallback to Groq.

### Fallback Behavior

When `fallback_enabled` is true:

1. **Retryable Errors**: If the primary provider fails with a rate limit, server error, or credit exhaustion, the system automatically tries the fallback provider.
2. **Capability Fallback**: If the primary provider doesn't support a feature (e.g., Cerebras for categories), the system automatically uses the fallback provider for that feature.
3. **Graceful Degradation**: If both providers fail for optional features (categories, rich instructions), the system returns empty results rather than failing the entire recipe generation.

## Split Recipes

Remy automatically handles complex recipes with multiple components (like "Chicken + Sauce" or "Cake + Frosting"). When AI detects distinct parts, the recipe is structured accordingly.

### Parts Structure

Each part contains its own ingredients and instructions:

```json
{
  "recipe_id": "uuid",
  "total_steps": 15,
  "has_parts": true,
  "parts": [
    {
      "part_id": "uuid",
      "part_name": "Chicken",
      "display_order": 1,
      "is_optional": false,
      "steps": [
        {
          "step_number": 1,
          "instruction": "Marinate chicken in yogurt and spices for 30 minutes",
          "instruction_rich": "Marinate chicken in yogurt and spices for {{timer:0}}",
          "ingredients": [...],
          "timers": [...]
        }
      ]
    },
    {
      "part_id": "uuid",
      "part_name": "Sauce",
      "display_order": 2,
      "is_optional": false,
      "steps": [
        {
          "step_number": 1,
          "instruction": "Saute onions in butter until golden",
          "instruction_rich": "Saute onions in butter until {{timer:0}}",
          "ingredients": [...],
          "timers": [...]
        }
      ]
    }
  ]
}
```

### API Response Examples

**Recipe with parts (from `/api/recipes/{id}/steps`):**

```json
{
  "recipe_id": "550e8400-e29b-41d4-a716-446655440000",
  "total_steps": 12,
  "has_parts": true,
  "parts": [
    {
      "part_id": "660e8400-e29b-41d4-a716-446655440001",
      "part_name": "Blueberry Muffins",
      "is_optional": false,
      "display_order": 1,
      "steps": [
        {
          "step_number": 1,
          "instruction": "Preheat oven to 375°F (190°C). Line muffin tin with paper liners.",
          "instruction_rich": "Preheat oven to 375°F (190°C). Line muffin tin with paper liners.",
          "ingredients": [],
          "timers": []
        },
        {
          "step_number": 2,
          "instruction": "In a large bowl, whisk together 2 cups flour, 1/2 cup sugar, 2 tsp baking powder, and 1/2 tsp salt.",
          "instruction_rich": "In a large bowl, whisk together {{ingredient:770e8400-e29b-41d4-a716-446655440002}}, {{ingredient:...}}, 2 tsp baking powder, and 1/2 tsp salt.",
          "ingredients": [
            {
              "id": "770e8400-e29b-41d4-a716-446655440002",
              "name": "all-purpose flour",
              "step_quantity": "2 cups",
              "total_quantity": "2 cups",
              "unit": "cups"
            }
          ],
          "timers": []
        }
      ]
    },
    {
      "part_id": "660e8400-e29b-41d4-a716-446655440003",
      "part_name": "Streusel Topping",
      "is_optional": true,
      "display_order": 2,
      "steps": [
        {
          "step_number": 1,
          "instruction": "Mix 1/4 cup flour, 2 tbsp sugar, and 2 tbsp cold butter until crumbly.",
          "instruction_rich": "Mix 1/4 cup {{ingredient:...}}, 2 tbsp sugar, and 2 tbsp cold butter until crumbly.",
          "ingredients": [...],
          "timers": []
        }
      ]
    }
  ]
}
```

**Recipe without parts (flat structure):**

```json
{
  "recipe_id": "550e8400-e29b-41d4-a716-446655440000",
  "total_steps": 8,
  "has_parts": false,
  "steps": [
    {
      "step_number": 1,
      "instruction": "Heat olive oil in a large skillet over medium heat.",
      "instruction_rich": "Heat {{ingredient:...}} in a large skillet over medium heat.",
      "ingredients": [...],
      "timers": []
    }
  ]
}
```

### Key Points

- Each part's instructions start from `step_number: 1`
- Parts are ordered by the `display_order` field
- `is_optional` indicates parts that can be omitted (like toppings or sauces)
- Ingredients are scoped to each part but linked to the overall recipe
- Not all recipes have parts. The field is optional.

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
| `YOUTUBE_API_KEY` | Yes | For YouTube video scraping. See [setup guide](docs/youtube-api-setup.md). |
