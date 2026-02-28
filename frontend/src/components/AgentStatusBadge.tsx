import { Badge } from "@/components/ui/badge";

const STATUS_CONFIG: Record<
  string,
  { variant: "default" | "secondary" | "destructive"; label: string }
> = {
  registered: { variant: "default", label: "Active" },
  pending: { variant: "secondary", label: "Pending" },
  deactivated: { variant: "destructive", label: "Deactivated" },
};

export function AgentStatusBadge({ status }: { status: string }) {
  const config = STATUS_CONFIG[status] ?? {
    variant: "secondary" as const,
    label: status,
  };

  return (
    <Badge variant={config.variant} className="rounded-full">
      {config.label}
    </Badge>
  );
}
