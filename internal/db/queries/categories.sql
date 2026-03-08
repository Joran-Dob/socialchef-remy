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

-- name: GetCuisineCategoriesByUser :many
SELECT DISTINCT cc.name 
FROM cuisine_categories cc
JOIN recipe_cuisine_categories rcc ON cc.id = rcc.cuisine_category_id
JOIN recipes r ON rcc.recipe_id = r.id
WHERE r.created_by = $1
ORDER BY cc.name;

-- name: GetMealTypesByUser :many
SELECT DISTINCT mt.name 
FROM meal_types mt
JOIN recipe_meal_types rmt ON mt.id = rmt.meal_type_id
JOIN recipes r ON rmt.recipe_id = r.id
WHERE r.created_by = $1
ORDER BY mt.name;

-- name: GetOccasionsByUser :many
SELECT DISTINCT o.name 
FROM occasions o
JOIN recipe_occasions ro ON o.id = ro.occasion_id
JOIN recipes r ON ro.recipe_id = r.id
WHERE r.created_by = $1
ORDER BY o.name;

-- name: GetDietaryRestrictionsByUser :many
SELECT DISTINCT dr.name 
FROM dietary_restrictions dr
JOIN recipe_dietary_restrictions rdr ON dr.id = rdr.dietary_restriction_id
JOIN recipes r ON rdr.recipe_id = r.id
WHERE r.created_by = $1
ORDER BY dr.name;

-- name: GetEquipmentByUser :many
SELECT DISTINCT e.name 
FROM equipment e
JOIN recipe_equipment re ON e.id = re.equipment_id
JOIN recipes r ON re.recipe_id = r.id
WHERE r.created_by = $1
ORDER BY e.name;