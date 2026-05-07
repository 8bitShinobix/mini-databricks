CREATE TABLE runs (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id         UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  workspace_id   UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  state          run_state NOT NULL DEFAULT 'SUBMITTED',
  attempt_number INTEGER NOT NULL DEFAULT 1,
  tasks_total    INTEGER NOT NULL DEFAULT 0,
  tasks_done     INTEGER NOT NULL DEFAULT 0,
  error_message  TEXT,
  created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at     TIMESTAMP,
  finished_at    TIMESTAMP
);
