import { useState, useCallback, useMemo, useRef } from "react";
import { Bot, Shield } from "lucide-react";
import { toast } from "sonner";
import { useApprovals, type ApprovalSummary } from "@/hooks/useApprovals";
import {
  useStandingApprovalRequests,
  type StandingApprovalRequestSummary,
} from "@/hooks/useStandingApprovalRequests";
import { useAgents, type Agent } from "@/hooks/useAgents";
import { useDenyApproval } from "@/hooks/useDenyApproval";
import { useDenyStandingApprovalRequest } from "@/hooks/useDenyStandingApprovalRequest";
import { useDenyAllApprovals } from "@/hooks/useDenyAllApprovals";
import { getAgentDisplayName } from "@/lib/agents";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { buildSummary } from "@/components/ActionPreviewSummary";
import { CountdownBadge, RiskBadge } from "./approval-components";
import { ReviewApprovalDialog } from "./ReviewApprovalDialog";
import { ReviewBulkApprovalDialog } from "./ReviewBulkApprovalDialog";
import { ReviewStandingApprovalRequestDialog } from "./ReviewStandingApprovalRequestDialog";
import { DeclineRequestButton } from "./DeclineRequestButton";
import { DeclineAllApprovalsDialog } from "./DeclineAllApprovalsDialog";

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
  onDecline,
  isDeclining,
}: {
  request: StandingApprovalRequestSummary;
  agentDisplayName: string;
  onOpenDialog: (request: StandingApprovalRequestSummary, agentDisplayName: string) => void;
  onDecline: (requestId: string) => Promise<void>;
  isDeclining: boolean;
}) {
  return (
    <div className="flex w-full items-center gap-2 rounded-lg border border-violet-500/30 bg-violet-500/5 px-4 py-3 text-sm text-foreground shadow-sm transition-colors hover:bg-violet-500/10">
      <button
        type="button"
        onClick={() => onOpenDialog(request, agentDisplayName)}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 border-0 bg-transparent p-0 text-left"
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
      <DeclineRequestButton
        ariaLabel="Decline request"
        disabled={isDeclining}
        onDecline={() => onDecline(request.request_id)}
      />
    </div>
  );
}

function BulkApprovalBannerItem({
  bulkGroupId,
  actionType,
  itemCount,
  agentDisplayName,
  expiresAt,
  onOpenDialog,
}: {
  bulkGroupId: string;
  actionType: string;
  itemCount: number;
  agentDisplayName: string;
  expiresAt: string;
  onOpenDialog: (bulkGroupId: string, agentDisplayName: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onOpenDialog(bulkGroupId, agentDisplayName)}
      className="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-info/30 bg-info/5 px-4 py-3 text-left text-sm text-foreground shadow-sm transition-colors hover:bg-info/10"
      aria-label={`Bulk approval: ${itemCount} ${actionType} from ${agentDisplayName}`}
    >
      <Bot className="size-5 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1">
        <span className="font-medium">{actionType}</span>
        <Badge variant="secondary" className="text-xs">
          {itemCount} items
        </Badge>
        <span className="text-xs opacity-75">{agentDisplayName}</span>
        <CountdownBadge expiresAt={expiresAt} />
      </div>
      <span className="shrink-0 text-xs font-medium underline underline-offset-2 opacity-75">
        Review batch
      </span>
    </button>
  );
}

