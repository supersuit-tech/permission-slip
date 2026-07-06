import { useState, useCallback, useEffect, useMemo } from "react";
import { Loader2, AlertTriangle, CheckCircle, XCircle, Check } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
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
import { useStandingApprovals } from "@/hooks/useStandingApprovals";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useActionConfigs } from "@/hooks/useActionConfigs";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { ActionPreviewCard } from "@/components/previews/ActionPreviewCard";
import {
  EmailThreadPreview,
  EMAIL_REPLY_ACTION_TYPES,
  parseEmailThreadFromDetails,
} from "@/components/previews/EmailThreadPreview";
import {
  ProtonInReplyToPreview,
  PROTON_REPLY_ACTION_TYPE,
  parseProtonInReplyTo,
} from "@/components/previews/ProtonInReplyToPreview";
import { SlackContextPreview } from "@/components/previews/SlackContextPreview";
import {
  ProtonBatchEmailsPreview,
  parseProtonBatchEmails,
} from "@/components/previews/ProtonBatchEmailsPreview";
import { ImessageParticipantsRow } from "@/components/previews/ImessageParticipantsRow";
import type { components } from "@/api/schema";
import { TimelineView, type TimelineEntry } from "@/components/TimelineView";
import { formatAbsoluteTime } from "@/lib/utils";
import { useCountdown, RiskBadge, CountdownBadge } from "./approval-components";
import { ApprovalSection } from "./ApprovalSection";
import { formatConnectorDisplayName } from "./approvalConnectorLabel";
import { EmailApprovalDetailsRow } from "@/components/previews/EmailApprovalDetailsRow";
import { emailDetailsUnavailable } from "@/lib/emailEnrichment";
import { shouldShowEmailApprovalSection } from "@/lib/emailApprovalDetails";
import { parseStandingApprovalFallthrough } from "@/lib/standingApprovalFallthrough";
import {
  buildCreateStandingApprovalFromApproval,
  findMatchingActionConfigForApproval,
} from "./standingApprovalFromApproval";

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

function buildTimelineEntries(approval: ApprovalSummary): TimelineEntry[] {
  const entries: TimelineEntry[] = [
    { label: "Created", value: formatAbsoluteTime(approval.created_at) },
    { label: "Expires", value: formatAbsoluteTime(approval.expires_at) },
  ];

  if (approval.approved_at) {
    entries.push({
      label: "Approved",
      value: formatAbsoluteTime(approval.approved_at),
      dotColor: "success",
    });
  }
  if (approval.denied_at) {
    entries.push({
      label: "Denied",
      value: formatAbsoluteTime(approval.denied_at),
      dotColor: "error",
    });
  }
  if (approval.cancelled_at) {
    entries.push({
      label: "Cancelled",
      value: formatAbsoluteTime(approval.cancelled_at),
    });
  }

  return entries;
}

