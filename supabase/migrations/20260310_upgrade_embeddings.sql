-- Upgrade embedding model to text-embedding-3-large (3072 dimensions)
-- This is a breaking change - all embeddings need regeneration
--
-- Migration strategy:
-- 1. Add new column with 3072 dimensions
-- 2. Create index for the new column
-- 3. Backfill existing recipes with new embeddings (separate job)
-- 4. Once verified, drop old embedding column in future migration

-- Add new embedding column with 3072 dimensions for text-embedding-3-large
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS embedding_v3 vector(3072);

-- Create index for new embedding column using ivfflat
-- Note: ivfflat requires some data to be present for training
-- Index will be created after initial backfill
-- CREATE INDEX CONCURRENTLY idx_recipes_embedding_v3 ON recipes
-- USING ivfflat (embedding_v3 vector_cosine_ops)
-- WITH (lists = 100);

-- Add comment documenting the migration
COMMENT ON COLUMN recipes.embedding_v3 IS 'Embedding vector using text-embedding-3-large (3072 dimensions) - replaces old ada-002 embedding';

-- Note: Existing embeddings in 'embedding' column (1536 dimensions) are incompatible
-- A backfill job is required to regenerate all embeddings using the new model
-- Once migration is complete and verified, a future migration will drop the old column
