-- name: CreateJob :one
INSERT INTO jobs (workspace_id, created_by, dataset_id, entrypoint, parameters, compute, max_retries, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetJobByID :one
SELECT * FROM jobs
WHERE id = $1 AND workspace_id = $2;

-- name: ListJobsByWorkspace :many
SELECT * FROM jobs
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: UpdateJobState :one
UPDATE jobs
SET state = $2
WHERE id = $1
RETURNING *;

-- name: GetJobByRunID :one
SELECT j.* FROM jobs j
JOIN runs r ON r.job_id = j.id
WHERE r.id = $1;

-- name: CompleteJob :one
UPDATE jobs
SET
  state       = $2,
  finished_at = NOW()
WHERE id = $1
RETURNING *;
