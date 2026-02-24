-- name: GetRecipe :one
SELECT * FROM recipes WHERE id = $1;

-- name: GetRecipesByUser :many
SELECT * FROM recipes WHERE created_by = $1 ORDER BY created_at DESC;

-- name: CreateRecipe :one
INSERT INTO recipes (
    id, created_by, recipe_name, description, prep_time, cooking_time, original_serving_size, difficulty_rating, origin, url, owner_id, thumbnail_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: UpdateRecipe :one
UPDATE recipes 
SET 
    recipe_name = $2, 
    description = $3, 
    prep_time = $4, 
    cooking_time = $5, 
    original_serving_size = $6, 
    difficulty_rating = $7, 
    origin = $8,
    url = $9,
    owner_id = $10,
    thumbnail_id = $11,
    updated_at = NOW()
WHERE id = $1 AND created_by = $12
RETURNING *;

-- name: DeleteRecipe :exec
DELETE FROM recipes WHERE id = $1 AND created_by = $2;

-- name: UpdateRecipeThumbnail :exec
UPDATE recipes SET thumbnail_id = $2, updated_at = NOW() WHERE id = $1;
