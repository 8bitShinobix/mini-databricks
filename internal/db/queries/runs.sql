-- name: CreateRun :one
INSERT INTO runs (job_id, workspace_id)
VALUES ($1, $2)
RETURNING *;

-- name: GetRunByID :one
SELECT * FROM runs
WHERE id = $1 AND workspace_id = $2;

-- name: GetRunState :one
SELECT state FROM runs
WHERE id = $1;

-- name: ListRunsByJob :many
SELECT * FROM runs
WHERE job_id = $1
ORDER BY created_at DESC;

-- name: UpdateRunState :one
UPDATE runs
SET state = $2
WHERE id = $1
RETURNING *;

-- name: InitRunProgress :one
UPDATE runs
SET tasks_pending = tasks_total
WHERE id = $1
RETURNING *;

-- name: GetRunProgress :one
SELECT
  id,
  job_id,
  state,
  tasks_total,
  tasks_pending,
  tasks_running,
  tasks_done,
  tasks_failed,
  CASE
    WHEN tasks_total = 0 THEN 0
    ELSE ROUND((tasks_done::numeric / tasks_total::numeric) * 100)
  END AS percent_complete,
  started_at,
  finished_at
FROM runs
WHERE id = $1;

-- name: TaskStarted :one
UPDATE runs
SET
  tasks_pending = tasks_pending - 1,
  tasks_running = tasks_running + 1,
  started_at    = CASE WHEN started_at IS NULL THEN NOW() ELSE started_at END
WHERE id = $1
RETURNING *;

-- name: TaskSucceeded :one
UPDATE runs
SET
  tasks_running = tasks_running - 1,
  tasks_done    = tasks_done + 1
WHERE id = $1
RETURNING *;

-- name: TaskFailed :one
UPDATE runs
SET
  tasks_running = tasks_running - 1,
  tasks_failed  = tasks_failed + 1
WHERE id = $1
RETURNING *;

-- name: UpdateRunTasksTotal :one
UPDATE runs
SET tasks_total = $2
WHERE id = $1
RETURNING *;

-- name: CompleteRun :one
UPDATE runs
SET
  state      = $2,
  finished_at = NOW()
WHERE id = $1
RETURNING *;
