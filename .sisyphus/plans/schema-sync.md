# Schema Sync Plan: Supabase ↔ Go Project

## Problem

The Go project's `internal/db/queries/schema.sql` doesn't match the actual Supabase database schema:
- Column names differ (`name` vs `recipe_name`, `cook_time` vs `cooking_time`)
- Missing columns (`cuisine_categories`, `meal_type`, `occasion`, etc.)
- Extra columns that don't exist (`created_by`, `is_public`, `embedding`)

This causes runtime errors like:
```
ERROR: column "name" of relation "recipes" does not exist
```

## Solution: Dump Schema from Production

The source of truth is the **actual Supabase production database**, not the TypeScript migrations. We'll dump the schema directly from production.

---

## TODOs

### Wave 1: Extract Real Schema

- [ ] 1. **Dump schema from Supabase production database**
  
  **What to do**:
  - Use `pg_dump` with the Supabase connection string to extract only the schema (no data)
  - Focus on tables the Go backend needs: `recipes`, `recipe_ingredients`, `recipe_instructions`, `recipe_nutrition`, `recipe_import_jobs`, `social_media_owners`, `profiles`
  - Output to `internal/db/queries/schema.sql`
  
  **Command**:
  ```bash
  pg_dump "$DATABASE_URL" \
    --schema=public \
    --no-owner \
    --no-privileges \
    --schema-only \
    --table=recipes \
    --table=recipe_ingredients \
    --table=recipe_instructions \
    --table=recipe_nutrition \
    --table=recipe_import_jobs \
    --table=social_media_owners \
    --table=profiles \
    > internal/db/queries/schema.sql
  ```
  
  **Acceptance Criteria**:
  - [ ] `schema.sql` contains exact column definitions from production
  - [ ] All tables used by Go backend are included

- [ ] 2. **Create sync script for future use**
  
  **What to do**:
  - Create `scripts/sync-schema.sh` that wraps the pg_dump command
  - Make it idempotent and safe to run anytime
  - Add to Makefile as `make sync-schema`
  
  **Acceptance Criteria**:
  - [ ] `./scripts/sync-schema.sh` dumps schema to correct location
  - [ ] `make sync-schema` works

### Wave 2: Update SQL Queries

- [ ] 3. **Update all sqlc queries to match real schema**
  
  **What to do**:
  - Review each `.sql` file in `internal/db/queries/`
  - Fix column names to match production schema:
    - `name` → `recipe_name`
    - `cook_time` → `cooking_time`
    - `servings` → `original_serving_size`
    - `difficulty` → `difficulty_rating`
  - Remove queries for columns that don't exist
  - Add queries for new columns if needed
  
  **Files to update**:
  - `recipes.sql`
  - `ingredients.sql`
  - `instructions.sql`
  - `nutrition.sql`
  
  **Acceptance Criteria**:
  - [ ] All queries use correct column names
  - [ ] `sqlc generate` succeeds

- [ ] 4. **Regenerate sqlc code**
  
  **What to do**:
  - Run `sqlc generate` (via Docker)
  - Verify generated Go types match production schema
  
  **Acceptance Criteria**:
  - [ ] `sqlc generate` succeeds
  - [ ] Generated structs have correct field names

### Wave 3: Update Go Code

- [ ] 5. **Update worker handlers to use new schema**
  
  **What to do**:
  - Fix `internal/worker/handlers.go` to use correct field names
  - Update `CreateRecipeParams` usage
  - Fix ingredient, instruction, nutrition saves
  
  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] All field references updated

- [ ] 6. **Update API handlers if needed**
  
  **What to do**:
  - Check if API response shapes need updates
  - Update any direct field accesses
  
  **Acceptance Criteria**:
  - [ ] `go build ./...` succeeds
  - [ ] All tests pass

### Wave 4: Verification

- [ ] 7. **Run E2E verification**
  
  **What to do**:
  - Deploy to Fly.io
  - Run `make verify-e2e`
  - Verify recipe saving works
  
  **Acceptance Criteria**:
  - [ ] Recipe import job completes successfully
  - [ ] Recipe saved to database with correct columns

- [ ] 8. **Document sync process**
  
  **What to do**:
  - Add section to `docs/deployment.md` explaining schema sync
  - Document when to run `make sync-schema`
  
  **Acceptance Criteria**:
  - [ ] Documentation explains how to sync schema
  - [ ] Clear guidance on when sync is needed

---

## Long-term Sync Strategy

### When Schema Changes

1. **Schema change in Supabase** (via migration):
   ```bash
   # After deploying migration to Supabase
   make sync-schema
   sqlc generate
   # Fix any Go code that breaks
   go build ./...
   go test ./...
   ```

2. **Add Makefile target**:
   ```makefile
   .PHONY: sync-schema
   sync-schema:
       ./scripts/sync-schema.sh
       docker run --rm -v $(PWD):/src -w /src sqlc/sqlc generate
   ```

### Preventing Drift

- **Never manually edit** `internal/db/queries/schema.sql` - always sync from production
- **Source of truth**: Production Supabase database
- **Sync frequency**: After any Supabase migration that touches tables used by Go backend

---

## Tables Used by Go Backend

| Table | Purpose |
|-------|---------|
| `profiles` | User profiles (RLS user ID) |
| `recipes` | Main recipe data |
| `recipe_ingredients` | Ingredient list |
| `recipe_instructions` | Step-by-step instructions |
| `recipe_nutrition` | Nutritional info |
| `recipe_import_jobs` | Async job tracking |
| `social_media_owners` | Instagram/TikTok creators |

## Tables NOT Used by Go Backend

These exist in Supabase but the Go backend doesn't touch them:
- `content_*` tables (CMS)
- `recipe_categories`, `user_recipe_views`
- `feedback_*` tables

---

## Success Criteria

- [ ] `make sync-schema` dumps production schema
- [ ] `sqlc generate` succeeds with real schema
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] E2E test: recipe import completes and saves to database
