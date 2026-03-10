import { useState, useMemo } from "react";
import { Inbox, Bot } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
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

function ApprovalRow({
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
    <button
      type="button"
      aria-label={`Review ${approval.action.type} from ${agentDisplayName}`}
      className="hover:bg-muted/50 flex w-full items-center gap-3 border-b px-3 py-3 text-left transition-colors last:border-b-0 sm:px-4"
      onClick={onReview}
    >
      <div className="bg-muted shrink-0 rounded-full p-1.5">
        <Bot className="text-muted-foreground size-4" aria-hidden="true" />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">
            {approval.action.type}
          </span>
          <RiskBadge level={approval.context.risk_level} />
        </div>
        <p className="text-muted-foreground truncate text-xs" title={summary}>
          {summary}
        </p>
        <div className="text-muted-foreground flex items-center gap-2 text-xs">
          <span className="truncate">{agentDisplayName}</span>
          <span aria-hidden="true">&middot;</span>
          <CountdownBadge expiresAt={approval.expires_at} />
        </div>
      </div>

      <span className="text-primary hidden shrink-0 text-sm font-medium sm:inline">
        Review &rarr;
      </span>
    </button>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <Inbox className="text-muted-foreground mb-3 size-10" aria-hidden="true" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No pending requests
      </p>
      <p className="text-muted-foreground max-w-xs text-xs">
        When an agent needs permission to perform an action, it will appear here
        for your review. Requests expire if not reviewed in time.
      </p>
    </div>
  );
}

export function PendingApprovalsCard() {
  const { approvals, isLoading, error, refetch } = useApprovals();
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

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pending Approvals</CardTitle>
        {approvals.length > 0 && (
          <CardDescription>
            {approvals.length} request{approvals.length !== 1 ? "s" : ""}{" "}
            awaiting your review
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="px-3 md:px-6">
        {isLoading ? (
          <div className="overflow-hidden rounded-lg border" role="status" aria-label="Loading approvals">
            {[0, 1, 2].map((i) => (
              <div key={i} className="flex items-center gap-3 border-b px-3 py-3 last:border-b-0 sm:px-4">
                <Skeleton className="size-7 shrink-0 rounded-full" />
                <div className="min-w-0 flex-1 space-y-1.5">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-20" />
                </div>
                <Skeleton className="hidden h-4 w-16 sm:block" />
              </div>
            ))}
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <p className="text-destructive mb-2 text-sm">{error}</p>
            <Button variant="ghost" size="sm" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        ) : approvals.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="overflow-hidden rounded-lg border">
            {approvals.map((approval) => (
              <ApprovalRow
                key={approval.approval_id}
                approval={approval}
                agentDisplayName={resolveAgentName(
                  approval.agent_id,
                  agentMap,
                )}
                onReview={() => setReviewingApproval(approval)}
              />
            ))}
          </div>
        )}
      </CardContent>

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
    </Card>
  );
}
