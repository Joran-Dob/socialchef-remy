-- name: GetIngredientsByRecipe :many
SELECT * FROM recipe_ingredients WHERE recipe_id = $1;

-- name: CreateIngredient :one
INSERT INTO recipe_ingredients (
    recipe_id, quantity, unit, name
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: CreateIngredients :copyfrom
INSERT INTO recipe_ingredients (recipe_id, quantity, unit, name) VALUES ($1, $2, $3, $4);

-- name: DeleteIngredientsByRecipe :exec
DELETE FROM recipe_ingredients WHERE recipe_id = $1;
