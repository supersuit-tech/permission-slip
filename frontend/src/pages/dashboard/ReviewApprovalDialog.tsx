import { useState, useCallback, useEffect, useMemo } from "react";
import { Loader2, Clock, Bot, AlertTriangle, CheckCircle, XCircle, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useApproveApproval } from "@/hooks/useApproveApproval";
import type { ApproveResult } from "@/hooks/useApproveApproval";
import { useDenyApproval } from "@/hooks/useDenyApproval";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { useAgents } from "@/hooks/useAgents";
import { useStandingApprovals } from "@/hooks/useStandingApprovals";
import { ActionPreviewSummary } from "@/components/ActionPreviewSummary";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { CreateStandingApprovalDialog } from "./CreateStandingApprovalDialog";
import {
  useCountdown,
  formatCountdown,
  urgencyColor,
  RiskBadge,
} from "./approval-components";

/** Auto-close delay (ms) after a successful approval. */
const SUCCESS_AUTO_CLOSE_MS = 3_000;

interface ReviewApprovalDialogProps {
  approval: ApprovalSummary;
  agentDisplayName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function ExecutionStatusBanner({ result }: { result: ApproveResult }) {
  const isSuccess = result.execution_status === "success";
  const isError = result.execution_status === "error";
  const isPending = result.execution_status === "pending";

  if (isPending) {
    return (
      <div className="flex flex-col items-center gap-4">
        <div className="rounded-full bg-blue-100 p-3 dark:bg-blue-900/30">
          <Loader2 className="size-8 animate-spin text-blue-600 dark:text-blue-400" aria-hidden="true" />
        </div>
        <p className="text-lg font-semibold">Request Approved</p>
        <p className="text-muted-foreground text-sm text-center">
          The action is being executed…
        </p>
      </div>
    );
  }

  if (isError) {
    const errorDetail = result.execution_result
      ? (result.execution_result as Record<string, unknown>)["execution_error"]
      : undefined;
    return (
      <div className="flex flex-col items-center gap-4">
        <div className="rounded-full bg-red-100 p-3 dark:bg-red-900/30">
          <XCircle className="size-8 text-red-600 dark:text-red-400" aria-hidden="true" />
        </div>
        <p className="text-lg font-semibold">Execution Failed</p>
        <p className="text-muted-foreground text-sm text-center">
          The action was approved but execution failed.
          {typeof errorDetail === "string" && ` ${errorDetail}`}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center gap-4">
      <div className="rounded-full bg-green-100 p-3 dark:bg-green-900/30">
        <CheckCircle className="size-8 text-green-600 dark:text-green-400" aria-hidden="true" />
      </div>
      <p className="text-lg font-semibold">
        {isSuccess ? "Action Executed Successfully" : "Request Approved"}
      </p>
      <p className="text-muted-foreground text-sm text-center">
        {isSuccess
          ? "The action has been executed. The agent has been notified."
          : "The action was approved. The agent has been notified."}
      </p>
    </div>
  );
}

/**
 * Derive initial constraints from an approval's action parameters.
 * Each parameter becomes a "fixed" constraint by default.
 */
function deriveConstraintsFromParams(
  parameters: Record<string, unknown>,
): Record<string, unknown> {
  const constraints: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(parameters)) {
    // Keep the raw value — CreateStandingApprovalDialog's
    // initConstraintsFromRecord handles type coercion.
    constraints[key] = value ?? "";
  }
  return constraints;
}

export function ReviewApprovalDialog({
  approval,
  agentDisplayName,
  open,
  onOpenChange,
}: ReviewApprovalDialogProps) {
  const [approveResult, setApproveResult] = useState<ApproveResult | null>(null);
  const [standingDialogOpen, setStandingDialogOpen] = useState(false);
  const [standingApprovalCreated, setStandingApprovalCreated] = useState(false);
  const [autoCloseBlocked, setAutoCloseBlocked] = useState(false);
  const [pendingAction, setPendingAction] = useState<"approve" | "alwaysAllow" | null>(null);
  const isApproved = approveResult !== null;
  const { approveApproval } = useApproveApproval();
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const { schema, actionName, displayTemplate } = useActionSchema(approval.action.type);
  const remaining = useCountdown(approval.expires_at);
  const isExpired = remaining <= 0;
  const isBusy = pendingAction !== null || isDenying;

  // Data for "Always Allow This"
  const { agents } = useAgents();
  // API defaults to status=active, so only active standing approvals are returned
  const { standingApprovals, isLoading: standingApprovalsLoading } = useStandingApprovals();
  const params = approval.action.parameters as Record<string, unknown>;
  const hasParams = Object.keys(params).length > 0;
  const hasExistingStandingApproval = useMemo(
    () =>
      standingApprovals.some(
        (sa) =>
          sa.agent_id === approval.agent_id &&
          sa.action_type === approval.action.type,
      ),
    [standingApprovals, approval.agent_id, approval.action.type],
  );
  const showAlwaysAllow = hasParams && !standingApprovalsLoading && !hasExistingStandingApproval;

  // Auto-close dialog after successful approval (unless user is creating standing approval)
  useEffect(() => {
    if (!isApproved || autoCloseBlocked) return;
    const timer = setTimeout(() => onOpenChange(false), SUCCESS_AUTO_CLOSE_MS);
    return () => clearTimeout(timer);
  }, [isApproved, autoCloseBlocked, onOpenChange]);

  const handleApprove = useCallback(async () => {
    setPendingAction("approve");
    try {
      const result = await approveApproval(approval.approval_id);
      setApproveResult(result);
    } catch {
      toast.error("Failed to approve request. Please try again.");
    } finally {
      setPendingAction(null);
    }
  }, [approveApproval, approval.approval_id]);

  const handleDeny = useCallback(async () => {
    try {
      await denyApproval(approval.approval_id);
      toast.success("Request denied");
      onOpenChange(false);
    } catch {
      toast.error("Failed to deny request. Please try again.");
    }
  }, [denyApproval, approval.approval_id, onOpenChange]);

  const handleAlwaysAllow = useCallback(async () => {
    setPendingAction("alwaysAllow");
    try {
      const result = await approveApproval(approval.approval_id);
      setApproveResult(result);
      setAutoCloseBlocked(true);
      if (result.execution_status !== "error") {
        setStandingDialogOpen(true);
      }
    } catch {
      toast.error("Failed to approve request. Please try again.");
    } finally {
      setPendingAction(null);
    }
  }, [approveApproval, approval.approval_id]);

  function handleStandingDialogChange(nextOpen: boolean) {
    setStandingDialogOpen(nextOpen);
    if (!nextOpen && !standingApprovalCreated) {
      // SA wizard was dismissed without creating — unblock auto-close
      // so the user can click Done or let the dialog close naturally.
      setAutoCloseBlocked(false);
    }
  }

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      setApproveResult(null);
      setStandingApprovalCreated(false);
      setAutoCloseBlocked(false);
      setPendingAction(null);
    }
    onOpenChange(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Review Approval Request</DialogTitle>
        </DialogHeader>

        {isApproved ? (
          <div className="space-y-6 py-4" role="status" aria-live="polite">
            <ExecutionStatusBanner result={approveResult} />
            {standingApprovalCreated && (
              <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800 dark:bg-green-950/30">
                <ShieldCheck className="size-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden="true" />
                <p className="text-sm text-green-800 dark:text-green-300">
                  Standing approval created. Future matching requests will be auto-approved.
                </p>
              </div>
            )}
          </div>
        ) : (
          <div className="space-y-4 sm:space-y-6">
            {/* Agent */}
            <div className="flex items-center gap-3">
              <div className="bg-muted rounded-full p-2">
                <Bot className="text-muted-foreground size-5" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{agentDisplayName}</p>
                {agentDisplayName !== String(approval.agent_id) && (
                  <p className="text-muted-foreground truncate font-mono text-xs">
                    {approval.agent_id}
                  </p>
                )}
              </div>
            </div>

            {/* Action & Risk */}
            <div className="bg-muted/50 space-y-3 rounded-lg border p-3 sm:p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex min-w-0 flex-wrap items-center gap-2">
                  <div className="min-w-0">
                    <span className="block truncate text-sm font-semibold">
                      {actionName ?? approval.action.type}
                    </span>
                    {actionName && (
                      <span className="text-muted-foreground block truncate font-mono text-xs">
                        {approval.action.type}
                      </span>
                    )}
                  </div>
                  <RiskBadge level={approval.context.risk_level} />
                </div>
                <span
                  aria-label={isExpired ? "Request expired" : `Time remaining: ${formatCountdown(remaining)}`}
                  className={`inline-flex shrink-0 items-center gap-1 text-xs font-medium ${urgencyColor(remaining)}`}
                >
                  <Clock className="size-3" aria-hidden="true" />
                  {isExpired ? "Expired" : formatCountdown(remaining)}
                </span>
              </div>

              {approval.context.description && (
                <p className="text-muted-foreground text-sm">
                  {approval.context.description}
                </p>
              )}

              {/* Action preview summary */}
              <div className="border-t pt-3">
                <ActionPreviewSummary
                  actionType={approval.action.type}
                  parameters={approval.action.parameters as Record<string, unknown>}
                  schema={schema}
                  actionName={actionName}
                  displayTemplate={displayTemplate}
                  resourceDetails={approval.resource_details as Record<string, unknown> | undefined}
                />
              </div>
            </div>

            {/* Parameters (collapsible — summary above covers the essentials) */}
            {Object.keys(approval.action.parameters).length > 0 && (
              <details className="group">
                <summary className="flex cursor-pointer items-center gap-1 text-sm font-medium select-none">
                  <span className="text-muted-foreground transition-transform group-open:rotate-90" aria-hidden="true">&#9656;</span>
                  Parameters
                </summary>
                <div className="bg-muted/50 mt-2 overflow-x-auto rounded-lg border p-3 sm:p-4">
                  <SchemaParameterDetails
                    parameters={approval.action.parameters as Record<string, unknown>}
                    schema={schema}
                  />
                </div>
              </details>
            )}

            {/* Expired notice */}
            {isExpired && (
              <div role="alert" className="bg-muted/50 flex items-start gap-2 rounded-lg border p-3">
                <Clock className="text-muted-foreground mt-0.5 size-4 shrink-0" aria-hidden="true" />
                <p className="text-muted-foreground text-sm">
                  This request has expired. The agent will need to submit a new
                  request if the action is still needed.
                </p>
              </div>
            )}

            {/* High risk warning */}
            {approval.context.risk_level === "high" && (
              <div role="alert" className="bg-destructive/10 border-destructive/20 flex items-start gap-2 rounded-lg border p-3">
                <AlertTriangle className="text-destructive mt-0.5 size-4 shrink-0" aria-hidden="true" />
                <p className="text-destructive text-sm">
                  This is a high-risk action. Please review the details carefully
                  before approving.
                </p>
              </div>
            )}
          </div>
        )}

        {isApproved ? (
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              size="lg"
              onClick={() => handleClose(false)}
            >
              Done
            </Button>
          </DialogFooter>
        ) : (
          <DialogFooter>
            <Button
              variant="outline"
              size="lg"
              disabled={isBusy || isExpired}
              onClick={handleDeny}
              aria-label={isDenying ? "Denying request…" : undefined}
            >
              {isDenying ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                "Deny"
              )}
            </Button>
            {showAlwaysAllow && (
              <Button
                size="lg"
                variant="secondary"
                disabled={isBusy || isExpired}
                onClick={handleAlwaysAllow}
                aria-label={pendingAction === "alwaysAllow" ? "Approving and creating rule…" : undefined}
              >
                {pendingAction === "alwaysAllow" ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <>
                    <ShieldCheck className="mr-1 size-4" />
                    Always Allow
                  </>
                )}
              </Button>
            )}
            <Button
              size="lg"
              disabled={isBusy || isExpired}
              onClick={handleApprove}
              aria-label={pendingAction === "approve" ? "Approving request…" : undefined}
            >
              {pendingAction === "approve" ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                "Approve"
              )}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>

      {/* Standing approval creation dialog — only mounted when needed */}
      {standingDialogOpen && (
        <CreateStandingApprovalDialog
          agents={agents}
          open={standingDialogOpen}
          onOpenChange={handleStandingDialogChange}
          onCreated={() => setStandingApprovalCreated(true)}
          initialAgentId={approval.agent_id}
          initialActionType={approval.action.type}
          initialConstraints={deriveConstraintsFromParams(params)}
        />
      )}
    </Dialog>
  );
}
