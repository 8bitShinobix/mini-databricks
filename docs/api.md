# Mini Databricks API Reference

Mini Databricks exposes a REST API for the platform control plane: user authentication, workspace management, dataset registration, job submission, progress tracking, and artifact retrieval.

This reference is intentionally hybrid in style. It is concise enough to read as a portfolio artifact, but accurate enough to use as a real integration guide for the current implementation.

---

## Base URL

`http://localhost:8080/api/v1`

## API Conventions

- Protected endpoints require `Authorization: Bearer <token>`.
- Request and response bodies use JSON.
- Several read endpoints require `workspace_id` as a query parameter for tenant scoping.
- Error responses use a common envelope:

```json
{
  "error": "description of what went wrong"
}
```

- The current API does not normalize every success response shape:
  `jobs` endpoints return wrapped payloads such as `{ "job": ... }` and `{ "jobs": ... }`, while several `workspace` and `dataset` endpoints return the resource or list directly.
- Common status codes in the current handlers:
  `200 OK`, `201 Created`, `400 Bad Request`, `401 Unauthorized`, `404 Not Found`, `500 Internal Server Error`, `503 Service Unavailable`
- Some service-layer failures currently surface as `500` rather than more specific application-level codes. The documentation below reflects current behavior, not an idealized contract.

## Resource States and Enums

| Type | Values |
| --- | --- |
| `WorkspacePlan` | `free`, `pro`, `enterprise` |
| `FileFormat` | `csv`, `parquet`, `json`, `jsonl` |
| `JobState` | `SUBMITTED`, `VALIDATING`, `QUEUED`, `PROVISIONING`, `RUNNING`, `AGGREGATING`, `RETRYING`, `SUCCEEDED`, `FAILED`, `CANCELLED` |
| `TaskState` | `PENDING`, `LEASED`, `RUNNING`, `CHECKPOINTING`, `SUCCEEDED`, `FAILED`, `DEAD` |
| `DatasetState` | `INITIATED`, `UPLOADING`, `VALIDATING`, `REGISTERING`, `READY`, `FAILED`, `DEPRECATED` |

## Typical Workflow

```text
register -> login -> create workspace -> initiate dataset upload
         -> upload bytes to signed MinIO URL -> complete dataset
         -> submit job -> poll progress -> list artifacts -> download artifact
```

The API is responsible for metadata, orchestration, and status. Dataset file bytes are uploaded directly to object storage using the signed `upload_url` returned by `POST /datasets/initiate`.

---

## Auth

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | No | Create a user account |
| `POST` | `/auth/login` | No | Authenticate and receive a JWT |

### `POST /auth/register`

Creates a user account.

**Request**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response — `201 Created`**

```json
{
  "id": "4d1f4c3a-2d95-4d32-9ba2-8ed0b93d2d78",
  "email": "user@example.com",
  "role": "viewer"
}
```

### `POST /auth/login`

Authenticates a user and returns a JWT.

**Request**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response — `200 OK`**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

## Identity

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/me` | Yes | Return the authenticated user ID and role |

**Response — `200 OK`**

```json
{
  "user_id": "4d1f4c3a-2d95-4d32-9ba2-8ed0b93d2d78",
  "role": "viewer"
}
```

---

## Workspaces

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/workspaces` | Yes | Create a workspace |
| `GET` | `/workspaces` | Yes | List workspaces owned by the current user |
| `GET` | `/workspaces/:id` | Yes | Fetch a single workspace |
| `PATCH` | `/workspaces/:id` | Yes | Update workspace name or plan |
| `DELETE` | `/workspaces/:id` | Yes | Delete a workspace |

### `POST /workspaces`

Creates a workspace for the authenticated user.

**Request**

```json
{
  "name": "analytics-team",
  "plan": "free"
}
```

**Response — `201 Created`**