function ApprovalBannerItem({
  approval,
  agentDisplayName,
  onOpenDialog,
  onDecline,
  isDeclining,
}: {
  approval: ApprovalSummary;
  agentDisplayName: string;
  onOpenDialog: (approval: ApprovalSummary, agentDisplayName: string) => void;
  onDecline: (approvalId: string) => Promise<void>;
  isDeclining: boolean;
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
    <div className="flex w-full items-center gap-2 rounded-lg border border-info/30 bg-info/5 px-4 py-3 text-sm text-foreground shadow-sm transition-colors hover:bg-info/10">
      <button
        type="button"
        onClick={() => onOpenDialog(approval, agentDisplayName)}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 border-0 bg-transparent p-0 text-left"
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
      <DeclineRequestButton
        ariaLabel="Decline request"
        disabled={isDeclining}
        onDecline={() => onDecline(approval.approval_id)}
      />
    </div>
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
  const { denyApproval, isPending: isDenyingApproval } = useDenyApproval();
  const { denyRequest, isPending: isDenyingRule } = useDenyStandingApprovalRequest();
  const { denyAllApprovals, isPending: isDenyingAll } = useDenyAllApprovals();

  const [declineAllOpen, setDeclineAllOpen] = useState(false);
  const [dismissedApprovalIds, setDismissedApprovalIds] = useState<Set<string>>(
    () => new Set(),
  );
  const [dismissedRuleIds, setDismissedRuleIds] = useState<Set<string>>(
    () => new Set(),
  );

  // Dialog state is lifted here so it survives the approval being removed
  // from the pending list (e.g. after approve + query invalidation).
  const [dialogOpen, setDialogOpen] = useState(false);
  const activeApprovalRef = useRef<ApprovalSummary | null>(null);
  const activeAgentNameRef = useRef<string>("");

  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const activeRuleRef = useRef<StandingApprovalRequestSummary | null>(null);
  const activeRuleAgentNameRef = useRef<string>("");

  const [bulkDialogOpen, setBulkDialogOpen] = useState(false);
  const activeBulkGroupIdRef = useRef<string>("");
  const activeBulkAgentNameRef = useRef<string>("");

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

  const { standaloneApprovals, bulkGroups } = useMemo(() => {
    const groups = new Map<
      string,
      { actionType: string; itemCount: number; expiresAt: string; agentId: number }
    >();
    const standalone: ApprovalSummary[] = [];
    for (const approval of approvals) {
      const gid = approval.bulk_group_id;
      if (gid) {
        const existing = groups.get(gid);
        if (!existing) {
          groups.set(gid, {
            actionType: approval.action.type,
            itemCount: 1,
            expiresAt: approval.expires_at,
            agentId: approval.agent_id,
          });
        } else {
          existing.itemCount += 1;
        }
      } else {
        standalone.push(approval);
      }
    }
    return {
      standaloneApprovals: standalone,
      bulkGroups: [...groups.entries()].map(([id, meta]) => ({ id, ...meta })),
    };
  }, [approvals]);

  const visibleStandaloneApprovals = useMemo(
    () =>
      standaloneApprovals.filter(
        (approval) => !dismissedApprovalIds.has(approval.approval_id),
      ),
    [standaloneApprovals, dismissedApprovalIds],
  );

  const visibleRuleProposals = useMemo(
    () =>
      ruleProposals.filter(
        (request) => !dismissedRuleIds.has(request.request_id),
      ),
    [ruleProposals, dismissedRuleIds],
  );

  const handleOpenBulkDialog = useCallback(
    (bulkGroupId: string, agentDisplayName: string) => {
      activeBulkGroupIdRef.current = bulkGroupId;
      activeBulkAgentNameRef.current = agentDisplayName;
      setBulkDialogOpen(true);
    },
    [],
  );

  const handleBulkDialogChange = useCallback((nextOpen: boolean) => {
    setBulkDialogOpen(nextOpen);
    if (!nextOpen) {
      activeBulkGroupIdRef.current = "";
      activeBulkAgentNameRef.current = "";
    }
  }, []);

  const handleDeclineApproval = useCallback(
    async (approvalId: string) => {
      setDismissedApprovalIds((prev) => new Set(prev).add(approvalId));
      try {
        await denyApproval(approvalId);
        toast.success("Request declined");
      } catch {
        setDismissedApprovalIds((prev) => {
          const next = new Set(prev);
          next.delete(approvalId);
          return next;
        });
        throw new Error("Failed to decline request");
      }
    },
    [denyApproval],
  );

  const handleDeclineRule = useCallback(
    async (requestId: string) => {
      setDismissedRuleIds((prev) => new Set(prev).add(requestId));
      try {
        await denyRequest(requestId);
        toast.success("Rule proposal declined");
      } catch {
        setDismissedRuleIds((prev) => {
          const next = new Set(prev);
          next.delete(requestId);
          return next;
        });
        throw new Error("Failed to decline rule proposal");
      }
    },
    [denyRequest],
  );

  const handleConfirmDeclineAll = useCallback(async () => {
    try {
      const result = await denyAllApprovals();
      setDeclineAllOpen(false);
      toast.success(
        result?.denied_count === 1
          ? "Declined 1 request"
          : `Declined ${result?.denied_count ?? 0} requests`,
      );
    } catch {
      toast.error("Failed to decline pending requests. Please try again.");
    }
  }, [denyAllApprovals]);

  if (isLoading || rulesLoading) return null;

  if (error || rulesError) {
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

  const totalPending =
    visibleStandaloneApprovals.length +
    bulkGroups.length +
    visibleRuleProposals.length;
  if (totalPending === 0 && !dialogOpen && !ruleDialogOpen && !bulkDialogOpen) {
    return null;
  }

  return (
    <>
      <span className="sr-only" aria-live="polite" aria-atomic="true">
        {totalPending} pending item{totalPending !== 1 ? "s" : ""}
      </span>
      {(visibleRuleProposals.length > 0 ||
        visibleStandaloneApprovals.length > 0 ||
        bulkGroups.length > 0) && (
        <div className="space-y-2" aria-label="Pending approvals and rule proposals">
          {visibleStandaloneApprovals.length > 0 && (
            <div className="flex justify-end">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isDenyingAll}
                onClick={() => setDeclineAllOpen(true)}
              >
                Decline all ({visibleStandaloneApprovals.length})
              </Button>
            </div>
          )}
          {visibleRuleProposals.map((request) => (
            <RuleProposalBannerItem
              key={request.request_id}
              request={request}
              agentDisplayName={resolveAgentName(request.agent_id, agentMap)}
              onOpenDialog={handleOpenRuleDialog}
              onDecline={handleDeclineRule}
              isDeclining={isDenyingRule}
            />
          ))}
          {bulkGroups.map((group) => (
            <BulkApprovalBannerItem
              key={group.id}
              bulkGroupId={group.id}
              actionType={group.actionType}
              itemCount={group.itemCount}
              agentDisplayName={resolveAgentName(group.agentId, agentMap)}
              expiresAt={group.expiresAt}
              onOpenDialog={handleOpenBulkDialog}
            />
          ))}
          {visibleStandaloneApprovals.map((approval) => (
            <ApprovalBannerItem
              key={approval.approval_id}
              approval={approval}
              agentDisplayName={resolveAgentName(approval.agent_id, agentMap)}
              onOpenDialog={handleOpenDialog}
              onDecline={handleDeclineApproval}
              isDeclining={isDenyingApproval}
            />
          ))}
        </div>
      )}

      <DeclineAllApprovalsDialog
        open={declineAllOpen}
        onOpenChange={setDeclineAllOpen}
        pendingCount={visibleStandaloneApprovals.length}
        onConfirm={handleConfirmDeclineAll}
        isPending={isDenyingAll}
      />

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
      {activeBulkGroupIdRef.current && (
        <ReviewBulkApprovalDialog
          bulkGroupId={activeBulkGroupIdRef.current}
          agentDisplayName={activeBulkAgentNameRef.current}
          open={bulkDialogOpen}
          onOpenChange={handleBulkDialogChange}
        />
      )}
    </>
  );
}
