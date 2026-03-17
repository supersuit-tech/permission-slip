import {
  Bot,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Plug,
} from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import {
  type AuditEvent,
  OutcomeBadge,
  getAuditAgentName,
  getActionSummary,
} from "@/lib/auditEvents";
import { formatAbsoluteTime, formatRelativeTime } from "@/lib/utils";
import { useApprovalDetail } from "@/hooks/useApprovalDetail";
import { useActionSchema } from "@/hooks/useActionSchema";
import { ActionPreviewSummary } from "@/components/ActionPreviewSummary";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { RiskBadge } from "@/pages/dashboard/approval-components";
import { MetadataLabel } from "@/components/MetadataLabel";

/** Matches Radix Sheet exit animation duration so content stays rendered during close. */
export const SHEET_CLOSE_DELAY_MS = 300;

interface ActivityDetailSheetProps {
  event: AuditEvent | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Extract the action type from an audit event. */
function getActionType(event: AuditEvent): string {
  const action = event.action as Record<string, unknown> | null | undefined;
  if (!action) return "";
  return typeof action.type === "string" ? action.type : "";
}

/** Returns true if this event type can link back to an approval record. */
function isApprovalEvent(event: AuditEvent): boolean {
  return (
    event.source_type === "approval" &&
    !!event.source_id &&
    event.event_type !== "agent.registered" &&
    event.event_type !== "agent.deactivated"
  );
}

function ExecutionStatusSection({
  status,
  error,
}: {
  status: string;
  error?: string | null;
}) {
  const isSuccess = status === "success";
  const isFailure = status === "failure";
  const isTimeout = status === "timeout";

  const Icon = isSuccess
    ? CheckCircle2
    : isFailure || isTimeout
      ? XCircle
      : Clock;
  const color = isSuccess
    ? "text-green-600 dark:text-green-400"
    : isFailure || isTimeout
      ? "text-red-600 dark:text-red-400"
      : "text-muted-foreground";

  return (
    <div className="space-y-2">
      <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
        Execution
      </h4>
      <div className="flex items-center gap-2">
        <Icon className={`size-4 ${color}`} />
        <span className="text-sm font-medium capitalize">{status}</span>
      </div>
      {error && (
        <div className="bg-destructive/10 border-destructive/20 rounded-md border p-3">
          <div className="flex items-start gap-2">
            <AlertTriangle className="text-destructive mt-0.5 size-4 shrink-0" />
            <p className="text-destructive text-sm">{error}</p>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Supplemental approval details — parameters, execution result, risk, description.
 * Does NOT show a loading spinner; the action type is already visible from the
 * audit event. Renders nothing while loading, and shows available data or a
 * "details removed" message once loaded.
 */
function ApprovalSupplementalDetails({ approvalId }: { approvalId: string }) {
  const { approval, isLoading, error } = useApprovalDetail(approvalId);

  // While loading, render nothing — the action type header from the audit
  // event is already visible above.
  if (isLoading || !approval) return null;

  // Show a slim error message so API failures are visible to the user.
  if (error) {
    return (
      <p className="text-muted-foreground text-xs py-1">
        Could not load approval details.
      </p>
    );
  }

  const action = approval.action as Record<string, unknown> | undefined;
  const actionType = typeof action?.type === "string" ? action.type : "";
  const parameters = (action?.parameters ?? {}) as Record<string, unknown>;
  const context = approval.context as Record<string, unknown> | undefined;
  const riskLevel = context?.risk_level as "low" | "medium" | "high" | undefined;
  const description = typeof context?.description === "string" ? context.description : null;
  const executionResult = approval.execution_result as Record<string, unknown> | undefined;
  const executionError = typeof executionResult?.error === "string" ? executionResult.error : undefined;
  const resourceDetails = approval.resource_details as Record<string, unknown> | undefined;
  const executionStatus = approval.execution_status ?? undefined;
  const executedAt = approval.executed_at ?? undefined;

  const hasParameters = Object.keys(parameters).length > 0;
  const hasExecutionResult = executionResult && Object.keys(executionResult).length > 0;
  // NOTE: This heuristic can false-positive on actions that legitimately have
  // no parameters and no execution result. A backend `scrubbed_at` field would
  // eliminate the ambiguity, but for now this matches the pre-existing pattern
  // and is acceptable since most executed actions do produce at least one of
  // parameters or execution_result.
  // For denied/cancelled approvals, use the approval's resolved timestamp
  // (denied_at/cancelled_at) since they have no executed_at or execution_status.
  const resolvedAt = executedAt ?? approval?.denied_at ?? approval?.cancelled_at ?? approval?.approved_at;
  const isResolved = !!(executionStatus || approval?.status === "denied" || approval?.status === "cancelled");
  const isScrubbed = isResolved && !hasExecutionResult && !hasParameters && resolvedAt &&
    new Date(resolvedAt).getTime() < Date.now() - 30 * 60 * 1000;

  return (
    <ApprovalSupplementalContent
      actionType={actionType}
      parameters={parameters}
      riskLevel={riskLevel}
      description={description}
      executionStatus={executionStatus}
      executionError={executionError}
      executionResult={executionResult}
      resourceDetails={resourceDetails}
      isScrubbed={!!isScrubbed}
    />
  );
}

/** Action type header — shows the action name and type immediately from the audit event. */
function ActionTypeHeader({ actionType, actionName }: { actionType: string; actionName: string | null }) {
  return (
    <div className="space-y-2">
      <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
        Action
      </h4>
      <div className="bg-muted/50 rounded-lg border p-3">
        <span className="text-sm font-semibold">
          {actionName ?? actionType}
        </span>
        {actionName && (
          <div>
            <span className="text-muted-foreground font-mono text-xs">
              {actionType}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Supplemental content from the approval record: risk badge, description,
 * parameters, execution result, and scrubbed-data notice.
 */
function ApprovalSupplementalContent({
  actionType,
  parameters,
  riskLevel,
  description,
  executionStatus,
  executionError,
  executionResult,
  resourceDetails,
  isScrubbed,
}: {
  actionType: string;
  parameters: Record<string, unknown>;
  riskLevel?: "low" | "medium" | "high" | null;
  description?: string | null;
  executionStatus?: string;
  executionError?: string;
  executionResult?: Record<string, unknown>;
  resourceDetails?: Record<string, unknown>;
  isScrubbed: boolean;
}) {
  const { schema, actionName, displayTemplate } = useActionSchema(actionType);
  const hasParameters = Object.keys(parameters).length > 0;

  return (
    <div className="space-y-5">
      {/* Risk & description (from approval context) */}
      {(riskLevel || description) && (
        <div className="bg-muted/50 rounded-lg border p-3">
          {riskLevel && (
            <div className="mb-2">
              <RiskBadge level={riskLevel} />
            </div>
          )}
          {description && (
            <p className="text-muted-foreground text-sm">{description}</p>
          )}
        </div>
      )}

      {/* Action preview summary */}
      {hasParameters && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Preview
          </h4>
          <div className="bg-muted/50 rounded-lg border p-3">
            <ActionPreviewSummary
              actionType={actionType}
              parameters={parameters}
              schema={schema}
              actionName={actionName}
              displayTemplate={displayTemplate}
              resourceDetails={resourceDetails}
            />
          </div>
        </div>
      )}

      {/* Parameters */}
      {hasParameters && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Parameters
          </h4>
          <div className="bg-muted/50 overflow-x-auto rounded-lg border p-3">
            <SchemaParameterDetails
              parameters={parameters}
              schema={schema}
            />
          </div>
        </div>
      )}

      {/* Execution Result */}
      {executionStatus && (
        <ExecutionStatusSection status={executionStatus} error={executionError} />
      )}
      {executionResult && Object.keys(executionResult).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Execution Result
          </h4>
          <pre className="bg-muted/50 overflow-x-auto rounded-lg border p-3 text-xs">
            {JSON.stringify(executionResult, null, 2)}
          </pre>
        </div>
      )}
      {isScrubbed && (
        <p className="text-muted-foreground text-xs italic">
          Execution details are automatically removed after 30 minutes.
        </p>
      )}
    </div>
  );
}

/**
 * Full action content for non-approval events (standing approvals, inline events).
 * Shows action type, parameters, and execution result in a single block.
 */
function ApprovalContent({
  actionType,
  parameters,
  riskLevel,
  description,
  executionStatus,
  executionError,
  executionResult,
  executedAt,
  resourceDetails,
}: {
  actionType: string;
  parameters: Record<string, unknown>;
  riskLevel?: "low" | "medium" | "high" | null;
  description?: string | null;
  executionStatus?: string;
  executionError?: string;
  executionResult?: Record<string, unknown>;
  executedAt?: string;
  resourceDetails?: Record<string, unknown>;
}) {
  const { schema, actionName, displayTemplate } = useActionSchema(actionType);

  return (
    <div className="space-y-5">
      {/* Action Summary */}
      {actionType && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Action
          </h4>
          <div className="bg-muted/50 rounded-lg border p-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-sm font-semibold">
                {actionName ?? actionType}
              </span>
              {riskLevel && <RiskBadge level={riskLevel} />}
            </div>
            {actionName && (
              <span className="text-muted-foreground font-mono text-xs">
                {actionType}
              </span>
            )}
            {description && (
              <p className="text-muted-foreground mt-2 text-sm">{description}</p>
            )}
            {Object.keys(parameters).length > 0 && (
              <div className="border-t mt-3 pt-3">
                <ActionPreviewSummary
                  actionType={actionType}
                  parameters={parameters}
                  schema={schema}
                  actionName={actionName}
                  displayTemplate={displayTemplate}
                  resourceDetails={resourceDetails}
                />
              </div>
            )}
          </div>
        </div>
      )}

      {/* Parameters */}
      {Object.keys(parameters).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Parameters
          </h4>
          <div className="bg-muted/50 overflow-x-auto rounded-lg border p-3">
            <SchemaParameterDetails
              parameters={parameters}
              schema={schema}
            />
          </div>
        </div>
      )}

      {/* Execution Result */}
      {executionStatus && (
        <ExecutionStatusSection status={executionStatus} error={executionError} />
      )}
      {executionResult && Object.keys(executionResult).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
            Execution Result
          </h4>
          <pre className="bg-muted/50 overflow-x-auto rounded-lg border p-3 text-xs">
            {JSON.stringify(executionResult, null, 2)}
          </pre>
        </div>
      )}
      {executionStatus && !executionResult && executedAt &&
        Object.keys(parameters).length === 0 &&
        new Date(executedAt).getTime() < Date.now() - 30 * 60 * 1000 && (
        <p className="text-muted-foreground text-xs italic">
          Execution details are automatically removed after 30 minutes.
        </p>
      )}
    </div>
  );
}

export function ActivityDetailSheet({
  event,
  open,
  onOpenChange,
}: ActivityDetailSheetProps) {
  const actionType = event ? getActionType(event) : "";
  const shouldFetchApproval = event ? isApprovalEvent(event) : false;
  // Single useActionSchema call — shared between ActionTypeHeader and
  // ApprovalSupplementalContent to avoid duplicate hook subscriptions.
  const { actionName } = useActionSchema(actionType);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Activity Detail</SheetTitle>
          <SheetDescription className="sr-only">
            Details for this activity event
          </SheetDescription>
        </SheetHeader>

        {event && (
          <div className="space-y-5 px-4 pb-6">
          {/* Event header — outcome badge + metadata pills */}
          <div className="space-y-3">
            <OutcomeBadge outcome={event.outcome} />
            <div className="flex flex-wrap items-center gap-2">
              <MetadataLabel icon={Clock} title={formatRelativeTime(event.timestamp)}>
                {formatAbsoluteTime(event.timestamp)}
              </MetadataLabel>
              <MetadataLabel icon={Bot} title={`ID: ${event.agent_id}`}>{getAuditAgentName(event)}</MetadataLabel>
              {event.connector_id && (
                <MetadataLabel icon={Plug} className="capitalize">
                  {event.connector_id}
                </MetadataLabel>
              )}
            </div>
          </div>

          {/* Action type — shown immediately from audit event data */}
          {actionType && shouldFetchApproval && (
            <ActionTypeHeader actionType={actionType} actionName={actionName} />
          )}

          {/* Action summary from audit event (non-approval events) */}
          {actionType && !shouldFetchApproval && (
            <InlineActionDetails event={event} actionType={actionType} />
          )}

          {/* Supplemental details from approval record (parameters, execution, risk) */}
          {shouldFetchApproval && event.source_id && (
            <ApprovalSupplementalDetails approvalId={event.source_id} />
          )}

          {/* Execution status from audit event (for standing approvals) */}
          {event.execution_status && !shouldFetchApproval && (
            <ExecutionStatusSection
              status={event.execution_status}
              error={event.execution_error}
            />
          )}

          {/* Lifecycle event message */}
          {!actionType && (
            <div className="bg-muted/50 rounded-lg border p-4 text-center">
              <p className="text-sm">{getActionSummary(event)}</p>
            </div>
          )}
        </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

/**
 * Inline action details for events that have params directly on the audit event
 * (e.g., standing_approval.executed, payment_method.charged).
 */
function InlineActionDetails({
  event,
  actionType,
}: {
  event: AuditEvent;
  actionType: string;
}) {
  const action = event.action as Record<string, unknown>;
  const parameters = (action?.parameters ?? {}) as Record<string, unknown>;

  // Payment events have their own fields instead of parameters
  if (event.event_type === "payment_method.charged") {
    return (
      <div className="space-y-2">
        <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
          Payment Details
        </h4>
        <div className="bg-muted/50 rounded-lg border p-3 text-sm">
          <p>{getActionSummary(event)}</p>
        </div>
      </div>
    );
  }

  return (
    <ApprovalContent
      actionType={actionType}
      parameters={parameters}
    />
  );
}
