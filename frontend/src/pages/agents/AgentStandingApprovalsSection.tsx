import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Loader2, ShieldCheck, ExternalLink } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import {
  useStandingApprovals,
  type StandingApproval,
} from "@/hooks/useStandingApprovals";

function formatExpiresIn(expiresAt: string | null | undefined): string {
  if (expiresAt === null) return "Never";
  if (!expiresAt) return "\u2014";

  const exp = new Date(expiresAt);
  if (Number.isNaN(exp.getTime())) return "\u2014";

  const now = new Date();
  const diffMs = exp.getTime() - now.getTime();

  if (diffMs <= 0) return "Expired";

  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  if (diffHours < 24) return `${diffHours}h`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d`;
}

function connectorIdFromActionType(actionType: string): string {
  const dot = actionType.indexOf(".");
  return dot === -1 ? actionType : actionType.slice(0, dot);
}

function StandingApprovalSummaryRow({ sa }: { sa: StandingApproval }) {
  const connectorId = connectorIdFromActionType(sa.action_type);
  const href = `/agents/${sa.agent_id}/connectors/${encodeURIComponent(connectorId)}`;

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="min-w-0">
          <p className="truncate">{sa.name ?? sa.action_type}</p>
          <p className="text-muted-foreground truncate font-mono text-xs">
            {sa.action_type}
          </p>
        </div>
      </TableCell>
      <TableCell className="text-sm">{connectorId}</TableCell>
      <TableCell>{formatExpiresIn(sa.expires_at)}</TableCell>
      <TableCell className="text-right">
        <Link
          to={href}
          className="text-primary inline-flex items-center gap-1 text-sm underline-offset-4 hover:underline"
        >
          Manage
          <ExternalLink className="size-3.5" aria-hidden />
        </Link>
      </TableCell>
    </TableRow>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <ShieldCheck className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No active standing approvals
      </p>
      <p className="text-muted-foreground mb-4 max-w-md text-xs leading-relaxed">
        Standing approvals are created from an agent&apos;s connector page.
        Open a connector, then add a standing approval to enable auto-approve
        with expiry you control.
      </p>
    </div>
  );
}

interface AgentStandingApprovalsSectionProps {
  agentId: number;
}

export function AgentStandingApprovalsSection({
  agentId,
}: AgentStandingApprovalsSectionProps) {
  const { standingApprovals, isLoading, error, refetch } = useStandingApprovals();

  const activeForAgent = useMemo(
    () =>
      standingApprovals.filter(
        (sa) => sa.agent_id === agentId && sa.status === "active",
      ),
    [standingApprovals, agentId],
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CardTitle>Standing approvals</CardTitle>
        </div>
        <p className="text-muted-foreground text-sm">
          Pre-authorized rules that auto-approve matching requests. Manage
          expiry and revocation on each connector&apos;s standing approvals page.
        </p>
      </CardHeader>
      <CardContent className="px-3 md:px-6">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <p className="text-destructive mb-2 text-sm">{error}</p>
            <button
              type="button"
              className="text-primary text-sm underline-offset-4 hover:underline"
              onClick={() => refetch()}
            >
              Retry
            </button>
          </div>
        ) : activeForAgent.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="overflow-x-auto">
            <div className="min-w-[480px] overflow-hidden rounded-lg">
              <Table>
                <TableHeader>
                  <TableRow className="border-none bg-muted/50 hover:bg-muted/50">
                    <TableHead>Rule</TableHead>
                    <TableHead>Connector</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead className="text-right">
                      <span className="sr-only">Open</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {activeForAgent.map((sa) => (
                    <StandingApprovalSummaryRow
                      key={sa.standing_approval_id}
                      sa={sa}
                    />
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
