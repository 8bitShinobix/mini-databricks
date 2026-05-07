-- jobs: most common query is by workspace_id + state
CREATE INDEX idx_jobs_workspace_id ON jobs(workspace_id);
CREATE INDEX idx_jobs_workspace_state ON jobs(workspace_id, state);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- runs: queried by job_id constantly
CREATE INDEX idx_runs_job_id ON runs(job_id);
CREATE INDEX idx_runs_workspace_id ON runs(workspace_id);

-- tasks: leasing query hits state + lease_expires_at every 2 seconds
CREATE INDEX idx_tasks_state ON tasks(state);
CREATE INDEX idx_tasks_run_id ON tasks(run_id);
CREATE INDEX idx_tasks_lease_expires ON tasks(lease_expires_at) WHERE state = 'LEASED';

-- artifacts: queried by run_id
CREATE INDEX idx_artifacts_run_id ON artifacts(run_id);

-- outbox: dispatcher queries pending events constantly
CREATE INDEX idx_outbox_status ON outbox(status) WHERE status = 'PENDING';

-- datasets: queried by workspace_id
CREATE INDEX idx_datasets_workspace_id ON datasets(workspace_id);
