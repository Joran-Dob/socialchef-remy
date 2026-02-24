-- name: GetIngredientsByRecipe :many
SELECT * FROM recipe_ingredients WHERE recipe_id = $1;

-- name: CreateIngredient :one
INSERT INTO recipe_ingredients (
    recipe_id, quantity, unit, original_quantity, original_unit, name
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: CreateIngredients :copyfrom
INSERT INTO recipe_ingredients (recipe_id, quantity, unit, original_quantity, original_unit, name) VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteIngredientsByRecipe :exec
DELETE FROM recipe_ingredients WHERE recipe_id = $1;
