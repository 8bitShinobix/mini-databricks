import time
import uuid
from typing import Dict, List, Optional

from .client import Client


class JobsAPI:
    def __init__(self, client: Client):
        self.client = client

    def submit(
        self,
        workspace_id: str,
        dataset_id: str,
        entrypoint: str,
        parameters: Optional[dict] = None,
        compute: Optional[dict] = None,
        max_retries: int = 3,
    ) -> dict:
        return self.client.post(
            "/jobs",
            {
                "workspace_id": workspace_id,
                "dataset_id": dataset_id,
                "entrypoint": entrypoint,
                "parameters": parameters or {},
                "compute": compute or {"cpu": 2, "memory_gb": 4, "workers": 1},
                "max_retries": max_retries,
                "idempotency_key": str(uuid.uuid4()),
            },
        )

    def get(self, job_id: str, workspace_id: str) -> dict:
        return self.client.get(f"/jobs/{job_id}", params={"workspace_id": workspace_id})

    def list(self, workspace_id: str) -> dict:
        return self.client.get("/jobs", params={"workspace_id": workspace_id})

    def cancel(self, job_id: str) -> dict:
        return self.client.post(f"/jobs/{job_id}/cancel")

    def progress(self, job_id: str, workspace_id: str) -> dict:
        """Returns task-level progress: done, total, failed."""
        return self.client.get(
            f"/jobs/{job_id}/progress",
            params={"workspace_id": workspace_id},
        )

    def wait(
        self,
        job_id: str,
        workspace_id: str,
        poll_interval: int = 2,
    ) -> dict:
        """
        Poll until job reaches a terminal state.
        Shows a live progress bar: ████░░░ 2/3 tasks (RUNNING)
        """
        try:
            from tqdm import tqdm as tqdm_cls
        except ImportError:
            tqdm_cls = None

        terminal_states = {"SUCCEEDED", "FAILED", "CANCELLED"}
        bar = None

        try:
            while True:
                job = self.get(job_id, workspace_id)["job"]
                state = job["state"]

                try:
                    prog = self.progress(job_id, workspace_id)
                    tasks_done = prog.get("tasks_done", 0)
                    tasks_total = prog.get("tasks_total", 0)
                    tasks_failed = prog.get("tasks_failed", 0)
                except Exception:
                    tasks_done = tasks_total = tasks_failed = 0

                if tqdm_cls and tasks_total > 0:
                    if bar is None:
                        bar = tqdm_cls(
                            total=tasks_total,
                            desc=f"job {job_id[:8]}",
                            unit="task",
                            bar_format="{l_bar}{bar}| {n_fmt}/{total_fmt} tasks [{elapsed}]",
                        )
                    advance = tasks_done - bar.n
                    if advance > 0:
                        bar.update(advance)
                    bar.set_postfix(state=state, failed=tasks_failed)
                else:
                    if tasks_total > 0:
                        print(
                            f"  {job_id[:8]} [{state}] {tasks_done}/{tasks_total} tasks",
                            end="\r",
                        )
                    else:
                        print(f"  {job_id[:8]} [{state}]", end="\r")

                if state in terminal_states:
                    if bar:
                        bar.close()
                    print()
                    return job

                time.sleep(poll_interval)

        except KeyboardInterrupt:
            if bar:
                bar.close()
            print(f"\ncancelling job {job_id}...")
            self.cancel(job_id)
            return self.get(job_id, workspace_id)["job"]

    def artifacts(self, job_id: str, workspace_id: str) -> List[Dict]:
        """List all artifacts produced by a job."""
        resp = self.client.get(
            f"/jobs/{job_id}/artifacts",
            params={"workspace_id": workspace_id},
        )
        return resp.get("artifacts", [])

    def download_artifact(self, job_id: str, artifact_id: str, dest_path: str) -> str:
        """
        Download an artifact to a local file.
        Returns the path it was saved to.
        """
        import requests

        try:
            from tqdm import tqdm as tqdm_cls
        except ImportError:
            tqdm_cls = None

        resp = self.client.get(f"/jobs/{job_id}/artifacts/{artifact_id}/download")
        url = resp.get("download_url")
        if not url:
            raise Exception("no download URL returned")

        r = requests.get(url, stream=True)
        r.raise_for_status()

        total = int(r.headers.get("content-length", 0))
        bar = (
            tqdm_cls(total=total, unit="B", unit_scale=True, desc=dest_path)
            if tqdm_cls
            else None
        )

        with open(dest_path, "wb") as f:
            for chunk in r.iter_content(chunk_size=8192):
                f.write(chunk)
                if bar:
                    bar.update(len(chunk))

        if bar:
            bar.close()

        return dest_path
