import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Loader2, ShieldCheck, ExternalLink } from "lucide-react";
import { LimitBadge } from "@/components/LimitBadge";
import { UpgradePrompt } from "@/components/UpgradePrompt";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardFooter,
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
import { useAgents, type Agent } from "@/hooks/useAgents";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import { useActionConfigMap } from "@/hooks/useActionConfigMap";
import { useResourceLimit } from "@/hooks/useResourceLimit";
import { getAgentDisplayName } from "@/lib/agents";

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

function resolveAgentName(
  agentId: number,
  agentMap: Map<number, Agent>,
): string {
  const agent = agentMap.get(agentId);
  return agent ? getAgentDisplayName(agent) : `Agent ${agentId}`;
}

function StandingApprovalSummaryRow({
  sa,
  config,
  agentMap,
}: {
  sa: StandingApproval;
  config: ActionConfiguration | undefined;
  agentMap: Map<number, Agent>;
}) {
  const agentName = resolveAgentName(sa.agent_id, agentMap);
  const href =
    config != null
      ? `/agents/${sa.agent_id}/connectors/${encodeURIComponent(config.connector_id)}`
      : `/agents/${sa.agent_id}`;

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="min-w-0">
          <p className="truncate">
            {config?.name ?? "Unknown configuration"}
          </p>
          {config && (
            <p className="text-muted-foreground truncate font-mono text-xs">
              {config.action_type}
            </p>
          )}
        </div>
      </TableCell>
      <TableCell className="text-sm">
        {config?.connector_id ?? "\u2014"}
      </TableCell>
      <TableCell className="max-w-[200px] truncate text-xs">
        {agentName}
      </TableCell>
      <TableCell>
        <span className="text-muted-foreground text-xs">On</span>
      </TableCell>
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
        Standing approvals are created from an agent&apos;s connector page, on each
        action configuration. Open a connector, then use the Standing Approval column
        to enable auto-approve with expiry you control.
      </p>
    </div>
  );
}

export function StandingApprovalsCard() {
  const { standingApprovals, isLoading, error, refetch } = useStandingApprovals();
  const { agents } = useAgents();

  const linkedActive = useMemo(
    () =>
      standingApprovals.filter(
        (sa) =>
          !!sa.source_action_configuration_id &&
          sa.status === "active",
      ),
    [standingApprovals],
  );

  const agentIds = useMemo(
    () => [...new Set(linkedActive.map((sa) => sa.agent_id))],
    [linkedActive],
  );
  const configMap = useActionConfigMap(agentIds);
  const agentMap = useMemo(() => {
    const map = new Map<number, Agent>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);

  const {
    max: maxStandingApprovals,
    current: standingApprovalCount,
    atLimit,
    hasData: hasBillingData,
  } = useResourceLimit("max_standing_approvals", standingApprovals.length);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CardTitle>Standing approvals</CardTitle>
          {hasBillingData && (
            <LimitBadge
              current={standingApprovalCount}
              max={maxStandingApprovals}
              resource="standing approvals"
            />
          )}
        </div>
        <p className="text-muted-foreground text-sm">
          Action configurations with an active standing approval. Manage expiry and
          revocation on each connector&apos;s configuration page.
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
        ) : linkedActive.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="overflow-x-auto">
            <div className="min-w-[640px] overflow-hidden rounded-lg">
              <Table>
                <TableHeader>
                  <TableRow className="border-none bg-muted/50 hover:bg-muted/50">
                    <TableHead>Configuration</TableHead>
                    <TableHead>Connector</TableHead>
                    <TableHead>Agent</TableHead>
                    <TableHead>Auto-approve</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead className="text-right">
                      <span className="sr-only">Open</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {linkedActive.map((sa) => {
                    const cid = sa.source_action_configuration_id;
                    const config =
                      cid != null ? configMap.get(cid) : undefined;
                    return (
                      <StandingApprovalSummaryRow
                        key={sa.standing_approval_id}
                        sa={sa}
                        config={config}
                        agentMap={agentMap}
                      />
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          </div>
        )}
      </CardContent>
      {atLimit ? (
        <CardFooter>
          <UpgradePrompt feature="Upgrade to create more standing approvals." />
        </CardFooter>
      ) : null}
    </Card>
  );
}
