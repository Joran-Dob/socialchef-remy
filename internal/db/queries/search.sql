-- name: SearchRecipes :many
SELECT r.*, 1 - (r.embedding <=> $1::vector) as similarity
FROM recipes r
WHERE (r.created_by = $2 OR r.is_public = true)
ORDER BY r.embedding <=> $1::vector
LIMIT $3;
