-- name: CreateArtifact :one
INSERT INTO artifacts (run_id, workspace_id, name, storage_path, content_type, size_bytes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListArtifactsByRun :many
SELECT * FROM artifacts
WHERE run_id = $1
ORDER BY created_at DESC;

-- name: ListArtifactsByJob :many
SELECT a.* FROM artifacts a
JOIN runs r ON r.id = a.run_id
WHERE r.job_id = $1
ORDER BY a.created_at DESC;

-- name: GetArtifact :one
SELECT * FROM artifacts
WHERE id = $1;
