-- name: GetBulkImportJob :one
SELECT * FROM bulk_import_jobs WHERE id = $1;

-- name: GetBulkImportJobByJobID :one
SELECT * FROM bulk_import_jobs WHERE job_id = $1;

-- name: GetBulkImportJobsByUser :many
SELECT * FROM bulk_import_jobs WHERE user_id = $1 ORDER BY created_at DESC;

-- name: CreateBulkImportJob :one
INSERT INTO bulk_import_jobs (
    id, job_id, user_id, total_urls, status
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateBulkImportJobStatus :exec
UPDATE bulk_import_jobs 
SET 
    status = $2,
    summary = $3,
    updated_at = NOW()
WHERE job_id = $1;

-- name: IncrementBulkImportCounters :exec
UPDATE bulk_import_jobs 
SET 
    processed_count = processed_count + 1,
    success_count = success_count + $2,
    failed_count = failed_count + $3,
    updated_at = NOW()
WHERE job_id = $1;

-- name: GetUserActiveBulkImportCount :one
SELECT COUNT(*) FROM bulk_import_jobs 
WHERE user_id = $1 
AND status IN ('QUEUED', 'EXECUTING');

-- name: CancelBulkImportJob :exec
UPDATE bulk_import_jobs 
SET 
    status = 'CANCELED',
    updated_at = NOW()
WHERE job_id = $1 
AND status IN ('QUEUED', 'EXECUTING');

-- name: GetImportJobsByBulkJobID :many
SELECT * FROM recipe_import_jobs WHERE bulk_job_id = $1 ORDER BY created_at ASC;

-- name: UpdateImportJobWithBulkID :exec
UPDATE recipe_import_jobs 
SET bulk_job_id = $2
WHERE job_id = $1;
