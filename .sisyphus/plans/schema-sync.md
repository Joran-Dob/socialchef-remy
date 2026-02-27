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

- [x] 1. **Dump schema from Supabase production database**
- [x] 2. **Create sync script for future use**
### Wave 2: Update SQL Queries

- [x] 3. **Update all sqlc queries to match real schema**
- [x] 4. **Regenerate sqlc code**
### Wave 3: Update Go Code

- [x] 5. **Update worker handlers to use new schema**
- [x] 6. **Update API handlers if needed**
### Wave 4: Verification

- [x] 7. **Run E2E verification**
- [x] 8. **Document sync process**
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

- [x] `make sync-schema` dumps production schema
- [x] `sqlc generate` succeeds with real schema
- [x] `go build ./...` succeeds
- [x] `go test ./...` passes
- [x] E2E test: recipe import completes and saves to database

---

## Status: ✅ COMPLETE

Completed: 2026-02-27
