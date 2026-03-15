# SocialChef Remy API - Bruno Collection

This Bruno collection provides automated API testing for the SocialChef Remy recipe extraction service.

## Prerequisites

1. Install Bruno CLI:
   ```bash
   npm install -g @usebruno/cli
   ```

2. Copy environment template and fill in your values:
   ```bash
   cp ../.env.bruno.example .env.bruno
   ```
   
   Edit `.env.bruno` and add:
   - `SUPABASE_URL` - Your Supabase project URL
   - `SUPABASE_JWT_SECRET` - Your JWT secret for token generation

## Usage

### Run All Requests (Local)
```bash
bru run --env local
```

### Run All Requests (Production)
```bash
bru run --env fly
```

### Run Specific Folder
```bash
bru run --env local --folder "1-Recipe"
```

## Collection Structure

```
bruno/
├── 0-Health/           # Health check (no auth)
├── 1-Recipe/           # Recipe import endpoints
├── 2-Embedding/        # Embedding generation
├── 3-Search/           # Search endpoints
├── 4-Bulk-Import/      # Bulk import endpoints
├── environments/
│   ├── local.bru      # Local development
│   └── fly.bru        # Production (fly.io)
└── collection.bru     # JWT auth script
```

## Authentication

This collection uses **automatic JWT token generation**:

- Tokens are generated using `SUPABASE_JWT_SECRET` from environment
- Tokens cached for 1 hour (with 1-minute safety buffer)
- Tokens automatically refreshed when expired
- 401 responses trigger token regeneration

No manual token management required!

## Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/health` | GET | No | Health check |
| `/api/recipe` | POST | Yes | Import recipe from URL |
| `/api/recipe-status` | GET | Yes | Check import job status |
| `/api/recipes/{recipeID}/steps` | GET | Yes | Get recipe steps with ingredients |
| `/api/instruction-ingredients-count` | GET | Yes | Get count of instruction-ingredient linkages for a recipe |
| `/api/user-import-status` | GET | Yes | List user import jobs |
| `/api/generate-embedding` | POST | Yes | Queue embedding generation |
| `/api/v1/search` | POST | Yes | Hybrid search (delegates to semantic) |
| `/api/v1/search/semantic` | POST | Yes | Semantic/vector search |
| `/api/v1/search/by-name` | POST | Yes | Text search on recipe names |
| `/api/bulk-import` | POST | Yes | Bulk import recipes from URLs |
| `/api/bulk-import/{id}` | GET | Yes | Get bulk import job status |
| `/api/bulk-imports` | GET | Yes | List user's bulk import jobs |
| `/api/bulk-import/{id}` | DELETE | Yes | Cancel pending bulk import |

## Troubleshooting

### "Missing required environment variables"
Create `.env.bruno` file from template with your Supabase credentials.

### 401 Unauthorized
Token may be invalid. The collection will auto-refresh on next request.

### 404 on search endpoints
Make sure the server is running and the routes are registered in `cmd/server/main.go`.

## Notes

- Health check is public (no JWT required)
- All other endpoints require valid JWT token
- Search endpoints are now fully functional with category arrays populated
- Recipe steps endpoint may include `parts` array for recipes with distinct components (e.g., main dish + sauce)

## Instruction-Ingredient Linking Tests

The collection includes comprehensive tests that verify instruction-ingredient junction entries are created correctly during recipe import:

### Test Files
- `1-Recipe/import-recipe-verify-ingredients.bru` - Tests single recipe import with database verification
- `4-Bulk-Import/bulk-import-verify-ingredients.bru` - Tests bulk recipe import with database verification

### How It Works
1. Import a recipe (single or bulk)
2. Poll for job completion
3. Query the `/api/instruction-ingredients-count` endpoint with the recipe_id
4. Assert that the count is greater than 0, confirming that the instruction_ingredients junction table has been populated

### Important Notes
These tests use async operations for polling job completion. **Recommended:** Use Bruno Desktop App for these tests as it handles async operations better than the CLI.

### Endpoint
```
GET /api/instruction-ingredients-count?recipe_id={recipe_id}
```

Response:
```json
{
  "count": 15
}
```

## Split Recipe Tests

The collection includes tests for recipes with parts - recipes that have distinct components like "Main Dish + Sauce" or multiple courses:

### Test Files
- `1-Recipe/import-recipe.bru` - Import a recipe (use a complex recipe URL to test parts)
- `1-Recipe/get-recipe-status.bru` - Poll for import completion
- `1-Recipe/verify-split-recipe-parts.bru` - Verifies the recipe contains properly structured parts array

### How It Works
1. Import a complex recipe from a URL (e.g., butter chicken with sauce components)
2. Poll for job completion
3. Query the `/api/recipes/{recipeID}/steps` endpoint with the recipe_id
4. Assert that the response contains a `parts` array with the correct structure

### Parts Structure

When a recipe has parts, the steps response includes:

```json
{
  "recipe_id": "uuid",
  "total_steps": 15,
  "steps": [...],
  "parts": [
    {
      "name": "Chicken",
      "display_order": 1,
      "ingredients": [...],
      "instructions": [
        {
          "step_number": 1,
          "instruction_rich": "...",
          "timers": []
        }
      ],
      "prep_time": null,
      "cooking_time": null,
      "is_optional": false
    },
    {
      "name": "Sauce",
      "display_order": 2,
      "ingredients": [...],
      "instructions": [...],
      "prep_time": null,
      "cooking_time": null,
      "is_optional": false
    }
  ]
}
```

### Important Notes
- Each part's instructions start from `step_number: 1`
- Parts are ordered by `display_order` field
- `is_optional` indicates if the part can be omitted
- Not all recipes have parts - the field is optional


## Known Issues

### Async JWT Generation in CLI
**Problem:** Bruno CLI doesn't wait for async crypto operations in pre-request scripts. The request is sent before the JWT is generated.

**Solution:** Add a pre-generated JWT token to your environment:

1. Generate a JWT using your `SUPABASE_JWT_SECRET`:
   ```bash
   # Use a JWT generator or https://jwt.io
   # Payload should be:
   # {
   #   "sub": "test-user-uuid-1234-5678-9012-345678901234",
   #   "iss": "YOUR_SUPABASE_URL/auth/v1",
   #   "exp": 1234567890
   # }
   ```

2. Add the token to `bruno/environments/local.bru`:
   ```
   vars {
     baseUrl: http://localhost:8080
     testUserId: test-user-uuid-1234-5678-9012-345678901234
     jwtToken: eyJhbGciOiJIUzI1NiIs...
   }
   ```

**Alternative:** Use Bruno Desktop App, which handles async operations better than CLI.
