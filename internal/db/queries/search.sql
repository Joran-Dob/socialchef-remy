-- name: SearchRecipesByName :many
SELECT r.*
FROM recipes r
WHERE r.recipe_name ILIKE '%' || $1 || '%'
ORDER BY r.created_at DESC
LIMIT $2;
