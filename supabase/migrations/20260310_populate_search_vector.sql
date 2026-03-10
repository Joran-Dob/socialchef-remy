-- Migration: Populate search_vector column for full-text search
-- This enables the hybrid search feature to use text ranking

UPDATE recipes 
SET search_vector = 
    setweight(to_tsvector('english', COALESCE(recipe_name, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(array_to_string(ingredient_names, ' '), '')), 'C');

-- Create index for fast full-text search (if not exists)
CREATE INDEX IF NOT EXISTS recipe_search_idx ON recipes USING GiST (search_vector);

-- Verify population
SELECT 
    COUNT(*) as total_recipes,
    COUNT(search_vector) as with_search_vector,
    COUNT(*) - COUNT(search_vector) as without_search_vector
FROM recipes;
