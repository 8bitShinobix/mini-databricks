ALTER TABLE runs
  DROP COLUMN tasks_pending,
  DROP COLUMN tasks_running,
  DROP COLUMN tasks_failed;
