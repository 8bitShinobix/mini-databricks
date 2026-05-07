from mini_databricks import MiniDatabricks

sdk = MiniDatabricks(base_url="http://localhost:8080/api/v1")
sdk.login("test@test.com", "password123")
print("✅ logged in")

WORKSPACE_ID = "99ded1e7-faf0-4a59-a611-916047cd43ae"
DATASET_ID = "f2497747-9ce2-4153-9a7f-cccb1507e9ce"

# submit job
job = sdk.jobs.submit(
    workspace_id=WORKSPACE_ID,
    dataset_id=DATASET_ID,
    entrypoint="/Users/durgeshchandrakar/Documents/Coding/building_my_own_x/mini-databricks/sdk/python/jobs/analysis.py",
    parameters={"region": "IN"},
    compute={"cpu": 4, "memory_gb": 16, "workers": 3},
)
job_id = job["job"]["id"]
print(f"✅ submitted job: {job_id}")

# wait with progress bar
final = sdk.jobs.wait(job_id, WORKSPACE_ID)
print(f"✅ job finished: {final['state']}")

# list artifacts
artifacts = sdk.jobs.artifacts(job_id, WORKSPACE_ID)
print(f"✅ {len(artifacts)} artifact(s) produced:")
for a in artifacts:
    print(f"   - {a['storage_path']} ({a['size_bytes']} bytes)")

# download first artifact
if artifacts:
    path = sdk.jobs.download_artifact(
        job_id, artifacts[0]["id"], "/tmp/result-partition-0.csv"
    )
    print(f"✅ downloaded to {path}")
