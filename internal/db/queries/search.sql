-- name: SearchRecipesByName :many
SELECT r.*
FROM recipes r
WHERE r.recipe_name ILIKE '%' || $1 || '%'
ORDER BY r.created_at DESC
LIMIT $2;

-- Note: SearchRecipesHybrid uses database function that sqlc can't introspect.
-- Call it directly from Go using raw SQL or create a simpler wrapper.
-- For now, use SearchRecipesByEmbedding for vector search.

-- name: SearchRecipesByEmbedding :many
SELECT 
    id,
    recipe_name,
    description,
    CAST(1 - (embedding <=> $2::vector) AS float8) as similarity
FROM recipes
WHERE embedding IS NOT NULL
ORDER BY embedding <=> $2::vector
LIMIT $1;

-- name: GetRecipesWithoutEmbeddings :many
SELECT id, recipe_name, description 
FROM recipes 
WHERE embedding IS NULL 
LIMIT $1;


-- name: SearchRecipesHybrid :many
SELECT * FROM search_recipes($1, $2, $3, $4, $5);