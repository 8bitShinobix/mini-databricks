#!/usr/bin/env python3
"""
Task runner — spawned by the Go worker as a subprocess.
Receives task info via environment variables, processes data using Dask,
uploads results to MinIO, and prints the output path to stdout.
"""

import json
import os
import sys
import tempfile

import dask.dataframe as dd
import pandas as pd
from minio import Minio
from minio.error import S3Error


def get_minio_client():
    return Minio(
        os.environ["MINIO_ENDPOINT"],
        access_key=os.environ["MINIO_ACCESS_KEY"],
        secret_key=os.environ["MINIO_SECRET_KEY"],
        secure=False,
    )


def download_partition(client, bucket, storage_path, partition_index, total_partitions):
    """Download the dataset from MinIO and return the partition slice as a DataFrame."""
    with tempfile.NamedTemporaryFile(suffix=".csv", delete=False) as tmp:
        tmp_path = tmp.name

    client.fget_object(bucket, storage_path, tmp_path)

    # read full file and slice into partitions
    df = pd.read_csv(tmp_path)
    os.unlink(tmp_path)

    total_rows = len(df)
    rows_per_partition = max(1, total_rows // total_partitions)
    start = partition_index * rows_per_partition
    end = (
        start + rows_per_partition
        if partition_index < total_partitions - 1
        else total_rows
    )

    return df.iloc[start:end].copy()


def run_user_job(entrypoint, df, parameters):
    import importlib.util

    spec = importlib.util.spec_from_file_location("user_job", entrypoint)
    if spec is None or spec.loader is None:
        raise ValueError(f"could not load job script: {entrypoint}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    if not hasattr(module, "run"):
        raise ValueError(
            f"job script {entrypoint} must define a run(df, params) function"
        )

    print(f"calling run() with df shape={df.shape}", file=sys.stderr)
    result = module.run(df, parameters)
    print(f"run() returned: {type(result)} = {result}", file=sys.stderr)  # ← add this

    if result is None:
        raise ValueError("run() returned None — it must return a pandas DataFrame")

    return result


def upload_result(client, bucket, output_key, df):
    """Upload the result DataFrame as CSV to MinIO and return file size."""
    with tempfile.NamedTemporaryFile(suffix=".csv", delete=False, mode="w") as tmp:
        df.to_csv(tmp, index=False)
        tmp_path = tmp.name

    client.fput_object(bucket, output_key, tmp_path, content_type="text/csv")
    size = os.path.getsize(tmp_path)
    os.unlink(tmp_path)
    return size


def main():
    # read task info from environment variables (set by Go worker)
    task_id = os.environ["TASK_ID"]
    run_id = os.environ["RUN_ID"]
    dataset_storage_path = os.environ["DATASET_STORAGE_PATH"]
    partition_index = int(os.environ["PARTITION_INDEX"])
    total_partitions = int(os.environ["TOTAL_PARTITIONS"])
    entrypoint = os.environ["JOB_ENTRYPOINT"]
    parameters = json.loads(os.environ.get("JOB_PARAMETERS", "{}"))
    bucket = os.environ.get("MINIO_BUCKET", "mini-databricks")

    print(
        f"task_runner starting: task={task_id} partition={partition_index}/{total_partitions}",
        file=sys.stderr,
    )

    client = get_minio_client()

    # download partition
    print(
        f"downloading partition {partition_index} from {dataset_storage_path}",
        file=sys.stderr,
    )
    df = download_partition(
        client, bucket, dataset_storage_path, partition_index, total_partitions
    )
    print(f"partition {partition_index} loaded: {len(df)} rows", file=sys.stderr)

    # run user job
    print(f"running entrypoint: {entrypoint}", file=sys.stderr)
    result_df = run_user_job(entrypoint, df, parameters)
    if result_df is None:
        raise ValueError("run() returned None — it must return a pandas DataFrame")
    print(f"job complete: {len(result_df)} rows in result", file=sys.stderr)

    # upload result
    output_key = f"artifacts/{run_id}/partition-{partition_index}.csv"
    size = upload_result(client, bucket, output_key, result_df)
    print(f"result uploaded to {output_key}", file=sys.stderr)

    # print output path and size to stdout so Go can read it
    print(f"{output_key}:{size}")


if __name__ == "__main__":
    main()
