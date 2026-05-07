-- name: CreateWorkspace :one
INSERT INTO workspaces (owner_id, name, plan)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWorkspaceByID :one
SELECT * FROM workspaces
WHERE id = $1;

-- name: GetWorkspacesByOwner :many
SELECT * FROM workspaces
WHERE owner_id = $1;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1 AND owner_id = $2;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = COALESCE($2, name),
    plan = COALESCE($3, plan)
WHERE id = $1 AND owner_id = $4
RETURNING *;