```json
{
  "id": "99ded1e7-faf0-4a59-a611-916047cd43ae",
  "owner_id": "4d1f4c3a-2d95-4d32-9ba2-8ed0b93d2d78",
  "name": "analytics-team",
  "plan": "free",
  "created_at": "2026-05-12T10:15:30Z"
}
```

### Notes

- `GET /workspaces` returns a raw JSON array of workspaces.
- Newly registered users are created with the default role `viewer`.
- `PATCH /workspaces/:id` accepts partial updates:

```json
{
  "name": "analytics-prod",
  "plan": "pro"
}
```

- `DELETE /workspaces/:id` returns:

```json
{
  "message": "workspace deleted"
}
```

---

## Datasets

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/datasets/initiate` | Yes | Create dataset metadata and return a signed upload URL |
| `POST` | `/datasets/:id/complete` | Yes | Mark an upload as complete |
| `GET` | `/datasets/:id?workspace_id=...` | Yes | Fetch a single dataset |
| `GET` | `/datasets?workspace_id=...` | Yes | List datasets in a workspace |
| `DELETE` | `/datasets/:id?workspace_id=...` | Yes | Delete a dataset |

### `POST /datasets/initiate`

Creates the dataset record and returns a signed URL for direct upload to MinIO.

**Request**

```json
{
  "workspace_id": "99ded1e7-faf0-4a59-a611-916047cd43ae",
  "name": "sales-data",
  "file_format": "csv"
}
```

**Response — `201 Created`**

```json
{
  "dataset": {
    "id": "f2497747-9ce2-4153-9a7f-cccb1507e9ce",
    "workspace_id": "99ded1e7-faf0-4a59-a611-916047cd43ae",
    "created_by": "4d1f4c3a-2d95-4d32-9ba2-8ed0b93d2d78",
    "name": "sales-data",
    "state": "INITIATED",
    "file_format": "csv",
    "created_at": "2026-05-12T10:18:02Z",
    "updated_at": "2026-05-12T10:18:02Z"
  },
  "upload_url": "http://localhost:9000/mini-databricks/99ded1e7-faf0-4a59-a611-916047cd43ae/..."
}
```

### `POST /datasets/:id/complete`

Marks the upload as complete after the client uploads bytes to the signed URL.

**Request**

```json
{
  "storage_path": "99ded1e7-faf0-4a59-a611-916047cd43ae/f2497747-9ce2-4153-9a7f-cccb1507e9ce/sales-data.csv",
  "size_bytes": 1024
}
```

### Notes

- `GET /datasets/:id` returns the dataset object directly.
- `GET /datasets` returns a raw JSON array of datasets.
- `DELETE /datasets/:id` returns:

```json
{
  "message": "dataset deleted"
}
```

---

## Jobs

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/jobs` | Yes | Submit a new compute job |
| `GET` | `/jobs?workspace_id=...` | Yes | List jobs in a workspace |
| `GET` | `/jobs/:id?workspace_id=...` | Yes | Fetch a single job |
| `POST` | `/jobs/:id/cancel` | Yes | Cancel a job and mark runs as cancelled |
| `GET` | `/jobs/:id/progress?workspace_id=...` | Yes | Return run-level progress counters |
| `GET` | `/jobs/:id/artifacts?workspace_id=...` | Yes | List artifacts produced by a job |
| `GET` | `/jobs/:id/artifacts/:artifact_id/download` | Yes | Return a signed download URL |

### `POST /jobs`

Submits a job against an existing dataset.

**Request**

```json
{
  "workspace_id": "99ded1e7-faf0-4a59-a611-916047cd43ae",
  "dataset_id": "f2497747-9ce2-4153-9a7f-cccb1507e9ce",
  "entrypoint": "/absolute/path/to/sdk/python/jobs/analysis.py",
  "parameters": {
    "region": "IN"
  },
  "compute": {
    "cpu": 4,
    "memory_gb": 16,
    "workers": 3
  },
  "max_retries": 3,
  "idempotency_key": "5e4a2cdb-e4f2-44a8-8f2d-fb2e41ea7b0d"
}
```

