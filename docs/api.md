# Mini Databricks API Reference

Base URL: `http://localhost:8080/api/v1`

All protected endpoints require `Authorization: Bearer <token>` header.

---

## Authentication

### POST /auth/register
Register a new user.

**Body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "name": "Test User"
}
```

### POST /auth/login
Login and get a JWT token.

**Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response:**
```json
{"token": "eyJ..."}
```

---

## Jobs

### POST /jobs
Submit a new job.

**Body:**
```json
{
  "workspace_id": "uuid",
  "dataset_id": "uuid",
  "entrypoint": "jobs/analysis.py",
  "parameters": {"region": "IN"},
  "compute": {"cpu": 4, "memory_gb": 16, "workers": 10},
  "max_retries": 3,
  "idempotency_key": "unique-key"
}
```

### GET /jobs?workspace_id=uuid
List all jobs in a workspace.

### GET /jobs/:id?workspace_id=uuid
Get a specific job.

### GET /jobs/:id/progress?workspace_id=uuid
Get job progress with task breakdown.

**Response:**
```json
{
  "job_id": "uuid",
  "state": "SUCCEEDED",
  "progress": {
    "total": 3, "pending": 0, "running": 0,
    "done": 3, "failed": 0, "percent": 100
  },
  "started_at": "2026-04-27T11:29:40Z",
  "finished_at": "2026-04-27T11:29:53Z"
}
```

### POST /jobs/:id/cancel
Cancel a running job.

### GET /jobs/:id/artifacts?workspace_id=uuid
List artifacts produced by a job.

### GET /jobs/:id/artifacts/:artifact_id/download
Get a signed download URL for an artifact.

---

## Datasets

### POST /datasets/initiate
Initiate a dataset upload.

**Body:**
```json
{
  "workspace_id": "uuid",
  "name": "sales-data",
  "file_format": "csv"
}
```

**Response:**
```json
{
  "dataset": {"id": "uuid", ...},
  "upload_url": "https://minio/..."
}
```

### POST /datasets/:id/complete
Mark a dataset upload as complete.

**Body:**
```json
{
  "storage_path": "workspace-id/dataset-id/name.csv",
  "size_bytes": 1024
}
```

### GET /datasets?workspace_id=uuid
List all datasets in a workspace.

### DELETE /datasets/:id?workspace_id=uuid
Delete a dataset.

---

## Health

### GET /health
Liveness check. Returns `{"status": "ok"}`.

### GET /ready
Readiness check. Verifies DB connectivity.

### GET /admin/stats
Queue depth and active task counts.

---

## Error Responses

All errors follow this format:
```json
{"error": "description of what went wrong"}
```

Common HTTP status codes:
- `400` — Bad request / validation error
- `401` — Missing or invalid token
- `404` — Resource not found
- `429` — Rate limit exceeded
- `500` — Internal server error

---

## Job States
`SUBMITTED → QUEUED → RUNNING → SUCCEEDED | FAILED | CANCELLED`

## Task States
`PENDING → LEASED → SUCCEEDED | FAILED | DEAD`

## Dataset States
`INITIATED → READY`
