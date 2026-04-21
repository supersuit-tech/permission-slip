import { useState, useCallback, useEffect, useMemo } from "react";
import { Loader2, Clock, AlertTriangle, CheckCircle, XCircle, ShieldCheck, Check } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { getInitials } from "@/components/ui/avatar";
import { useApproveApproval } from "@/hooks/useApproveApproval";
import type { ApproveResult } from "@/hooks/useApproveApproval";
import { useDenyApproval } from "@/hooks/useDenyApproval";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { ApprovalSummary } from "@/hooks/useApprovals";
import { useAgents } from "@/hooks/useAgents";
import { useStandingApprovals } from "@/hooks/useStandingApprovals";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { ActionPreviewCard } from "@/components/previews/ActionPreviewCard";
import { CreateStandingApprovalDialog } from "./CreateStandingApprovalDialog";
import {
  useCountdown,
  formatCountdown,
  urgencyColor,
  RiskBadge,
} from "./approval-components";
import { formatConnectorDisplayName } from "./approvalConnectorLabel";

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
  const [paramsOpen, setParamsOpen] = useState(false);
  const isApproved = approveResult !== null;
  const { approveApproval } = useApproveApproval();
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const { schema, actionName, displayTemplate, preview, connectorName, connectorLogoSvg, isLoading: schemaLoading } =
    useActionSchema(approval.action.type);
  const connectorInstanceDisplayStr =
    typeof approval.action === "object" &&
    approval.action !== null &&
    "_connector_instance_display" in approval.action &&
    typeof (approval.action as { _connector_instance_display?: unknown })._connector_instance_display === "string"
      ? (approval.action as { _connector_instance_display: string })._connector_instance_display
      : undefined;
  const connectorInstanceLabelStr =
    typeof approval.action === "object" &&
    approval.action !== null &&
    "_connector_instance_label" in approval.action &&
    typeof (approval.action as { _connector_instance_label?: unknown })._connector_instance_label === "string"
      ? (approval.action as { _connector_instance_label: string })._connector_instance_label
      : undefined;
  const remaining = useCountdown(approval.expires_at);
  const isExpired = remaining <= 0;
  const isBusy = pendingAction !== null || isDenying;

  // Data for "Always Allow This"
  const { agents } = useAgents();
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

  // Auto-close dialog after successful approval (never while the nested standing-approval
  // wizard is open — a render with isApproved true before autoCloseBlocked flips true
  // would otherwise start the timer and unmount the child dialog).
  useEffect(() => {
    if (!isApproved || autoCloseBlocked || standingDialogOpen) return;
    const timer = setTimeout(() => onOpenChange(false), SUCCESS_AUTO_CLOSE_MS);
    return () => clearTimeout(timer);
  }, [isApproved, autoCloseBlocked, standingDialogOpen, onOpenChange]);

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
    setAutoCloseBlocked(true);
    try {
      const result = await approveApproval(approval.approval_id);
      setApproveResult(result);
      if (result.execution_status === "error") {
        setAutoCloseBlocked(false);
      } else {
        setStandingDialogOpen(true);
      }
    } catch {
      setAutoCloseBlocked(false);
      toast.error("Failed to approve request. Please try again.");
    } finally {
      setPendingAction(null);
    }
  }, [approveApproval, approval.approval_id]);

  function handleStandingDialogChange(nextOpen: boolean) {
    setStandingDialogOpen(nextOpen);
    // Unblock parent auto-close whenever the nested wizard closes (cancel or success).
    // Do not gate on standingApprovalCreated — that state updates after this handler runs,
    // so a closure check would always see the pre-create value.
    if (!nextOpen) {
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
        {/* Connector header with logo, action name, and connector name */}
        <DialogHeader>
          <div className="flex items-center gap-3">
            {schemaLoading ? (
              <>
                <Skeleton className="size-10 rounded-lg shrink-0" />
                <div className="min-w-0 flex-1 space-y-2">
                  <DialogTitle className="sr-only">Loading action details…</DialogTitle>
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-3 w-24" />
                </div>
              </>
            ) : (
              <>
                <ConnectorLogo
                  name={connectorName ?? approval.action.type}
                  logoSvg={connectorLogoSvg}
                  size="lg"
                />
                <div className="min-w-0">
                  <DialogTitle className="truncate text-base">
                    {actionName ?? approval.action.type}
                  </DialogTitle>
                  <p className="text-muted-foreground text-sm">
                    {formatConnectorDisplayName({
                      connectorName,
                      actionType: approval.action.type,
                      instanceDisplay: connectorInstanceDisplayStr,
                      instanceLabel: connectorInstanceLabelStr,
                    })}
                  </p>
                </div>
              </>
            )}
          </div>
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

            <Button
              variant="outline"
              size="lg"
              className="w-full"
              onClick={() => handleClose(false)}
            >
              Done
            </Button>
          </div>
        ) : (
          <div className="space-y-4 sm:space-y-5">
            {/* Agent info + status */}
            <div className="flex items-center justify-between gap-3">
              <div className="flex min-w-0 items-center gap-3">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30">
                  <span className="text-sm font-bold text-violet-700 dark:text-violet-300" aria-hidden="true">
                    {getInitials(agentDisplayName)}
                  </span>
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">{agentDisplayName}</p>
                  <p className="text-muted-foreground text-xs">wants to perform an action</p>
                </div>
              </div>
              <Badge variant="warning-soft" className="shrink-0 uppercase">
                Pending
              </Badge>
            </div>

            {/* Rich action preview card */}
            <div className="space-y-2">
              {/* Action header above the card */}
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="truncate text-sm font-semibold">
                    {actionName ?? approval.action.type}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <RiskBadge level={approval.context.risk_level} />
                  <span
                    aria-label={isExpired ? "Request expired" : `Time remaining: ${formatCountdown(remaining)}`}
                    className={`inline-flex shrink-0 items-center gap-1 text-xs font-medium ${urgencyColor(remaining)}`}
                  >
                    <Clock className="size-3" aria-hidden="true" />
                    {isExpired ? "Expired" : formatCountdown(remaining)}
                  </span>
                </div>
              </div>

              {/* Context description */}
              {approval.context.description && (
                <p className="text-muted-foreground text-sm">
                  {approval.context.description}
                </p>
              )}

              {/* Preview card — show skeleton while connector metadata loads */}
              {schemaLoading ? (
                <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm space-y-3">
                  <div className="flex items-center gap-3">
                    <Skeleton className="size-10 rounded-lg shrink-0" />
                    <div className="flex-1 space-y-2">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                  </div>
                  <Skeleton className="h-3 w-full" />
                </div>
              ) : (
                <ActionPreviewCard
                  preview={preview}
                  parameters={params}
                  actionType={approval.action.type}
                  schema={schema}
                  actionName={actionName}
                  displayTemplate={displayTemplate}
                  resourceDetails={approval.resource_details as Record<string, unknown> | undefined}
                />
              )}
            </div>

            {/* Raw parameters (collapsible pill toggle) */}
            {hasParams && (
              <div className="space-y-2">
                <button
                  type="button"
                  onClick={() => setParamsOpen((o) => !o)}
                  className="text-muted-foreground hover:text-foreground hover:bg-muted/50 inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
                  aria-expanded={paramsOpen}
                >
                  <span
                    className="transition-transform duration-150"
                    style={{ display: "inline-block", transform: paramsOpen ? "rotate(90deg)" : "rotate(0deg)" }}
                    aria-hidden="true"
                  >
                    &#9656;
                  </span>
                  Parameters
                </button>
                {paramsOpen && (
                  <div className="bg-muted/30 overflow-x-auto rounded-xl border p-3 sm:p-4">
                    <SchemaParameterDetails
                      parameters={params}
                      schema={schema}
                      actionType={approval.action.type}
                      resourceDetails={
                        approval.resource_details as Record<string, unknown> | undefined
                      }
                    />
                  </div>
                )}
              </div>
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

            {/* Action buttons — stacked full-width matching mockup */}
            <div className="space-y-2 pt-2">
              <Button
                size="lg"
                className="w-full bg-emerald-600 text-white hover:bg-emerald-700 dark:bg-emerald-600 dark:hover:bg-emerald-700"
                disabled={isBusy || isExpired}
                onClick={handleApprove}
                aria-label={pendingAction === "approve" ? "Approving request…" : undefined}
              >
                {pendingAction === "approve" ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <>
                    <Check className="mr-1 size-4" />
                    Approve
                  </>
                )}
              </Button>

              <Button
                variant="outline"
                size="lg"
                className="w-full"
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
                <button
                  className="text-muted-foreground hover:text-foreground flex w-full items-center justify-center gap-1 py-1 text-sm transition-colors disabled:opacity-50"
                  disabled={isBusy || isExpired}
                  onClick={handleAlwaysAllow}
                  aria-label={pendingAction === "alwaysAllow" ? "Approving and creating rule…" : undefined}
                  type="button"
                >
                  {pendingAction === "alwaysAllow" ? (
                    <Loader2 className="size-3 animate-spin" />
                  ) : (
                    <>
                      <ShieldCheck className="size-3" />
                      Always allow this action
                    </>
                  )}
                </button>
              )}
            </div>
          </div>
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
