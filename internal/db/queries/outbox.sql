-- name: CreateOutboxEvent :one
INSERT INTO outbox (event_type, aggregate_id, payload, status)
VALUES ($1, $2, $3, 'PENDING')
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT * FROM outbox
WHERE status = 'PENDING'
ORDER BY created_at ASC
LIMIT 10;

-- name: MarkOutboxDelivered :one
UPDATE outbox
SET status = 'DELIVERED',
    published_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkOutboxFailed :one
UPDATE outbox
SET status = 'FAILED',
    attempts = attempts + 1
WHERE id = $1
RETURNING *;

-- name: CleanupDeliveredOutboxEvents :exec
DELETE FROM outbox
WHERE status = 'DELIVERED'
AND published_at < NOW() - INTERVAL '24 hours';