**Response — `201 Created`**

```json
{
  "job": {
    "id": "6d4b67d5-bb9b-4297-a255-a084d60f6033",
    "workspace_id": "99ded1e7-faf0-4a59-a611-916047cd43ae",
    "created_by": "4d1f4c3a-2d95-4d32-9ba2-8ed0b93d2d78",
    "dataset_id": "f2497747-9ce2-4153-9a7f-cccb1507e9ce",
    "state": "SUBMITTED",
    "entrypoint": "/absolute/path/to/sdk/python/jobs/analysis.py",
    "parameters": {
      "region": "IN"
    },
    "compute": {
      "cpu": 4,
      "memory_gb": 16,
      "workers": 3
    },
    "retry_count": 0,
    "max_retries": 3,
    "idempotency_key": "5e4a2cdb-e4f2-44a8-8f2d-fb2e41ea7b0d",
    "created_at": "2026-05-12T10:22:11Z"
  }
}
```

### `GET /jobs/:id/progress?workspace_id=...`

Returns run-level progress derived from the current run for the job.

**Response — `200 OK`**

```json
{
  "job_id": "6d4b67d5-bb9b-4297-a255-a084d60f6033",
  "run_id": "793d4c4d-9495-4674-9d90-cfd0ddb6811d",
  "state": "RUNNING",
  "progress": {
    "total": 3,
    "pending": 1,
    "running": 1,
    "done": 1,
    "failed": 0,
    "percent": 33
  },
  "started_at": "2026-05-12T10:22:18Z",
  "finished_at": null
}
```

### `GET /jobs/:id/artifacts/:artifact_id/download`

Returns a signed download URL for an artifact.

**Response — `200 OK`**

```json
{
  "download_url": "http://localhost:9000/mini-databricks/artifacts/793d4c4d-9495-4674-9d90-cfd0ddb6811d/partition-0.csv?X-Amz-Algorithm=AWS4-HMAC-SHA256..."
}
```

### Notes

- `GET /jobs` returns `{ "jobs": [...] }`.
- `GET /jobs/:id` returns `{ "job": { ... } }`.
- `POST /jobs/:id/cancel` returns `{ "job": { ... } }` after updating the job state to `CANCELLED`.
- `GET /jobs/:id/artifacts` returns `{ "artifacts": [...] }`.
- `workspace_id` is required for `GET /jobs`, `GET /jobs/:id`, `GET /jobs/:id/progress`, and `GET /jobs/:id/artifacts`.

---

## Health and Admin

These endpoints are public in the current implementation.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/health` | No | Liveness check |
| `GET` | `/ready` | No | Readiness check with database connectivity |
| `GET` | `/admin/stats` | No | Queue depth summary |

### Example responses

`GET /health`

```json
{
  "status": "ok"
}
```

`GET /ready`

```json
{
  "status": "ready"
}
```

`GET /admin/stats`

```json
{
  "pending_tasks": 2,
  "running_tasks": 1
}
```

---

## End-to-End Example

1. Register a user with `POST /auth/register`.
2. Authenticate with `POST /auth/login` and store the returned JWT.
3. Create a workspace with `POST /workspaces`.
4. Initiate dataset upload with `POST /datasets/initiate`.
5. Upload the dataset file directly to the returned `upload_url`.
6. Finalize metadata with `POST /datasets/:id/complete`.
7. Submit a job with `POST /jobs`.
8. Poll `GET /jobs/:id/progress?workspace_id=...` until the state becomes terminal.
9. List outputs with `GET /jobs/:id/artifacts?workspace_id=...`.
10. Download a result with `GET /jobs/:id/artifacts/:artifact_id/download`.

## Related References

- [README](../Readme.md)
- [Python SDK example](../sdk/python/example.py)
