-- name: GetImportJob :one
SELECT * FROM recipe_import_jobs WHERE id = $1;

-- name: GetImportJobByJobID :one
SELECT * FROM recipe_import_jobs WHERE job_id = $1;

-- name: GetImportJobsByUser :many
SELECT * FROM recipe_import_jobs WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateImportJob :one
INSERT INTO recipe_import_jobs (
    id, job_id, user_id, url, origin, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateImportJobStatus :exec
UPDATE recipe_import_jobs 
SET 
    status = $2, 
    progress_step = $3, 
    error = $4,
    updated_at = NOW()
WHERE job_id = $1;
