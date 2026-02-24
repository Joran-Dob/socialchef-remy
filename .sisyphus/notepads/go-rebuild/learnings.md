

## Integration Tests Complete (2026-02-24)

### Files Created
- `internal/integration/mocks_test.go` - Mock implementations for database and services
- `internal/integration/api_auth_test.go` - API endpoint and auth flow tests (676 lines)
- `internal/integration/worker_test.go` - Worker handler and pipeline tests (615 lines)

### Test Coverage
1. **API Endpoints (13 tests)**
   - POST /api/recipe - Missing auth, invalid body, empty URL
   - GET /api/recipe-status - Missing job ID, no auth
   - GET /api/user-import-status - With/without auth, user isolation
   - POST /api/generate-embedding - Missing recipe ID, invalid body, valid request

2. **Auth Flow (12 tests)**
   - Valid token acceptance
   - Invalid/expired token rejection
   - Missing Authorization header
   - Invalid Authorization format
   - Invalid signature
   - Invalid issuer
   - Missing sub claim
   - GetUserID from context
   - RequireAuth middleware
   - End-to-end auth flow

3. **Worker Handlers (23 tests)**
   - Invalid payload handling
   - Database mock operations (create/get/update jobs, recipes, ingredients, instructions, nutrition)
   - Recipe embedding updates
   - Scraper URL detection (Instagram/TikTok)
   - Asynq task creation
   - Progress broadcaster
   - Full recipe pipeline test

### Key Patterns
- Used table-driven tests for multiple test cases
- Mocked database with in-memory maps
- Mocked external services (no real API calls)
- Used httptest for HTTP endpoint testing
- JWT token helpers for auth testing

### Makefile Target
```makefile
test-integration:
	go test -v ./internal/integration/...
```

### Test Results
All 48 tests pass successfully:
```
ok  	github.com/socialchef/remy/internal/integration	0.842s
```

### Notes
- Tests do not require a real database connection (uses mocks)
- Tests do not make real API calls to external services (uses mocks)
- All tests are self-contained and can run in parallel

## Schema Migration Complete (2026-02-24)

### Summary
Successfully updated all SQL queries and Go code to match the real Supabase schema.

### Column Name Mappings Applied
- `recipes.name` → `recipes.recipe_name`
- `recipes.cook_time` → `recipes.cooking_time`
- `recipes.servings` → `recipes.original_serving_size`
- `recipes.difficulty` (TEXT) → `recipes.difficulty_rating` (SMALLINT)
- Removed: `origin_url`, `embedding`, `is_public` (don't exist in real schema)
- Added: `origin`, `url`, `owner_id`, `thumbnail_id` to recipes

### SQL Files Updated
1. `internal/db/queries/recipes.sql` - Fixed column names, removed embedding/embedding update
2. `internal/db/queries/ingredients.sql` - Added `original_quantity`, `original_unit` columns
3. `internal/db/queries/instructions.sql` - No changes needed (created_at is auto-generated)
4. `internal/db/queries/social_owners.sql` - Complete rewrite: uses `origin_id`, `platform`, `username`, `profile_pic_stored_image_id`
5. `internal/db/queries/profiles.sql` - Fixed `measurement_units` → `measurement_unit`
6. `internal/db/queries/images.sql` - Fixed `image_id` → `stored_image_id`, added `image_type`
7. `internal/db/queries/nutrition.sql` - Removed `calories` column
8. `internal/db/queries/search.sql` - Removed embedding-based search (embedding column doesn't exist)

### Go Files Updated
1. `internal/worker/handlers.go` - Updated CreateRecipe call with new field names and types
2. `internal/integration/mocks_test.go` - Updated mock to match new schema
3. `internal/integration/worker_test.go` - Updated all test data to use new field names
4. `internal/integration/api_auth_test.go` - Fixed `Name` → `RecipeName`

### Key Learnings
- sqlc generates Go types based on SQL queries, not just the schema
- When removing columns from queries, the generated Go types automatically exclude them
- Enum types in PostgreSQL become custom Go types (e.g., `RecipeOrigin`, `SocialMediaPlatform`)
- The `UpdateImportJobStatusParams` uses `JobID string` not `ID pgtype.UUID`
- `Error` field in `UpdateImportJobStatusParams` is `[]byte` not `pgtype.Text`

### Verification Commands
```bash
# Generate Go code from SQL
docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate

# Build and test
go build ./...
go test ./...
```

All tests pass successfully.