export function ReviewApprovalDialog({
  approval,
  agentDisplayName,
  open,
  onOpenChange,
}: ReviewApprovalDialogProps) {
  const [approveResult, setApproveResult] = useState<ApproveResult | null>(null);
  const [standingApprovalCreated, setStandingApprovalCreated] = useState(false);
  const [autoApproveFuture, setAutoApproveFuture] = useState(false);
  const [pendingAction, setPendingAction] = useState<"approve" | null>(null);
  const [denialReason, setDenialReason] = useState("");
  const [denyInitiated, setDenyInitiated] = useState(false);
  const isApproved = approveResult !== null;
  const { approveApproval } = useApproveApproval();
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const { createStandingApproval } = useCreateStandingApproval();
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

  const { standingApprovals, isLoading: standingApprovalsLoading } = useStandingApprovals();
  const { configs } = useActionConfigs(approval.agent_id);
  const params = approval.action.parameters as Record<string, unknown>;
  const hasParams = Object.keys(params).length > 0;
  const timelineEntries = useMemo(
    () => buildTimelineEntries(approval),
    [approval],
  );
  const hasExistingStandingApproval = useMemo(
    () =>
      standingApprovals.some(
        (sa) =>
          sa.agent_id === approval.agent_id &&
          sa.action_type === approval.action.type,
      ),
    [standingApprovals, approval.agent_id, approval.action.type],
  );
  const matchingActionConfig = useMemo(
    () => findMatchingActionConfigForApproval(configs, approval),
    [configs, approval],
  );
  const showAutoApproveCheckbox =
    !standingApprovalsLoading && !hasExistingStandingApproval;

  const emailThread = useMemo(
    () => parseEmailThreadFromDetails(approval.context.details),
    [approval.context.details],
  );
  const showEmailThreadPreview = EMAIL_REPLY_ACTION_TYPES.has(approval.action.type);

  const protonInReplyTo = useMemo(
    () =>
      parseProtonInReplyTo(
        approval.resource_details as Record<string, unknown> | undefined,
      ),
    [approval.resource_details],
  );
  const showProtonInReplyToPreview =
    approval.action.type === PROTON_REPLY_ACTION_TYPE;

  const protonBatchEmails = useMemo(
    () =>
      parseProtonBatchEmails(
        approval.resource_details as Record<string, unknown> | undefined,
      ),
    [approval.resource_details],
  );

  const standingApprovalFallthrough = useMemo(
    () => parseStandingApprovalFallthrough(approval.context.details),
    [approval.context.details],
  );

  const slackContext = useMemo(
    () =>
      (approval.context.details as Record<string, unknown> | undefined)
        ?.slack_context as components["schemas"]["SlackContext"] | undefined,
    [approval.context.details],
  );

  useEffect(() => {
    if (!isApproved) return;
    const timer = setTimeout(() => onOpenChange(false), SUCCESS_AUTO_CLOSE_MS);
    return () => clearTimeout(timer);
  }, [isApproved, onOpenChange]);

  const handleApprove = useCallback(async () => {
    setPendingAction("approve");
    try {
      const result = await approveApproval(approval.approval_id);
      setApproveResult(result);

      if (autoApproveFuture && result.execution_status !== "error") {
        try {
          await createStandingApproval(
            buildCreateStandingApprovalFromApproval(
              approval,
              matchingActionConfig?.id,
            ),
          );
          setStandingApprovalCreated(true);
        } catch (err) {
          toast.error(
            err instanceof Error
              ? err.message
              : "Request approved, but failed to create auto-approval rule.",
          );
        }
      }
    } catch {
      toast.error("Failed to approve request. Please try again.");
    } finally {
      setPendingAction(null);
    }
  }, [
    approveApproval,
    approval,
    autoApproveFuture,
    matchingActionConfig,
    createStandingApproval,
  ]);

  const handleDeny = useCallback(async () => {
    if (!denyInitiated) {
      setDenyInitiated(true);
      return;
    }

    try {
      const reason = denialReason.trim();
      await denyApproval(approval.approval_id, reason || undefined);
      toast.success("Request denied");
      onOpenChange(false);
    } catch {
      toast.error("Failed to deny request. Please try again.");
    }
  }, [denyApproval, approval.approval_id, denialReason, denyInitiated, onOpenChange]);

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) {
      setApproveResult(null);
      setStandingApprovalCreated(false);
      setAutoApproveFuture(false);
      setPendingAction(null);
      setDenialReason("");
      setDenyInitiated(false);
    }
    onOpenChange(nextOpen);
  }

  function cancelDeny() {
    setDenyInitiated(false);
    setDenialReason("");
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-h-[90dvh] w-[calc(100vw-1.5rem)] overflow-y-auto sm:max-w-2xl">
        {/* Connector header with logo, action name, and connector name */}
        <DialogHeader>
          <div className="flex items-center gap-3">
            {schemaLoading ? (
              <>
                <Skeleton className="size-10 shrink-0 rounded-lg" />
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
                <CheckCircle className="size-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden="true" />
                <p className="text-sm text-green-800 dark:text-green-300">
                  Future matching requests will be auto-approved.
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

            {/* Standing approval could not be auto-evaluated */}
            {standingApprovalFallthrough && (
              <div
                className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200"
                data-testid="standing-approval-fallthrough"
                role="status"
              >
                {standingApprovalFallthrough.message}
              </div>
            )}

            {/* Rich action preview card */}
            <div className="space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="truncate text-sm font-semibold">
                    {actionName ?? approval.action.type}
                  </span>
                </div>
                <RiskBadge level={approval.context.risk_level} />
              </div>

              {approval.context.description && (
                <p className="text-muted-foreground text-sm">
                  {approval.context.description}
                </p>
              )}

              {schemaLoading ? (
                <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm space-y-3">
                  <div className="flex items-center gap-3">
                    <Skeleton className="size-10 shrink-0 rounded-lg" />
                    <div className="flex-1 space-y-2">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                  </div>
                  <Skeleton className="h-3 w-full" />
                </div>
              ) : (
                <>
                  {shouldShowEmailApprovalSection(
                    approval.action.type,
                    approval.resource_details as Record<string, unknown> | undefined,
                  ) && (
                    <EmailApprovalDetailsRow
                      actionType={approval.action.type}
                      resourceDetails={
                        approval.resource_details as Record<string, unknown> | undefined
                      }
                    />
                  )}
                  {emailDetailsUnavailable(
                    approval.action.type,
                    approval.resource_details as Record<string, unknown> | undefined,
                  ) && (
                    <p
                      className="text-muted-foreground text-sm italic"
                      data-testid="email-details-unavailable"
                    >
                      Email details unavailable
                    </p>
                  )}
                  {showEmailThreadPreview && (
                    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
                      <EmailThreadPreview thread={emailThread} />
                    </div>
                  )}
                  {showProtonInReplyToPreview && (
                    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
                      <ProtonInReplyToPreview metadata={protonInReplyTo} />
                    </div>
                  )}
                  <ActionPreviewCard
                    preview={preview}
                    parameters={params}
                    actionType={approval.action.type}
                    schema={schema}
                    actionName={actionName}
                    displayTemplate={displayTemplate}
                    resourceDetails={approval.resource_details as Record<string, unknown> | undefined}
                  />
                  {protonBatchEmails && (
                    <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
                      <ProtonBatchEmailsPreview emails={protonBatchEmails} />
                    </div>
                  )}
                  <ImessageParticipantsRow
                    resourceDetails={
                      approval.resource_details as Record<string, unknown> | undefined
                    }
                  />
                  {slackContext && <SlackContextPreview slackContext={slackContext} />}
                </>
              )}
            </div>

            {/* Expiry */}
            <ApprovalSection label="Expiry">
              <div className="flex items-center gap-2">
                <CountdownBadge expiresAt={approval.expires_at} />
                <span className="text-muted-foreground text-sm">
                  {isExpired ? "This request has expired" : "remaining"}
                </span>
              </div>
              {isExpired && (
                <p className="text-muted-foreground mt-2 text-sm" role="alert">
                  The agent will need to submit a new request if the action is still needed.
                </p>
              )}
            </ApprovalSection>

            {/* Parameters */}
            {hasParams && (
              <ApprovalSection label="Parameters">
                <SchemaParameterDetails
                  parameters={params}
                  schema={schema}
                  actionType={approval.action.type}
                  resourceDetails={
                    approval.resource_details as Record<string, unknown> | undefined
                  }
                />
              </ApprovalSection>
            )}

            {/* Timeline */}
            <ApprovalSection label="Timeline">
              <TimelineView entries={timelineEntries} />
            </ApprovalSection>

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

            {/* Action buttons */}
            <div className="space-y-3 pt-2">
              {showAutoApproveCheckbox && !denyInitiated && (
                <div className="flex items-start gap-3 rounded-lg border p-3">
                  <Checkbox
                    id="auto-approve-future"
                    checked={autoApproveFuture}
                    onCheckedChange={(checked) =>
                      setAutoApproveFuture(checked === true)
                    }
                    disabled={isBusy || isExpired}
                    aria-describedby="auto-approve-future-label"
                  />
                  <Label
                    id="auto-approve-future-label"
                    htmlFor="auto-approve-future"
                    className="cursor-pointer text-sm font-normal leading-snug"
                  >
                    Auto-approve all future requests like this
                  </Label>
                </div>
              )}

              {denyInitiated && (
                <div className="space-y-2">
                  <Label htmlFor="denial-reason" className="text-sm font-medium">
                    Reason for denial (optional)
                  </Label>
                  <textarea
                    id="denial-reason"
                    value={denialReason}
                    onChange={(event) => setDenialReason(event.target.value)}
                    disabled={isBusy || isExpired}
                    maxLength={500}
                    rows={3}
                    autoFocus
                    placeholder="Explain why you're denying this request so the agent can adapt."
                    className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[80px] w-full rounded-md border px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  />
                </div>
              )}

              <div className="flex flex-col gap-2 sm:flex-row">
                {denyInitiated ? (
                  <Button
                    variant="outline"
                    size="lg"
                    className="w-full sm:flex-1"
                    disabled={isBusy || isExpired}
                    onClick={cancelDeny}
                  >
                    Cancel
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="lg"
                    className="w-full border-destructive/40 text-destructive hover:bg-destructive/5 hover:text-destructive sm:flex-1"
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
                )}

                {denyInitiated ? (
                  <Button
                    variant="destructive"
                    size="lg"
                    className="w-full sm:flex-1"
                    disabled={isBusy || isExpired}
                    onClick={handleDeny}
                    aria-label={isDenying ? "Confirming denial…" : undefined}
                  >
                    {isDenying ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      "Confirm deny"
                    )}
                  </Button>
                ) : (
                  <Button
                    size="lg"
                    className="w-full bg-emerald-600 text-white hover:bg-emerald-700 dark:bg-emerald-600 dark:hover:bg-emerald-700 sm:flex-1"
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
                )}
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
