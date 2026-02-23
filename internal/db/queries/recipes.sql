-- name: GetRecipe :one
SELECT * FROM recipes WHERE id = $1;

-- name: GetRecipesByUser :many
SELECT * FROM recipes WHERE created_by = $1 ORDER BY created_at DESC;

-- name: CreateRecipe :one
INSERT INTO recipes (
    id, created_by, name, description, prep_time, cook_time, servings, difficulty, origin_url, embedding, is_public
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateRecipe :one
UPDATE recipes 
SET 
    name = $2, 
    description = $3, 
    prep_time = $4, 
    cook_time = $5, 
    servings = $6, 
    difficulty = $7, 
    origin_url = $8, 
    embedding = $9, 
    is_public = $10,
    updated_at = NOW()
WHERE id = $1 AND created_by = $11
RETURNING *;

-- name: DeleteRecipe :exec
DELETE FROM recipes WHERE id = $1 AND created_by = $2;
