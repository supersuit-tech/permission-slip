import { Badge } from "@/components/ui/badge";

const STATUS_CONFIG: Record<
  string,
  { dotColor: string; label: string }
> = {
  registered: { dotColor: "bg-success", label: "Active" },
  pending: { dotColor: "bg-warning", label: "Pending" },
  deactivated: { dotColor: "bg-destructive", label: "Deactivated" },
};

export function AgentStatusBadge({ status }: { status: string }) {
  const config = STATUS_CONFIG[status] ?? {
    dotColor: "bg-muted-foreground",
    label: status,
  };

  return (
    <Badge variant="outline" className="gap-1.5 rounded-full">
      <span className={`size-2 rounded-full ${config.dotColor}`} />
      {config.label}
    </Badge>
  );
}
