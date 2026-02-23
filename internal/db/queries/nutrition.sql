-- name: GetNutritionByRecipe :one
SELECT * FROM recipe_nutrition WHERE recipe_id = $1;

-- name: CreateNutrition :one
INSERT INTO recipe_nutrition (
    recipe_id, calories, protein, carbs, fat, fiber
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateNutrition :one
UPDATE recipe_nutrition
SET 
    calories = $2,
    protein = $3,
    carbs = $4,
    fat = $5,
    fiber = $6,
    updated_at = NOW()
WHERE recipe_id = $1
RETURNING *;
