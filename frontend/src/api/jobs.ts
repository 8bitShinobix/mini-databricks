import client from "./client";

export interface Job {
  id: string;
  workspace_id: string;
  state: string;
  entrypoint: string;
  parameters: Record<string, string>;
  compute: Record<string, number>;
  retry_count: number;
  max_retries: number;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
  error_message: string | null;
}

export interface JobProgress {
  job_id: string;
  run_id: string;
  state: string;
  progress: {
    total: number;
    pending: number;
    running: number;
    done: number;
    failed: number;
    percent: number;
  };
  started_at: string | null;
  finished_at: string | null;
}

export interface Artifact {
  id: string;
  name: string;
  storage_path: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface Dataset {
  id: string;
  name: string;
  file_format: string;
  size_bytes: number;
  state: string;
  created_at: string;
}

const WORKSPACE_ID = "99ded1e7-faf0-4a59-a611-916047cd43ae";

export const listJobs = () =>
  client
    .get<{ jobs: Job[] }>(`/jobs?workspace_id=${WORKSPACE_ID}`)
    .then((r) => r.data.jobs);

export const getJob = (id: string) =>
  client
    .get<{ job: Job }>(`/jobs/${id}?workspace_id=${WORKSPACE_ID}`)
    .then((r) => r.data.job);

export const getJobProgress = (id: string) =>
  client
    .get<JobProgress>(`/jobs/${id}/progress?workspace_id=${WORKSPACE_ID}`)
    .then((r) => r.data);

export const cancelJob = (id: string) =>
  client.post(`/jobs/${id}/cancel`).then((r) => r.data);

export const listArtifacts = (id: string) =>
  client
    .get<{
      artifacts: Artifact[];
    }>(`/jobs/${id}/artifacts?workspace_id=${WORKSPACE_ID}`)
    .then((r) => r.data.artifacts);

export const getDownloadUrl = (jobId: string, artifactId: string) =>
  client
    .get<{
      download_url: string;
    }>(`/jobs/${jobId}/artifacts/${artifactId}/download`)
    .then((r) => r.data.download_url);

export const login = (email: string, password: string) =>
  client
    .post<{ token: string }>("/auth/login", { email, password })
    .then((r) => r.data.token);

export const listDatasets = () =>
  client
    .get<
      Dataset[] | { datasets: Dataset[] }
    >(`/datasets?workspace_id=${WORKSPACE_ID}`)
    .then((r) => (Array.isArray(r.data) ? r.data : (r.data.datasets ?? [])));

export const initiateUpload = (name: string, format: string) =>
  client
    .post<{
      dataset: { id: string; workspace_id: string };
      upload_url: string;
    }>("/datasets/initiate", {
      workspace_id: WORKSPACE_ID,
      name,
      file_format: format,
    })
    .then((r) => ({
      dataset_id: r.data.dataset.id,
      upload_url: r.data.upload_url,
      storage_path: `${WORKSPACE_ID}/${r.data.dataset.id}/${name}.${format}`,
    }));

export const completeUpload = (
  datasetId: string,
  storagePath: string,
  sizeBytes: number,
) =>
  client
    .post(`/datasets/${datasetId}/complete`, {
      storage_path: storagePath,
      size_bytes: sizeBytes,
    })
    .then((r) => r.data);

export const registerUser = (email: string, password: string, name: string) =>
  client.post("/auth/register", { email, password, name }).then((r) => r.data);
