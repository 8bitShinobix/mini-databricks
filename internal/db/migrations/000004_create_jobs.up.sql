CREATE TABLE jobs (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  created_by       UUID NOT NULL REFERENCES users(id),
  dataset_id       UUID NOT NULL REFERENCES datasets(id),
  state            job_state NOT NULL DEFAULT 'SUBMITTED',
  entrypoint       TEXT NOT NULL,
  parameters       JSONB NOT NULL DEFAULT '{}',
  compute          JSONB NOT NULL DEFAULT '{}',
  retry_count      INTEGER NOT NULL DEFAULT 0,
  max_retries      INTEGER NOT NULL DEFAULT 3,
  idempotency_key  TEXT UNIQUE,
  error_message    TEXT,
  created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at       TIMESTAMP,
  finished_at      TIMESTAMP
);
