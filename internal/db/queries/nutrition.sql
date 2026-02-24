-- name: GetNutritionByRecipe :one
SELECT * FROM recipe_nutrition WHERE recipe_id = $1;

-- name: CreateNutrition :one
INSERT INTO recipe_nutrition (
    recipe_id, protein, carbs, fat, fiber
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateNutrition :one
UPDATE recipe_nutrition
SET 
    protein = $2,
    carbs = $3,
    fat = $4,
    fiber = $5,
    updated_at = NOW()
WHERE recipe_id = $1
RETURNING *;
