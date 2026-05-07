
CREATE TYPE  workspace_plan AS ENUM ('free', 'pro', 'enterprise');

CREATE TABLE workspaces (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  plan        workspace_plan NOT NULL DEFAULT 'free',
  created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
