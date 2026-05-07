CREATE TYPE file_format AS ENUM ('csv', 'parquet', 'json', 'jsonl');

CREATE TABLE datasets (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  created_by   UUID NOT NULL REFERENCES users(id),
  name         TEXT NOT NULL,
  state        dataset_state NOT NULL DEFAULT 'INITIATED',
  storage_path TEXT,
  file_format  file_format NOT NULL,
  size_bytes   BIGINT,
  created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
