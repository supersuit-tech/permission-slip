import { useState, useMemo } from "react";
import { ShieldAlert } from "lucide-react";
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
  onReview,
}: {
  approval: ApprovalSummary;
  agentDisplayName: string;
  onReview: () => void;
}) {
  const summary = buildSummary(
    approval.action.type,
    approval.action.parameters as Record<string, unknown>,
    null,
    null,
  );

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onReview}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") onReview();
      }}
      className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-blue-300 bg-blue-50 px-4 py-3 text-left text-sm text-blue-900 shadow-sm transition-colors hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-950/50 dark:text-blue-200 dark:hover:bg-blue-950/70"
      aria-label={`Review ${approval.action.type} from ${agentDisplayName}`}
    >
      <ShieldAlert className="size-5 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-medium">
          <span className="font-semibold">{approval.action.type}</span>
          <span className="text-blue-700 dark:text-blue-300">
            {" "}from {agentDisplayName}
          </span>
        </span>
        <span className="hidden text-xs opacity-75 sm:inline" title={summary}>
          {summary}
        </span>
        <RiskBadge level={approval.context.risk_level} />
        <CountdownBadge expiresAt={approval.expires_at} />
      </div>
      <span className="shrink-0 text-xs font-medium underline underline-offset-2 opacity-75">
        Review
      </span>
    </div>
  );
}

export function PendingApprovalsBanner() {
  const { approvals, isLoading } = useApprovals();
  const { agents } = useAgents();
  const [reviewingApproval, setReviewingApproval] =
    useState<ApprovalSummary | null>(null);

  const agentMap = useMemo(() => {
    const map = new Map<number, Agent>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);

  if (isLoading || approvals.length === 0) {
    return null;
  }

  return (
    <>
      <div
        className="space-y-2"
        role="status"
        aria-label="Pending approval requests"
      >
        {approvals.map((approval) => (
          <ApprovalBannerItem
            key={approval.approval_id}
            approval={approval}
            agentDisplayName={resolveAgentName(approval.agent_id, agentMap)}
            onReview={() => setReviewingApproval(approval)}
          />
        ))}
      </div>

      {reviewingApproval && (
        <ReviewApprovalDialog
          approval={reviewingApproval}
          agentDisplayName={resolveAgentName(
            reviewingApproval.agent_id,
            agentMap,
          )}
          open
          onOpenChange={(open) => {
            if (!open) setReviewingApproval(null);
          }}
        />
      )}
    </>
  );
}
