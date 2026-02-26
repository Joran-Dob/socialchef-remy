-- Cuisine Categories
-- name: GetOrCreateCuisineCategory :one
INSERT INTO cuisine_categories (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: AddRecipeCuisineCategory :exec
INSERT INTO recipe_cuisine_categories (recipe_id, cuisine_category_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- Meal Types
-- name: GetOrCreateMealType :one
INSERT INTO meal_types (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: AddRecipeMealType :exec
INSERT INTO recipe_meal_types (recipe_id, meal_type_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- Occasions
-- name: GetOrCreateOccasion :one
INSERT INTO occasions (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: AddRecipeOccasion :exec
INSERT INTO recipe_occasions (recipe_id, occasion_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- Dietary Restrictions
-- name: GetOrCreateDietaryRestriction :one
INSERT INTO dietary_restrictions (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: AddRecipeDietaryRestriction :exec
INSERT INTO recipe_dietary_restrictions (recipe_id, dietary_restriction_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- Equipment
-- name: GetOrCreateEquipment :one
INSERT INTO equipment (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = $1
RETURNING id;

-- name: AddRecipeEquipment :exec
INSERT INTO recipe_equipment (recipe_id, equipment_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;