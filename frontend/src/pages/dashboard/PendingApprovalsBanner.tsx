import { useState, useMemo } from "react";
import { Bot } from "lucide-react";
import { useApprovals, type ApprovalSummary } from "@/hooks/useApprovals";
import { useAgents, type Agent } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import { buildSummary } from "@/components/ActionPreviewSummary";
import { CountdownBadge, RiskBadge } from "./approval-components";
import { ReviewApprovalDialog } from "./ReviewApprovalDialog";

function resolveAgentName(
  agentId: number,
  agentMap: Map<number, Agent>,
): string {
  const agent = agentMap.get(agentId);
  if (agent) return getAgentDisplayName(agent);
  return String(agentId);
}

function ApprovalBannerItem({
  approval,
  agentDisplayName,
}: {
  approval: ApprovalSummary;
  agentDisplayName: string;
}) {
  const [dialogOpen, setDialogOpen] = useState(false);

  const summary = buildSummary(
    approval.action.type,
    approval.action.parameters as Record<string, unknown>,
    null,
    null,
  );

  return (
    <>
      <button
        type="button"
        onClick={() => setDialogOpen(true)}
        className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-blue-300 bg-blue-50 px-4 py-3 text-left text-sm text-blue-900 shadow-sm transition-colors hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-950/50 dark:text-blue-200 dark:hover:bg-blue-950/70"
        aria-label={`Pending approval: ${approval.action.type} from ${agentDisplayName}`}
      >
        <Bot className="size-5 shrink-0" />
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
          <span className="font-medium">
            {approval.action.type}
          </span>
          <RiskBadge level={approval.context.risk_level} />
          <span className="text-muted-foreground truncate text-xs" title={summary}>
            {summary}
          </span>
          <span className="text-xs opacity-75">{agentDisplayName}</span>
          <CountdownBadge expiresAt={approval.expires_at} />
        </div>
        <span className="shrink-0 text-xs font-medium underline underline-offset-2 opacity-75">
          Review
        </span>
      </button>

      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName={agentDisplayName}
        open={dialogOpen}
        onOpenChange={setDialogOpen}
      />
    </>
  );
}

export function PendingApprovalsBanner() {
  const { approvals, isLoading, error, refetch } = useApprovals();
  const { agents } = useAgents();

  const agentMap = useMemo(() => {
    const map = new Map<number, Agent>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);

  if (isLoading) return null;

  if (error) {
    return (
      <div className="rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/50 dark:text-amber-200">
        Could not load pending approvals.{" "}
        <button
          type="button"
          className="underline"
          onClick={() => refetch()}
        >
          Retry
        </button>
      </div>
    );
  }

  if (approvals.length === 0) return null;

  return (
    <>
      <span className="sr-only" aria-live="polite" aria-atomic="true">
        {approvals.length} pending approval{approvals.length !== 1 ? "s" : ""}
      </span>
      <div className="space-y-2" aria-label="Pending approvals">
        {approvals.map((approval) => (
          <ApprovalBannerItem
            key={approval.approval_id}
            approval={approval}
            agentDisplayName={resolveAgentName(approval.agent_id, agentMap)}
          />
        ))}
      </div>
    </>
  );
}
