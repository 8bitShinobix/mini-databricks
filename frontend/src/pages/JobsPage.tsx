import { useEffect, useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { JobStatusBadge } from "@/components/JobStatusBadge";
import {
  listJobs,
  listDatasets,
  initiateUpload,
  completeUpload,
  type Job,
  type Dataset,
} from "@/api/jobs";

export function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [datasets, setDatasets] = useState<Dataset[]>([]);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadStatus, setUploadStatus] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchJobs = () => listJobs().then(setJobs).catch(console.error);
    const fetchDatasets = () =>
      listDatasets()
        .then((data) => setDatasets(data ?? []))
        .catch(() => setDatasets([]));
    fetchJobs();
    fetchDatasets();
    const interval = setInterval(() => {
      fetchJobs();
      fetchDatasets();
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  const handleUpload = async () => {
    const file = fileRef.current?.files?.[0];
    if (!file) {
      setUploadStatus("Please select a file first");
      return;
    }
    setUploading(true);
    setUploadStatus("Initiating upload...");
    try {
      const ext = file.name.split(".").pop() ?? "csv";
      const baseName = file.name.replace(`.${ext}`, "");
      const { dataset_id, upload_url, storage_path } = await initiateUpload(
        baseName,
        ext,
      );
      setUploadStatus("Uploading to storage...");
      await fetch(upload_url, {
        method: "PUT",
        body: file,
        headers: { "Content-Type": file.type || "application/octet-stream" },
      });
      setUploadStatus("Completing registration...");
      const result = await completeUpload(dataset_id, storage_path, file.size);
      console.log("complete result:", result);
      setUploadStatus("Upload complete!");
      listDatasets().then((data) => setDatasets(data ?? []));
      setTimeout(() => setUploadOpen(false), 1000);
    } catch (e) {
      setUploadStatus("Upload failed");
      console.error(e);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Mini Databricks</h1>
        <button
          className="text-sm text-gray-500 hover:text-gray-700"
          onClick={() => {
            localStorage.removeItem("token");
            navigate("/login");
          }}
        >
          Sign out
        </button>
      </div>

      <Tabs defaultValue="jobs">
        <TabsList className="mb-4">
          <TabsTrigger value="jobs">Jobs</TabsTrigger>
          <TabsTrigger value="datasets">Datasets</TabsTrigger>
        </TabsList>

        <TabsContent value="jobs">
          <div className="flex justify-between items-center mb-3">
            <span className="text-sm text-gray-500">
              Auto-refreshes every 3s
            </span>
          </div>
          <div className="flex flex-col gap-3">
            {jobs.length === 0 && (
              <p className="text-gray-500 text-sm">No jobs yet.</p>
            )}
            {jobs.map((job) => (
              <Card
                key={job.id}
                className="cursor-pointer hover:shadow-md transition-shadow"
                onClick={() => navigate(`/jobs/${job.id}`)}
              >
                <CardHeader className="pb-2">
                  <div className="flex justify-between items-center">
                    <CardTitle className="text-sm font-mono">
                      {job.id}
                    </CardTitle>
                    <JobStatusBadge state={job.state} />
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-gray-600">{job.entrypoint}</p>
                  <p className="text-xs text-gray-400 mt-1">
                    {new Date(job.created_at).toLocaleString()}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="datasets">
          <div className="flex justify-between items-center mb-3">
            <span className="text-sm text-gray-500">
              {datasets.length} datasets
            </span>
            <Button
              size="sm"
              onClick={() => {
                setUploadStatus("");
                setUploadOpen(true);
              }}
            >
              Upload Dataset
            </Button>
          </div>
          <div className="flex flex-col gap-3">
            {datasets.length === 0 && (
              <p className="text-gray-500 text-sm">No datasets yet.</p>
            )}
            {datasets.map((dataset) => (
              <Card key={dataset.id}>
                <CardHeader className="pb-2">
                  <div className="flex justify-between items-center">
                    <CardTitle className="text-sm">{dataset.name}</CardTitle>
                    <JobStatusBadge state={dataset.state} />
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-gray-400">
                    {(dataset.file_format ?? "").toUpperCase()} ·{" "}
                    {dataset.size_bytes} bytes ·{" "}
                    {new Date(dataset.created_at).toLocaleString()}
                  </p>
                  <p className="text-xs text-gray-300 font-mono mt-1">
                    {dataset.id}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={uploadOpen} onOpenChange={setUploadOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Upload Dataset</DialogTitle>
          </DialogHeader>
          <div className="flex flex-col gap-4 mt-2">
            <label className="flex flex-col gap-2">
              <span className="text-sm text-gray-600">Select a file</span>
              <input
                ref={fileRef}
                type="file"
                accept=".csv,.json,.parquet"
                className="text-sm border rounded px-3 py-2 cursor-pointer"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) setUploadStatus(`Selected: ${file.name}`);
                }}
              />
            </label>
            {uploadStatus && (
              <p className="text-sm text-blue-600">{uploadStatus}</p>
            )}
            <Button onClick={handleUpload} disabled={uploading}>
              {uploading ? "Uploading..." : "Upload"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
