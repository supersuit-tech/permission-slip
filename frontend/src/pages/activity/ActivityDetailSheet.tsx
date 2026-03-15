import {
  Loader2,
  Bot,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
} from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
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

function ApprovalDetails({ approvalId }: { approvalId: string }) {
  const { approval, isLoading, error } = useApprovalDetail(approvalId);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 py-4">
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
        <span className="text-muted-foreground text-sm">Loading details...</span>
      </div>
    );
  }

  if (error || !approval) {
    return (
      <p className="text-muted-foreground text-sm py-2">
        {error ?? "Approval details not available."}
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

  return (
    <ApprovalContent
      actionType={actionType}
      parameters={parameters}
      riskLevel={riskLevel}
      description={description}
      executionStatus={approval.execution_status ?? undefined}
      executionError={executionError}
      executionResult={executionResult}
      executedAt={approval.executed_at ?? undefined}
    />
  );
}

function ApprovalContent({
  actionType,
  parameters,
  riskLevel,
  description,
  executionStatus,
  executionError,
  executionResult,
  executedAt,
}: {
  actionType: string;
  parameters: Record<string, unknown>;
  riskLevel?: "low" | "medium" | "high" | null;
  description?: string | null;
  executionStatus?: string;
  executionError?: string;
  executionResult?: Record<string, unknown>;
  executedAt?: string;
}) {
  const { schema, actionName } = useActionSchema(actionType);

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
          {/* Event header */}
          <div className="flex items-center justify-between gap-3">
            <OutcomeBadge outcome={event.outcome} />
            <span className="text-muted-foreground text-xs" title={formatRelativeTime(event.timestamp)}>
              {formatAbsoluteTime(event.timestamp)}
            </span>
          </div>

          {/* Agent */}
          <div className="space-y-2">
            <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
              Agent
            </h4>
            <div className="flex items-center gap-3">
              <div className="bg-muted rounded-full p-2">
                <Bot className="text-muted-foreground size-4" />
              </div>
              <div>
                <p className="text-sm font-medium">{getAuditAgentName(event)}</p>
                <p className="text-muted-foreground font-mono text-xs">
                  ID: {event.agent_id}
                </p>
              </div>
            </div>
          </div>

          {/* Connector */}
          {event.connector_id && (
            <div className="space-y-2">
              <h4 className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
                Connector
              </h4>
              <Badge variant="outline" className="capitalize">
                {event.connector_id}
              </Badge>
            </div>
          )}

          {/* Action summary from audit event */}
          {actionType && !shouldFetchApproval && (
            <InlineActionDetails event={event} actionType={actionType} />
          )}

          {/* Full details from approval record */}
          {shouldFetchApproval && event.source_id && (
            <ApprovalDetails approvalId={event.source_id} />
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
