import { Badge } from "@/components/ui/badge";

const stateColors: Record<string, string> = {
  SUBMITTED: "bg-gray-100 text-gray-700",
  QUEUED: "bg-blue-100 text-blue-700",
  RUNNING: "bg-yellow-100 text-yellow-700",
  SUCCEEDED: "bg-green-100 text-green-700",
  FAILED: "bg-red-100 text-red-700",
  CANCELLED: "bg-orange-100 text-orange-700",
};

export function JobStatusBadge({ state }: { state: string }) {
  return (
    <Badge className={stateColors[state] ?? "bg-gray-100 text-gray-700"}>
      {state}
    </Badge>
  );
}
