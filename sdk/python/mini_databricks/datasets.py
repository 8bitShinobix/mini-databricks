import os

import requests

from .client import Client


class DatasetsAPI:
    def __init__(self, client: Client):
        self.client = client

    def upload(
        self, workspace_id: str, name: str, file_format: str, file_path: str
    ) -> dict:
        response = self.client.post(
            "/datasets/initiate",
            {
                "workspace_id": workspace_id,
                "name": name,
                "file_format": file_format,
            },
        )

        dataset = response["dataset"]
        upload_url = response["upload_url"]
        size_bytes = os.path.getsize(file_path)

        # upload with progress bar
        try:
            from tqdm import tqdm

            bar = tqdm(
                total=size_bytes,
                unit="B",
                unit_scale=True,
                desc=f"uploading {name}",
            )

            def _read_with_progress(path):
                with open(path, "rb") as f:
                    while True:
                        chunk = f.read(8192)
                        if not chunk:
                            break
                        bar.update(len(chunk))
                        yield chunk

            put_response = requests.put(upload_url, data=_read_with_progress(file_path))
            bar.close()
        except ImportError:
            print(f"uploading {name} ({size_bytes} bytes)...")
            with open(file_path, "rb") as f:
                put_response = requests.put(upload_url, data=f.read())

        if not put_response.ok:
            raise Exception(f"upload failed: {put_response.status_code}")

        storage_path = f"{workspace_id}/{dataset['id']}/{name}.{file_format}"

        completed = self.client.post(
            f"/datasets/{dataset['id']}/complete",
            {
                "storage_path": storage_path,
                "size_bytes": size_bytes,
            },
        )

        return completed

    def list(self, workspace_id: str) -> dict:
        return self.client.get("/datasets", params={"workspace_id": workspace_id})

    def delete(self, dataset_id: str, workspace_id: str) -> dict:
        return self.client.delete(f"/datasets/{dataset_id}?workspace_id={workspace_id}")
