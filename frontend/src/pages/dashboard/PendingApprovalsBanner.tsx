import { useState, useCallback, useMemo, useRef } from "react";
import { Bot, Shield } from "lucide-react";
import { useApprovals, type ApprovalSummary } from "@/hooks/useApprovals";
import {
  useStandingApprovalRequests,
  type StandingApprovalRequestSummary,
} from "@/hooks/useStandingApprovalRequests";
import { useAgents, type Agent } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import { Badge } from "@/components/ui/badge";
import { buildSummary } from "@/components/ActionPreviewSummary";
import { CountdownBadge, RiskBadge } from "./approval-components";
import { ReviewApprovalDialog } from "./ReviewApprovalDialog";
import { ReviewStandingApprovalRequestDialog } from "./ReviewStandingApprovalRequestDialog";

function resolveAgentName(
  agentId: number,
  agentMap: Map<number, Agent>,
): string {
  const agent = agentMap.get(agentId);
  if (agent) return getAgentDisplayName(agent);
  return String(agentId);
}

function RuleProposalBannerItem({
  request,
  agentDisplayName,
  onOpenDialog,
}: {
  request: StandingApprovalRequestSummary;
  agentDisplayName: string;
  onOpenDialog: (request: StandingApprovalRequestSummary, agentDisplayName: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onOpenDialog(request, agentDisplayName)}
      className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-violet-500/30 bg-violet-500/5 px-4 py-3 text-left text-sm text-foreground shadow-sm transition-colors hover:bg-violet-500/10"
      aria-label={`Rule proposal: ${request.action_type} from ${agentDisplayName}`}
    >
      <Shield className="size-5 shrink-0 text-violet-600 dark:text-violet-400" />
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-medium">{request.action_type}</span>
        <Badge variant="secondary" className="text-xs">
          Rule proposal
        </Badge>
        <span className="text-xs opacity-75">{agentDisplayName}</span>
      </div>
      <span className="shrink-0 text-xs font-medium underline underline-offset-2 opacity-75">
        Review
      </span>
    </button>
  );
}

function ApprovalBannerItem({
  approval,
  agentDisplayName,
  onOpenDialog,
}: {
  approval: ApprovalSummary;
  agentDisplayName: string;
  onOpenDialog: (approval: ApprovalSummary, agentDisplayName: string) => void;
}) {
  const summary = buildSummary(
    approval.action.type,
    approval.action.parameters as Record<string, unknown>,
    null,
    null,
    undefined,
    approval.resource_details as Record<string, unknown> | undefined,
  );

  return (
    <button
      type="button"
      onClick={() => onOpenDialog(approval, agentDisplayName)}
      className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-info/30 bg-info/5 px-4 py-3 text-left text-sm text-foreground shadow-sm transition-colors hover:bg-info/10"
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
  );
}

export function PendingApprovalsBanner() {
  const { approvals, isLoading, error, refetch } = useApprovals();
  const {
    requests: ruleProposals,
    isLoading: rulesLoading,
    error: rulesError,
    refetch: refetchRules,
  } = useStandingApprovalRequests();
  const { agents } = useAgents();

  // Dialog state is lifted here so it survives the approval being removed
  // from the pending list (e.g. after approve + query invalidation).
  const [dialogOpen, setDialogOpen] = useState(false);
  const activeApprovalRef = useRef<ApprovalSummary | null>(null);
  const activeAgentNameRef = useRef<string>("");

  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const activeRuleRef = useRef<StandingApprovalRequestSummary | null>(null);
  const activeRuleAgentNameRef = useRef<string>("");

  const agentMap = useMemo(() => {
    const map = new Map<number, Agent>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);

  const handleOpenDialog = useCallback(
    (approval: ApprovalSummary, agentDisplayName: string) => {
      activeApprovalRef.current = approval;
      activeAgentNameRef.current = agentDisplayName;
      setDialogOpen(true);
    },
    [],
  );

  const handleDialogChange = useCallback((nextOpen: boolean) => {
    setDialogOpen(nextOpen);
    if (!nextOpen) {
      activeApprovalRef.current = null;
      activeAgentNameRef.current = "";
    }
  }, []);

  const handleOpenRuleDialog = useCallback(
    (request: StandingApprovalRequestSummary, agentDisplayName: string) => {
      activeRuleRef.current = request;
      activeRuleAgentNameRef.current = agentDisplayName;
      setRuleDialogOpen(true);
    },
    [],
  );

  const handleRuleDialogChange = useCallback((nextOpen: boolean) => {
    setRuleDialogOpen(nextOpen);
    if (!nextOpen) {
      activeRuleRef.current = null;
      activeRuleAgentNameRef.current = "";
    }
  }, []);

  if (isLoading && rulesLoading) return null;

  if (error && rulesError) {
    return (
      <div className="rounded-lg border border-warning/30 bg-warning/5 px-4 py-3 text-sm text-foreground">
        Could not load pending approvals.{" "}
        <button
          type="button"
          className="underline"
          onClick={() => {
            refetch();
            refetchRules();
          }}
        >
          Retry
        </button>
      </div>
    );
  }

  const totalPending = approvals.length + ruleProposals.length;
  if (totalPending === 0 && !dialogOpen && !ruleDialogOpen) return null;

  return (
    <>
      <span className="sr-only" aria-live="polite" aria-atomic="true">
        {totalPending} pending item{totalPending !== 1 ? "s" : ""}
      </span>
      {(ruleProposals.length > 0 || approvals.length > 0) && (
        <div className="space-y-2" aria-label="Pending approvals and rule proposals">
          {ruleProposals.map((request) => (
            <RuleProposalBannerItem
              key={request.request_id}
              request={request}
              agentDisplayName={resolveAgentName(request.agent_id, agentMap)}
              onOpenDialog={handleOpenRuleDialog}
            />
          ))}
          {approvals.map((approval) => (
            <ApprovalBannerItem
              key={approval.approval_id}
              approval={approval}
              agentDisplayName={resolveAgentName(approval.agent_id, agentMap)}
              onOpenDialog={handleOpenDialog}
            />
          ))}
        </div>
      )}

      {/* Dialog rendered at banner level so it survives approval list changes */}
      {activeApprovalRef.current && (
        <ReviewApprovalDialog
          approval={activeApprovalRef.current}
          agentDisplayName={activeAgentNameRef.current}
          open={dialogOpen}
          onOpenChange={handleDialogChange}
        />
      )}
      {activeRuleRef.current && (
        <ReviewStandingApprovalRequestDialog
          request={activeRuleRef.current}
          agentDisplayName={activeRuleAgentNameRef.current}
          open={ruleDialogOpen}
          onOpenChange={handleRuleDialogChange}
        />
      )}
    </>
  );
}
