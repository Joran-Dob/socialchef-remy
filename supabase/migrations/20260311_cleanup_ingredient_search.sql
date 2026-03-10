-- Migration: Fix ingredient_names setup (idempotent cleanup)
-- This migration fixes any incomplete or broken previous migrations

-- 1. Ensure ingredient_names column exists
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS ingredient_names text[];

-- 2. Create/Update function to extract ingredient names from recipe_ingredients table
CREATE OR REPLACE FUNCTION extract_ingredient_names(p_recipe_id UUID)
RETURNS text[]
LANGUAGE plpgsql
AS $$
DECLARE
    names text[];
BEGIN
    SELECT array_agg(DISTINCT name)
    INTO names
    FROM recipe_ingredients
    WHERE recipe_id = p_recipe_id;
    RETURN names;
END;
$$;

-- 3. Populate ingredient_names from recipe_ingredients (only for empty rows)
UPDATE recipes r
SET ingredient_names = extract_ingredient_names(r.id)
WHERE ingredient_names IS NULL 
   OR array_length(ingredient_names, 1) IS NULL
   AND EXISTS (
    SELECT 1 FROM recipe_ingredients ri WHERE ri.recipe_id = r.id
);

-- 4. Ensure search_vector includes ingredients
UPDATE recipes
SET search_vector =
    setweight(to_tsvector('english', COALESCE(recipe_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(array_to_string(ingredient_names, ' '), '')), 'C')
WHERE search_vector IS NULL 
   OR search_vector = '';

-- 5. Create index if not exists (for ingredient search)
CREATE INDEX IF NOT EXISTS idx_recipes_ingredients ON recipes USING gin(ingredient_names);
CREATE INDEX IF NOT EXISTS recipe_search_idx ON recipes USING GiST (search_vector);

-- 6. Mark any conflicting old migrations as applied to prevent errors
-- This prevents the duplicate key errors from earlier broken migrations
INSERT INTO supabase_migrations.schema_migrations (version, name, statements, created_at)
VALUES 
  ('20260310201500', 'add_ingredient_names_fixed', ARRAY['-- superseded by cleanup migration'], NOW()),
  ('20260310203000', 'add_ingredient_names_column_fixed', ARRAY['-- superseded by cleanup migration'], NOW())
ON CONFLICT (version) DO UPDATE SET name = EXCLUDED.name;
