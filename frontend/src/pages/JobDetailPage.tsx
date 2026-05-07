import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { JobStatusBadge } from "@/components/JobStatusBadge";
import {
  getJobProgress,
  listArtifacts,
  getDownloadUrl,
  cancelJob,
  type JobProgress,
  type Artifact,
} from "@/api/jobs";

export function JobDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [progress, setProgress] = useState<JobProgress | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);

  useEffect(() => {
    if (!id) return;
    const fetch = () => {
      getJobProgress(id).then(setProgress).catch(console.error);
      listArtifacts(id)
        .then((data) => setArtifacts(data ?? []))
        .catch(() => setArtifacts([]));
    };
    fetch();
    const interval = setInterval(fetch, 2000);
    return () => clearInterval(interval);
  }, [id]);

  const handleCancel = async () => {
    if (!id) return;
    await cancelJob(id);
  };

  const handleDownload = async (artifactId: string) => {
    if (!id) return;
    const url = await getDownloadUrl(id, artifactId);
    window.open(url, "_blank");
  };

  if (!progress) return <div className="p-6">Loading...</div>;

  const isActive = ["SUBMITTED", "QUEUED", "RUNNING"].includes(progress.state);

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="flex items-center gap-4 mb-6">
        <Button variant="outline" onClick={() => navigate("/jobs")}>
          ← Back
        </Button>
        <h1 className="text-xl font-bold">Job Detail</h1>
      </div>

      <Card className="mb-4">
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle className="text-sm font-mono">{id}</CardTitle>
            <div className="flex gap-2 items-center">
              <JobStatusBadge state={progress.state} />
              {isActive && (
                <Button variant="destructive" size="sm" onClick={handleCancel}>
                  Cancel
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div>
            <div className="flex justify-between text-sm mb-1">
              <span>Progress</span>
              <span>
                {progress.progress.done}/{progress.progress.total} tasks (
                {progress.progress.percent}%)
              </span>
            </div>
            <Progress value={progress.progress.percent} />
          </div>
          <div className="grid grid-cols-4 gap-2 text-center text-sm">
            <div className="bg-gray-50 rounded p-2">
              <div className="font-bold">{progress.progress.pending}</div>
              <div className="text-gray-500">Pending</div>
            </div>
            <div className="bg-yellow-50 rounded p-2">
              <div className="font-bold">{progress.progress.running}</div>
              <div className="text-gray-500">Running</div>
            </div>
            <div className="bg-green-50 rounded p-2">
              <div className="font-bold">{progress.progress.done}</div>
              <div className="text-gray-500">Done</div>
            </div>
            <div className="bg-red-50 rounded p-2">
              <div className="font-bold">{progress.progress.failed}</div>
              <div className="text-gray-500">Failed</div>
            </div>
          </div>
          {progress.started_at && (
            <p className="text-xs text-gray-400">
              Started: {new Date(progress.started_at).toLocaleString()}
              {progress.finished_at &&
                ` · Finished: ${new Date(progress.finished_at).toLocaleString()}`}
            </p>
          )}
        </CardContent>
      </Card>

      {artifacts.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Artifacts</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-2">
              {artifacts.map((artifact) => (
                <div
                  key={artifact.id}
                  className="flex justify-between items-center text-sm border rounded px-3 py-2"
                >
                  <div>
                    <span className="font-mono">{artifact.name}</span>
                    <span className="text-gray-400 ml-2">
                      {artifact.size_bytes} bytes
                    </span>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleDownload(artifact.id)}
                  >
                    Download
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
