-- Enable pg_trgm extension for fuzzy text matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create GIN index for fast similarity search on recipe names
CREATE INDEX IF NOT EXISTS idx_recipes_name_trgm ON recipes
USING gin (recipe_name gin_trgm_ops);

-- Alternative: Add similarity search function
CREATE OR REPLACE FUNCTION search_recipes_by_similarity(
    query text,
    similarity_threshold float DEFAULT 0.3,
    match_limit int DEFAULT 10
)
RETURNS TABLE (
    id uuid,
    recipe_name text,
    similarity float
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT
        r.id,
        r.recipe_name,
        similarity(r.recipe_name, query) as similarity
    FROM recipes r
    WHERE r.recipe_name % query  -- trigram similarity operator
       OR similarity(r.recipe_name, query) > similarity_threshold
    ORDER BY similarity DESC
    LIMIT match_limit;
END;
$$;
