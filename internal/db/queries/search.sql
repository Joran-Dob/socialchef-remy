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
    r.updated_at,
    COALESCE(similarity(r.recipe_name, $1), 0) as name_similarity
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE 
    r.recipe_name ILIKE '%' || $1 || '%'
    OR r.recipe_name % $1
    OR similarity(r.recipe_name, $1) > 0.3
GROUP BY
    r.id, r.recipe_name, r.description, r.prep_time, r.cooking_time,
    r.total_time, r.original_serving_size, r.difficulty_rating, r.focused_diet,
    r.estimated_calories, r.origin, r.url, r.language, r.created_by,
    r.owner_id, r.thumbnail_id, r.created_at, r.updated_at
ORDER BY 
    similarity(r.recipe_name, $1) DESC,
    r.created_at DESC
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
    r.thumbnail_id,
    r.owner_id,
    p.username as owner_username,
    si.storage_path as thumbnail_storage_path,
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
    CAST(COALESCE(ts_rank(r.search_vector, plainto_tsquery('english', $3)), 0) as float8) as text_similarity,
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
LEFT JOIN profiles p ON r.owner_id = p.id
LEFT JOIN stored_images si ON r.thumbnail_id = si.id
WHERE r.embedding IS NOT NULL
GROUP BY r.id, r.recipe_name, r.description, r.thumbnail_id, r.owner_id, p.username, si.storage_path, r.embedding, r.search_vector
ORDER BY hybrid_score DESC
LIMIT $1;
-- name: SearchRecipesHybridWithFilters :many
SELECT
    r.id,
    r.recipe_name,
    r.description,
    COALESCE(array_agg(DISTINCT cc.name) FILTER (WHERE cc.name IS NOT NULL), ARRAY[]::text[]) as cuisine_categories,
    COALESCE(array_agg(DISTINCT mt.name) FILTER (WHERE mt.name IS NOT NULL), ARRAY[]::text[]) as meal_types,
    CAST(1 - (r.embedding <=> $2::vector) AS float8) as vector_similarity,
    COALESCE(ts_rank(r.search_vector, plainto_tsquery('english', $3)), 0) as text_rank,
    CAST(0.7 * CAST(1 - (r.embedding <=> $2::vector) AS float8) + 0.3 * COALESCE(ts_rank(r.search_vector, plainto_tsquery('english', $3)), 0) AS float8) as hybrid_score
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE r.embedding IS NOT NULL
  AND ($4::text[] IS NULL OR cc.name = ANY($4))
  AND ($5::text[] IS NULL OR mt.name = ANY($5))
  AND ($6 = 0 OR r.total_time <= $6)
GROUP BY r.id, r.recipe_name, r.description, r.embedding, r.search_vector
ORDER BY hybrid_score DESC
LIMIT $1;

-- name: SearchRecipesByIngredient :many
SELECT
    r.id,
    r.recipe_name,
    r.description,
    COALESCE(array_agg(DISTINCT cc.name) FILTER (WHERE cc.name IS NOT NULL), ARRAY[]::text[]) as cuisine_categories,
    COALESCE(array_agg(DISTINCT mt.name) FILTER (WHERE mt.name IS NOT NULL), ARRAY[]::text[]) as meal_types,
    r.ingredient_names
FROM recipes r
LEFT JOIN recipe_cuisine_categories rcc ON r.id = rcc.recipe_id
LEFT JOIN cuisine_categories cc ON rcc.cuisine_category_id = cc.id
LEFT JOIN recipe_meal_types rmt ON r.id = rmt.recipe_id
LEFT JOIN meal_types mt ON rmt.meal_type_id = mt.id
WHERE $1 = ANY(r.ingredient_names)
GROUP BY r.id, r.recipe_name, r.description, r.ingredient_names
ORDER BY r.created_at DESC
LIMIT $2;
