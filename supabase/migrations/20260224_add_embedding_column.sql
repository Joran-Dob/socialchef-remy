-- Migration: Add embedding column to recipes table
-- This migration must be run on Supabase before the Go code will work
-- 
-- Run this in Supabase SQL Editor or via CLI:
-- supabase db push

-- Add embedding column for vector similarity search
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS embedding vector(1536);

-- Create index for fast vector similarity search
CREATE INDEX IF NOT EXISTS idx_recipes_embedding ON recipes 
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Optional: Create a function for similarity search
CREATE OR REPLACE FUNCTION search_recipes_by_embedding(
    query_embedding vector(1536),
    match_limit int DEFAULT 10
)
RETURNS TABLE (
    id uuid,
    recipe_name text,
    description text,
    similarity float
)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id,
        r.recipe_name,
        r.description,
        1 - (r.embedding <=> query_embedding) as similarity
    FROM recipes r
    WHERE r.embedding IS NOT NULL
    ORDER BY r.embedding <=> query_embedding
    LIMIT match_limit;
END;
$$;
