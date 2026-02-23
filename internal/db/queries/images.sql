-- name: GetStoredImageByHash :one
SELECT * FROM stored_images WHERE content_hash = $1;

-- name: CreateStoredImage :one
INSERT INTO stored_images (
    id, content_hash, storage_path
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: CreateRecipeImage :one
INSERT INTO recipe_images (
    recipe_id, image_id
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetImagesByRecipe :many
SELECT si.* 
FROM stored_images si 
JOIN recipe_images ri ON si.id = ri.image_id 
WHERE ri.recipe_id = $1;

-- name: DeleteRecipeImages :exec
DELETE FROM recipe_images WHERE recipe_id = $1;
