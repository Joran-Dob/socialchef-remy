-- name: SearchRecipesByName :many
SELECT
    r.id,
    r.recipe_name,
    r.description,
    r.prep_time,
    r.cooking_time,
    r.total_time,
    r.original_serving_size,
    r.difficulty_rating,
    r.focused_diet,
    r.estimated_calories,
    r.origin,
    r.url,
    r.language,
    r.created_by,
    r.owner_id,
    r.thumbnail_id,
    COALESCE(
        array_agg(DISTINCT cc.name) FILTER (WHERE cc.name IS NOT NULL),
        ARRAY[]::text[]
    ) as cuisine_categories,
    COALESCE(
        array_agg(DISTINCT mt.name) FILTER (WHERE mt.name IS NOT NULL),
        ARRAY[]::text[]
    ) as meal_types,
    r.created_at,
    r.updated_at
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE r.recipe_name ILIKE '%' || $1 || '%'
GROUP BY
    r.id, r.recipe_name, r.description, r.prep_time, r.cooking_time,
    r.total_time, r.original_serving_size, r.difficulty_rating, r.focused_diet,
    r.estimated_calories, r.origin, r.url, r.language, r.created_by,
    r.owner_id, r.thumbnail_id, r.created_at, r.updated_at
ORDER BY r.created_at DESC
LIMIT $2;

-- Note: SearchRecipesHybrid uses database function that sqlc can't introspect.
-- Call it directly from Go using raw SQL or create a simpler wrapper.
-- For now, use SearchRecipesByEmbedding for vector search.

-- name: SearchRecipesByEmbedding :many
SELECT
    r.id,
    r.recipe_name,
    r.description,
    COALESCE(
        array_agg(DISTINCT cc.name) FILTER (WHERE cc.name IS NOT NULL),
        ARRAY[]::text[]
    ) as cuisine_categories,
    COALESCE(
        array_agg(DISTINCT mt.name) FILTER (WHERE mt.name IS NOT NULL),
        ARRAY[]::text[]
    ) as meal_types,
    CAST(1 - (r.embedding <=> $2::vector) AS float8) as similarity
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE r.embedding IS NOT NULL
GROUP BY r.id, r.recipe_name, r.description, r.embedding
ORDER BY r.embedding <=> $2::vector
LIMIT $1;

-- name: GetRecipesWithoutEmbeddings :many
SELECT id, recipe_name, description 
FROM recipes 
WHERE embedding IS NULL 
LIMIT $1;


-- name: SearchRecipesHybrid :many
SELECT
    r.id,
    r.recipe_name,
    r.description,
    COALESCE(
        array_agg(DISTINCT cc.name) FILTER (WHERE cc.name IS NOT NULL),
        ARRAY[]::text[]
    ) as cuisine_categories,
    COALESCE(
        array_agg(DISTINCT mt.name) FILTER (WHERE mt.name IS NOT NULL),
        ARRAY[]::text[]
    ) as meal_types,
    -- Vector similarity score (0-1)
    CAST(1 - (r.embedding <=> $2::vector) AS float8) as vector_similarity,
    -- Text search score (0-1)
    COALESCE(ts_rank(r.search_vector, plainto_tsquery('english', $3)), 0) as text_rank,
    -- Combined hybrid score
    CAST(
        0.7 * CAST(1 - (r.embedding <=> $2::vector) AS float8) +
        0.3 * COALESCE(ts_rank(r.search_vector, plainto_tsquery('english', $3)), 0)
        AS float8
    ) as hybrid_score
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE r.embedding IS NOT NULL
GROUP BY r.id, r.recipe_name, r.description, r.embedding, r.search_vector
ORDER BY hybrid_score DESC
LIMIT $1;