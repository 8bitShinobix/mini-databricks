-- name: CreateTask :one
INSERT INTO tasks (run_id, workspace_id, partition_index)
VALUES ($1, $2, $3)
RETURNING *;

-- name: LeaseTask :one
UPDATE tasks
SET state = 'LEASED',
    lease_owner = $1,
    lease_expires_at = NOW() + INTERVAL '30 seconds'
WHERE id = (
    SELECT id FROM tasks
    WHERE state = 'PENDING'
    AND (lease_expires_at IS NULL OR lease_expires_at < NOW())
    ORDER BY created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: HeartbeatTask :one
UPDATE tasks
SET lease_expires_at = NOW() + INTERVAL '30 seconds'
WHERE id = $1
AND lease_owner = $2
RETURNING *;

-- name: UpdateTaskState :one
UPDATE tasks
SET state = $2
WHERE id = $1
RETURNING *;

-- name: ListTasksByRun :many
SELECT * FROM tasks
WHERE run_id = $1
ORDER BY partition_index ASC;

-- name: GetPendingTaskCount :one
SELECT COUNT(*) FROM tasks
WHERE state = 'PENDING';

-- name: GetRunningTaskCount :one
SELECT COUNT(*) FROM tasks
WHERE state = 'LEASED';

-- name: RetryTask :one
UPDATE tasks
SET
  state       = 'PENDING',
  retry_count = retry_count + 1,
  lease_owner = NULL,
  lease_expires_at = NULL,
  error_message = $2
WHERE id = $1
RETURNING *;

-- name: FailTaskWithError :one
UPDATE tasks
SET
  state         = 'FAILED',
  error_message = $2,
  finished_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: CleanupDeadTasks :exec
DELETE FROM tasks
WHERE state IN ('DEAD', 'FAILED')
AND created_at < NOW() - INTERVAL '7 days';

-- name: GetRunTaskCount :one
SELECT COUNT(*) FROM tasks
WHERE run_id = $1;
