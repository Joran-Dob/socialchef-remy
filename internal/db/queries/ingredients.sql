-- name: GetIngredientsByRecipe :many
SELECT * FROM recipe_ingredients WHERE recipe_id = $1;

-- name: GetIngredientsByRecipeAndPart :many
SELECT * FROM recipe_ingredients WHERE recipe_id = $1 AND part_id = $2;

-- name: CreateIngredient :one
INSERT INTO recipe_ingredients (
    recipe_id, part_id, quantity, total_quantity, unit, original_quantity, original_unit, name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: CreateIngredients :copyfrom
INSERT INTO recipe_ingredients (recipe_id, part_id, quantity, unit, original_quantity, original_unit, name) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteIngredientsByRecipe :exec
DELETE FROM recipe_ingredients WHERE recipe_id = $1;
