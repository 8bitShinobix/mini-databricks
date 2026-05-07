CREATE TABLE tasks (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id           UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
  workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  state            task_state NOT NULL DEFAULT 'PENDING',
  partition_index  INTEGER NOT NULL,
  lease_owner      TEXT,
  lease_expires_at TIMESTAMP,
  retry_count      INTEGER NOT NULL DEFAULT 0,
  checkpoint_path  TEXT,
  error_message    TEXT,
  created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at       TIMESTAMP,
  finished_at      TIMESTAMP
);
