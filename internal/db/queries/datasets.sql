-- name: InitiateDataset :one
INSERT INTO datasets (workspace_id, created_by, name, file_format, state)
VALUES ($1, $2, $3, $4, 'INITIATED')
RETURNING *;

-- name: UpdateDatasetState :one
UPDATE datasets
SET state = $2
WHERE id = $1
RETURNING *;

-- name: UpdateDatasetStoragePath :one
UPDATE datasets
SET storage_path = $2,
    size_bytes = $3,
    state = 'READY'
WHERE id = $1
RETURNING *;

-- name: GetDatasetByID :one
SELECT * FROM datasets
WHERE id = $1 AND workspace_id = $2;

-- name: ListDatasetsByWorkspace :many
SELECT * FROM datasets
WHERE workspace_id = $1
ORDER BY created_at DESC;

-- name: DeleteDataset :exec
DELETE FROM datasets
WHERE id = $1 AND workspace_id = $2;

-- name: CleanupStaleDatasetsInitiated :exec
DELETE FROM datasets
WHERE state = 'INITIATED'
AND created_at < NOW() - INTERVAL '24 hours';

-- name: GetDatasetByJobID :one
SELECT d.* FROM datasets d
JOIN jobs j ON j.dataset_id = d.id
WHERE j.id = $1;
