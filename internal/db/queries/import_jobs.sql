-- name: GetImportJob :one
SELECT * FROM recipe_import_jobs WHERE id = $1;

-- name: GetImportJobsByUser :many
SELECT * FROM recipe_import_jobs WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateImportJob :one
INSERT INTO recipe_import_jobs (
    id, user_id, url, status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateImportJobStatus :exec
UPDATE recipe_import_jobs 
SET 
    status = $2, 
    progress_step = $3, 
    error = $4,
    updated_at = NOW()
WHERE id = $1;
