/**
 * Approval Dialog — Storybook stories
 *
 * Renders the approval dialog UI for all connector actions and states
 * using the real presentational sub-components. The hooks used by
 * ReviewApprovalDialog are replaced with static mock data so the
 * stories work standalone.
 */
import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  Check,
  Clock,
  Loader2,
  AlertTriangle,
  ShieldCheck,
  CheckCircle,
  XCircle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { ActionPreviewCard } from "@/components/previews/ActionPreviewCard";
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { RiskBadge } from "./approval-components";
import { getInitials } from "@/components/ui/avatar";
import type { ParametersSchema } from "@/lib/parameterSchema";
import type { ActionPreviewConfig } from "@/hooks/useActionSchema";

// ---------------------------------------------------------------------------
// SVG logos (embedded to avoid needing API calls)
// ---------------------------------------------------------------------------

const GOOGLE_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>Google</title><path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/></svg>`;

const GITHUB_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>GitHub</title><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`;

const SLACK_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>Slack</title><path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zM18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zM15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z"/></svg>`;

// ---------------------------------------------------------------------------
// Mock approval data types
// ---------------------------------------------------------------------------

interface MockApprovalProps {
  connectorName: string;
  connectorLogo: string;
  actionName: string;
  actionType: string;
  agentName: string;
  riskLevel: "low" | "medium" | "high";
  parameters: Record<string, unknown>;
  schema: ParametersSchema | null;
  preview: ActionPreviewConfig | null;
  displayTemplate?: string | null;
  resourceDetails?: Record<string, unknown>;
  /** Countdown display text (static for stories). */
  countdown?: string;
  /** Context description shown above the preview. */
  description?: string;
  /** Whether to show the dialog in an expired state. */
  expired?: boolean;
  /** Override dialog state: "pending" | "approved-success" | "approved-error" | "approved-pending". */
  dialogState?: "pending" | "approved-success" | "approved-error" | "approved-pending";
}

// ---------------------------------------------------------------------------
// Story component — mirrors ReviewApprovalDialog without hooks
// ---------------------------------------------------------------------------

function ApprovalDialogStory({
  connectorName,
  connectorLogo,
  actionName,
  actionType,
  agentName,
  riskLevel,
  parameters,
  schema,
  preview,
  displayTemplate,
  resourceDetails,
  countdown = "9:30",
  description,
  expired = false,
  dialogState = "pending",
}: MockApprovalProps) {
  const [rawOpen, setRawOpen] = useState(false);
  const hasParams = Object.keys(parameters).length > 0;

  if (dialogState === "approved-success") {
    return (
      <Dialog open>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <div className="flex items-center gap-3">
              <ConnectorLogo name={connectorName} logoSvg={connectorLogo} size="lg" />
              <div className="min-w-0">
                <DialogTitle className="truncate text-base">{actionName}</DialogTitle>
                <p className="text-muted-foreground text-sm">{connectorName}</p>
              </div>
            </div>
          </DialogHeader>
          <div className="space-y-6 py-4" role="status">
            <div className="flex flex-col items-center gap-4">
              <div className="rounded-full bg-green-100 p-3 dark:bg-green-900/30">
                <CheckCircle className="size-8 text-green-600 dark:text-green-400" />
              </div>
              <p className="text-lg font-semibold">Action Executed Successfully</p>
              <p className="text-muted-foreground text-sm text-center">
                The action has been executed. The agent has been notified.
              </p>
            </div>
            <Button variant="outline" size="lg" className="w-full">Done</Button>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  if (dialogState === "approved-error") {
    return (
      <Dialog open>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <div className="flex items-center gap-3">
              <ConnectorLogo name={connectorName} logoSvg={connectorLogo} size="lg" />
              <div className="min-w-0">
                <DialogTitle className="truncate text-base">{actionName}</DialogTitle>
                <p className="text-muted-foreground text-sm">{connectorName}</p>
              </div>
            </div>
          </DialogHeader>
          <div className="space-y-6 py-4" role="status">
            <div className="flex flex-col items-center gap-4">
              <div className="rounded-full bg-red-100 p-3 dark:bg-red-900/30">
                <XCircle className="size-8 text-red-600 dark:text-red-400" />
              </div>
              <p className="text-lg font-semibold">Execution Failed</p>
              <p className="text-muted-foreground text-sm text-center">
                The action was approved but execution failed. 404 Not Found: Calendar event not found.
              </p>
            </div>
            <Button variant="outline" size="lg" className="w-full">Done</Button>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  if (dialogState === "approved-pending") {
    return (
      <Dialog open>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <div className="flex items-center gap-3">
              <ConnectorLogo name={connectorName} logoSvg={connectorLogo} size="lg" />
              <div className="min-w-0">
                <DialogTitle className="truncate text-base">{actionName}</DialogTitle>
                <p className="text-muted-foreground text-sm">{connectorName}</p>
              </div>
            </div>
          </DialogHeader>
          <div className="space-y-6 py-4" role="status">
            <div className="flex flex-col items-center gap-4">
              <div className="rounded-full bg-blue-100 p-3 dark:bg-blue-900/30">
                <Loader2 className="size-8 animate-spin text-blue-600 dark:text-blue-400" />
              </div>
              <p className="text-lg font-semibold">Request Approved</p>
              <p className="text-muted-foreground text-sm text-center">
                The action is being executed…
              </p>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        {/* Connector header */}
        <DialogHeader>
          <div className="flex items-center gap-3">
            <ConnectorLogo name={connectorName} logoSvg={connectorLogo} size="lg" />
            <div className="min-w-0">
              <DialogTitle className="truncate text-base">{actionName}</DialogTitle>
              <p className="text-muted-foreground text-sm">{connectorName}</p>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4 sm:space-y-5">
          {/* Agent info + status */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30">
                <span className="text-sm font-bold text-violet-700 dark:text-violet-300">
                  {getInitials(agentName)}
                </span>
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">{agentName}</p>
                <p className="text-muted-foreground text-xs">wants to perform an action</p>
              </div>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">
              Pending
            </Badge>
          </div>

          {/* Action preview */}
          <div className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex min-w-0 items-center gap-2">
                <span className="truncate text-sm font-semibold">{actionName}</span>
              </div>
              <div className="flex items-center gap-2">
                <RiskBadge level={riskLevel} />
                <span
                  className={`inline-flex shrink-0 items-center gap-1 text-xs font-medium ${expired ? "text-muted-foreground" : "text-muted-foreground"}`}
                >
                  <Clock className="size-3" />
                  {expired ? "Expired" : countdown}
                </span>
              </div>
            </div>

            {description && (
              <p className="text-muted-foreground text-sm">{description}</p>
            )}

            <ActionPreviewCard
              preview={preview}
              parameters={parameters}
              actionType={actionType}
              schema={schema}
              actionName={actionName}
              displayTemplate={displayTemplate}
              resourceDetails={resourceDetails}
            />
          </div>

          {/* Raw parameters (collapsible pill toggle) */}
          {hasParams && (
            <div className="space-y-2">
              <button
                type="button"
                onClick={() => setRawOpen((o) => !o)}
                className="text-muted-foreground hover:text-foreground hover:bg-muted/50 inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
                aria-expanded={rawOpen}
              >
                <span
                  className="transition-transform duration-150"
                  style={{ display: "inline-block", transform: rawOpen ? "rotate(90deg)" : "rotate(0deg)" }}
                  aria-hidden="true"
                >
                  &#9656;
                </span>
                Parameters
              </button>
              {rawOpen && (
                <div className="bg-muted/30 overflow-x-auto rounded-xl border p-3 sm:p-4">
                  <SchemaParameterDetails parameters={parameters} schema={schema} />
                </div>
              )}
            </div>
          )}

          {/* Expired notice */}
          {expired && (
            <div role="alert" className="bg-muted/50 flex items-start gap-2 rounded-lg border p-3">
              <Clock className="text-muted-foreground mt-0.5 size-4 shrink-0" />
              <p className="text-muted-foreground text-sm">
                This request has expired. The agent will need to submit a new
                request if the action is still needed.
              </p>
            </div>
          )}

          {/* High risk warning */}
          {riskLevel === "high" && (
            <div role="alert" className="bg-destructive/10 border-destructive/20 flex items-start gap-2 rounded-lg border p-3">
              <AlertTriangle className="text-destructive mt-0.5 size-4 shrink-0" />
              <p className="text-destructive text-sm">
                This is a high-risk action. Please review the details carefully
                before approving.
              </p>
            </div>
          )}

          {/* Action buttons */}
          <div className="space-y-2 pt-2">
            <Button
              size="lg"
              className="w-full bg-emerald-600 text-white hover:bg-emerald-700 dark:bg-emerald-600 dark:hover:bg-emerald-700"
              disabled={expired}
            >
              <Check className="mr-1 size-4" />
              Approve
            </Button>
            <Button variant="outline" size="lg" className="w-full" disabled={expired}>
              Deny
            </Button>
            {hasParams && (
              <button
                className="text-muted-foreground hover:text-foreground flex w-full items-center justify-center gap-1 py-1 text-sm transition-colors"
                type="button"
                disabled={expired}
              >
                <ShieldCheck className="size-3" />
                Always allow this action
              </button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Meta
// ---------------------------------------------------------------------------

const meta: Meta<typeof ApprovalDialogStory> = {
  title: "UI/Dialog/Approval Dialog",
  component: ApprovalDialogStory,
  parameters: { layout: "centered" },
};
export default meta;
type Story = StoryObj<typeof ApprovalDialogStory>;

// ═══════════════════════════════════════════════════════════════════════════
// DIALOG STATES
// ═══════════════════════════════════════════════════════════════════════════

export const ApprovedSuccess: Story = {
  name: "State: Approved (Success)",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Delete Calendar Event",
    actionType: "google.delete_calendar_event",
    agentName: "Chiedobot",
    riskLevel: "high",
    parameters: { event_id: "fifrnnp6iai8qi5klOhl9cspms", calendar_id: "primary" },
    schema: null,
    preview: null,
    dialogState: "approved-success",
  },
};

export const ApprovedError: Story = {
  name: "State: Approved (Error)",
  args: {
    ...ApprovedSuccess.args,
    dialogState: "approved-error",
  },
};

export const ApprovedPending: Story = {
  name: "State: Approved (Executing)",
  args: {
    ...ApprovedSuccess.args,
    dialogState: "approved-pending",
  },
};

export const Expired: Story = {
  name: "State: Expired",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Delete Calendar Event",
    actionType: "google.delete_calendar_event",
    agentName: "Chiedobot",
    riskLevel: "high",
    parameters: { event_id: "fifrnnp6iai8qi5klOhl9cspms", calendar_id: "primary" },
    schema: {
      type: "object",
      required: ["event_id"],
      properties: {
        event_id: { type: "string", description: "The ID of the calendar event to delete" },
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
      },
    },
    preview: null,
    resourceDetails: { title: "Q1 Planning Review", start_time: "2026-03-16T14:00:00-05:00" },
    expired: true,
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// GOOGLE CONNECTOR — all actions
// ═══════════════════════════════════════════════════════════════════════════

export const GoogleSendEmail: Story = {
  name: "Google: Send Email",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Send Email",
    actionType: "google.send_email",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Send email to {{to}} — {{subject}}",
    preview: {
      layout: "message",
      fields: { to: "to", subject: "subject", body: "body" },
    },
    parameters: {
      to: "alice@example.com",
      subject: "Weekly standup notes — March 16",
      body: "Hi Alice,\n\nHere are this week's standup notes. The team made great progress on the API migration and the new dashboard is nearly ready for QA.\n\nBest,\nChiedobot",
    },
    schema: {
      type: "object",
      required: ["to", "subject", "body"],
      properties: {
        to: { type: "string", description: "Recipient email address" },
        subject: { type: "string", description: "Email subject line" },
        body: { type: "string", description: "Email body (plain text)" },
      },
    },
  },
};

export const GoogleListEmails: Story = {
  name: "Google: List Emails",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "List Emails",
    actionType: "google.list_emails",
    agentName: "Chiedobot",
    riskLevel: "low",
    displayTemplate: "List emails matching {{query}}",
    preview: null,
    parameters: { query: "from:hiring@company.com is:unread", max_results: 20 },
    schema: {
      type: "object",
      properties: {
        query: { type: "string", description: "Gmail search query" },
        max_results: { type: "integer", description: "Maximum number of emails to return (1-100, default 10)" },
      },
    },
  },
};

export const GoogleCreateCalendarEvent: Story = {
  name: "Google: Create Calendar Event",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Create Calendar Event",
    actionType: "google.create_calendar_event",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Create event {{summary}} on {{start_time:datetime}} with {{attendees:count}} attendees",
    preview: {
      layout: "event",
      fields: { title: "summary", start: "start_time", end: "end_time" },
    },
    parameters: {
      summary: "Q1 Planning Review",
      description: "Review Q1 OKRs and plan Q2 priorities",
      start_time: "2026-03-20T14:00:00-05:00",
      end_time: "2026-03-20T15:30:00-05:00",
      attendees: ["alice@example.com", "bob@example.com", "carol@example.com"],
      calendar_id: "primary",
    },
    schema: {
      type: "object",
      required: ["summary", "start_time", "end_time"],
      properties: {
        summary: { type: "string", description: "Event title" },
        description: { type: "string", description: "Event description" },
        start_time: { type: "string", description: "Start time in RFC 3339 format" },
        end_time: { type: "string", description: "End time in RFC 3339 format" },
        attendees: { type: "array", description: "List of attendee email addresses" },
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
      },
    },
  },
};

export const GoogleListCalendarEvents: Story = {
  name: "Google: List Calendar Events",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "List Calendar Events",
    actionType: "google.list_calendar_events",
    agentName: "Chiedobot",
    riskLevel: "low",
    displayTemplate: "List calendar events from {{calendar_id}}",
    preview: null,
    parameters: { calendar_id: "primary", max_results: 10 },
    schema: {
      type: "object",
      properties: {
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
        max_results: { type: "integer", description: "Maximum number of events to return" },
      },
    },
  },
};

export const GoogleUpdateCalendarEvent: Story = {
  name: "Google: Update Calendar Event",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Update Calendar Event",
    actionType: "google.update_calendar_event",
    agentName: "Chiedobot",
    riskLevel: "medium",
    parameters: {
      event_id: "abc123def456",
      calendar_id: "primary",
      summary: "Updated: Q1 Planning Review",
      start_time: "2026-03-20T15:00:00-05:00",
      end_time: "2026-03-20T16:00:00-05:00",
    },
    schema: {
      type: "object",
      required: ["event_id"],
      properties: {
        event_id: { type: "string", description: "The ID of the calendar event to update" },
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
        summary: { type: "string", description: "New event title" },
        start_time: { type: "string", description: "New start time in RFC 3339 format" },
        end_time: { type: "string", description: "New end time in RFC 3339 format" },
      },
    },
    preview: null,
    resourceDetails: { title: "Q1 Planning Review", start_time: "2026-03-20T14:00:00-05:00" },
  },
};

export const GoogleDeleteCalendarEvent: Story = {
  name: "Google: Delete Calendar Event",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Delete Calendar Event",
    actionType: "google.delete_calendar_event",
    agentName: "Chiedobot",
    riskLevel: "high",
    parameters: { event_id: "fifrnnp6iai8qi5klOhl9cspms", calendar_id: "primary" },
    schema: {
      type: "object",
      required: ["event_id"],
      properties: {
        event_id: { type: "string", description: "The ID of the calendar event to delete" },
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
      },
    },
    preview: null,
    resourceDetails: { title: "Q1 Planning Review", start_time: "2026-03-16T14:00:00-05:00" },
  },
};

export const GoogleCreateDocument: Story = {
  name: "Google: Create Document",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Create Document",
    actionType: "google.create_document",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Create document {{title}}",
    preview: null,
    parameters: { title: "Q1 Sprint Retrospective", content: "## What went well\n\n- Shipped API v2 on time\n- Zero downtime deployments\n\n## What could improve\n\n- Test coverage on edge cases" },
    schema: {
      type: "object",
      required: ["title"],
      properties: {
        title: { type: "string", description: "Title of the new Google Doc" },
        content: { type: "string", description: "Optional initial document body (plain text); same field as Update Document" },
      },
    },
  },
};

export const GoogleGetDocument: Story = {
  name: "Google: Get Document",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Get Document",
    actionType: "google.get_document",
    agentName: "Chiedobot",
    riskLevel: "low",
    displayTemplate: "Get document {{document_id}}",
    preview: null,
    parameters: { document_id: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms" },
    schema: {
      type: "object",
      required: ["document_id"],
      properties: {
        document_id: { type: "string", description: "The ID of the Google Doc to retrieve" },
      },
    },
    resourceDetails: { title: "Product Requirements Doc — v2" },
  },
};

export const GoogleUpdateDocument: Story = {
  name: "Google: Update Document",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Update Document",
    actionType: "google.update_document",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Update document {{document_id}} with new text",
    preview: null,
    parameters: {
      document_id: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms",
      content: "## New Section: Security Considerations\n\nAll API endpoints must validate JWT tokens before processing requests.",
    },
    schema: {
      type: "object",
      required: ["document_id", "content"],
      properties: {
        document_id: { type: "string", description: "The ID of the Google Doc to update" },
        content: { type: "string", description: "Text to insert into the document" },
        index: { type: "integer", description: "Character index to insert at (1-based)" },
      },
    },
    resourceDetails: { title: "Product Requirements Doc — v2" },
  },
};

export const GoogleSheetsReadRange: Story = {
  name: "Google: Read Spreadsheet Range",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Read Spreadsheet Range",
    actionType: "google.sheets_read_range",
    agentName: "Chiedobot",
    riskLevel: "low",
    displayTemplate: "Read {{range}} from spreadsheet",
    preview: null,
    parameters: { spreadsheet_id: "1abc2def3ghi4jkl", range: "Sheet1!A1:D10" },
    schema: {
      type: "object",
      required: ["spreadsheet_id", "range"],
      properties: {
        spreadsheet_id: { type: "string", description: "The ID of the spreadsheet" },
        range: { type: "string", description: "A1 notation range including sheet name" },
      },
    },
    resourceDetails: { title: "Q1 Revenue Tracker" },
  },
};

export const GoogleSheetsWriteRange: Story = {
  name: "Google: Write Spreadsheet Range",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Write Spreadsheet Range",
    actionType: "google.sheets_write_range",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Write to {{range}} in spreadsheet",
    preview: null,
    parameters: {
      spreadsheet_id: "1abc2def3ghi4jkl",
      range: "Sheet1!A1:C3",
      values: [["Name", "Email", "Role"], ["Alice", "alice@example.com", "Engineer"], ["Bob", "bob@example.com", "Designer"]],
    },
    schema: {
      type: "object",
      required: ["spreadsheet_id", "range", "values"],
      properties: {
        spreadsheet_id: { type: "string", description: "The ID of the spreadsheet" },
        range: { type: "string", description: "A1 notation range" },
        values: { type: "array", description: "2D array of cell values (rows of columns)" },
      },
    },
    resourceDetails: { title: "Team Directory" },
  },
};

export const GoogleSheetsAppendRows: Story = {
  name: "Google: Append Spreadsheet Rows",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Append Spreadsheet Rows",
    actionType: "google.sheets_append_rows",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Append rows to {{range}} in spreadsheet",
    preview: null,
    parameters: {
      spreadsheet_id: "1abc2def3ghi4jkl",
      range: "Sheet1",
      values: [["2026-03-16", "Deploy v2.4.1", "Completed"]],
    },
    schema: {
      type: "object",
      required: ["spreadsheet_id", "range", "values"],
      properties: {
        spreadsheet_id: { type: "string", description: "The ID of the spreadsheet" },
        range: { type: "string", description: "Sheet name or starting cell" },
        values: { type: "array", description: "2D array of row values to append" },
      },
    },
    resourceDetails: { title: "Deployment Log" },
  },
};

export const GoogleCreatePresentation: Story = {
  name: "Google: Create Presentation",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Create Presentation",
    actionType: "google.create_presentation",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Create presentation {{title}}",
    preview: null,
    parameters: { title: "Q1 2026 Company All-Hands" },
    schema: {
      type: "object",
      required: ["title"],
      properties: {
        title: { type: "string", description: "Title of the new presentation" },
      },
    },
  },
};

export const GoogleAddSlide: Story = {
  name: "Google: Add Slide",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Add Slide",
    actionType: "google.add_slide",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Add {{layout}} slide to presentation",
    preview: null,
    parameters: { presentation_id: "1pres2entation3id", layout: "TITLE_AND_BODY" },
    schema: {
      type: "object",
      required: ["presentation_id"],
      properties: {
        presentation_id: { type: "string", description: "The ID of the presentation" },
        layout: {
          type: "string",
          description: "Predefined slide layout",
          enum: ["BLANK", "TITLE", "TITLE_AND_BODY", "TITLE_ONLY", "SECTION_HEADER"],
          default: "BLANK",
        },
      },
    },
    resourceDetails: { title: "Q1 2026 Company All-Hands" },
  },
};

export const GoogleSendChatMessage: Story = {
  name: "Google: Send Chat Message",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Send Chat Message",
    actionType: "google.send_chat_message",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Send message to {{space_name}} — {{text}}",
    preview: null,
    parameters: { space_name: "spaces/AAAA1234", text: "Deploy completed successfully! All health checks passing." },
    schema: {
      type: "object",
      required: ["space_name", "text"],
      properties: {
        space_name: { type: "string", description: "The resource name of the space" },
        text: { type: "string", description: "The message text" },
      },
    },
  },
};

export const GoogleCreateMeeting: Story = {
  name: "Google: Create Meeting",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Create Meeting",
    actionType: "google.create_meeting",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Create meeting {{summary}} on {{start_time:datetime}} with {{attendees:count}} attendees",
    preview: null,
    parameters: {
      summary: "Design Review: Dashboard v2",
      start_time: "2026-03-17T10:00:00-05:00",
      end_time: "2026-03-17T10:30:00-05:00",
      attendees: ["alice@example.com", "bob@example.com"],
    },
    schema: {
      type: "object",
      required: ["summary", "start_time", "end_time"],
      properties: {
        summary: { type: "string", description: "Meeting title" },
        start_time: { type: "string", description: "Start time in RFC 3339 format" },
        end_time: { type: "string", description: "End time in RFC 3339 format" },
        attendees: { type: "array", description: "List of attendee email addresses" },
      },
    },
  },
};

export const GoogleDeleteDriveFile: Story = {
  name: "Google: Delete Drive File",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Delete Drive File",
    actionType: "google.delete_drive_file",
    agentName: "Chiedobot",
    riskLevel: "high",
    displayTemplate: "Delete Drive file {{file_id}}",
    preview: null,
    parameters: { file_id: "1abc2def3ghi4jkl5mno" },
    schema: {
      type: "object",
      required: ["file_id"],
      properties: {
        file_id: { type: "string", description: "The ID of the file to move to trash" },
      },
    },
    resourceDetails: { file_name: "old_budget_draft.xlsx" },
  },
};

export const GoogleUploadDriveFile: Story = {
  name: "Google: Upload Drive File",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Upload Drive File",
    actionType: "google.upload_drive_file",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Upload {{name}} to Drive",
    preview: null,
    parameters: { name: "meeting-notes-2026-03-16.md", content: "# Meeting Notes\n\n...", mime_type: "text/markdown" },
    schema: {
      type: "object",
      required: ["name", "content"],
      properties: {
        name: { type: "string", description: "File name" },
        content: { type: "string", description: "File content (text)" },
        mime_type: { type: "string", description: "MIME type", default: "text/plain" },
      },
    },
  },
};

export const GoogleReadEmail: Story = {
  name: "Google: Read Email",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Read Email",
    actionType: "google.read_email",
    agentName: "Chiedobot",
    riskLevel: "low",
    displayTemplate: "Read email {{message_id}}",
    preview: null,
    parameters: { message_id: "18e3a9f2b1c4d567" },
    schema: {
      type: "object",
      required: ["message_id"],
      properties: {
        message_id: { type: "string", description: "The Gmail message ID to read" },
        format: { type: "string", description: "Controls how much of the message is returned", enum: ["full", "metadata", "minimal"], default: "full" },
      },
    },
    resourceDetails: { subject: "Re: Q1 budget approval", from: "finance@company.com" },
  },
};

export const GoogleSendEmailReply: Story = {
  name: "Google: Reply to Email",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Reply to Email",
    actionType: "google.send_email_reply",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Reply to email thread {{thread_id}}",
    preview: null,
    parameters: { thread_id: "18e3a9f2b1c4d567", message_id: "18e3a9f2b1c4d567", body: "Thanks for the approval! I'll proceed with the vendor contract." },
    schema: {
      type: "object",
      required: ["thread_id", "message_id", "body"],
      properties: {
        thread_id: { type: "string", description: "The Gmail thread ID to reply to" },
        message_id: { type: "string", description: "The ID of the specific message to reply to" },
        body: { type: "string", description: "Reply body (plain text)" },
      },
    },
    resourceDetails: { subject: "Re: Q1 budget approval" },
  },
};

export const GoogleArchiveEmail: Story = {
  name: "Google: Archive Email",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Archive Email",
    actionType: "google.archive_email",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Archive email thread {{thread_id}}",
    preview: null,
    parameters: { thread_id: "18e3a9f2b1c4d567" },
    schema: {
      type: "object",
      required: ["thread_id"],
      properties: {
        thread_id: { type: "string", description: "The Gmail thread ID to archive" },
      },
    },
    resourceDetails: { subject: "Newsletter: March Updates", from: "newsletter@example.com" },
  },
};

export const GoogleCreateDriveFolder: Story = {
  name: "Google: Create Drive Folder",
  args: {
    connectorName: "Google",
    connectorLogo: GOOGLE_LOGO,
    actionName: "Create Drive Folder",
    actionType: "google.create_drive_folder",
    agentName: "Chiedobot",
    riskLevel: "medium",
    displayTemplate: "Create folder {{name}} in Drive",
    preview: null,
    parameters: { name: "Q2 2026 Planning" },
    schema: {
      type: "object",
      required: ["name"],
      properties: {
        name: { type: "string", description: "Name of the new folder" },
        parent_id: { type: "string", description: "Parent folder ID" },
      },
    },
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// GITHUB CONNECTOR — all actions
// ═══════════════════════════════════════════════════════════════════════════

export const GitHubCreateIssue: Story = {
  name: "GitHub: Create Issue",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Create Issue",
    actionType: "github.create_issue",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: {
      layout: "record",
      fields: { title: "title", subtitle: "repo" },
    },
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      title: "Add dark mode toggle to settings page",
      body: "## Summary\n\nImplement a dark mode toggle in the user settings page.\n\n## Acceptance Criteria\n- [ ] Toggle persists across sessions\n- [ ] Respects system preference by default",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "title"],
      properties: {
        owner: { type: "string", description: "Repository owner (user or organization)" },
        repo: { type: "string", description: "Repository name" },
        title: { type: "string", description: "Issue title" },
        body: { type: "string", description: "Issue body (Markdown supported)" },
      },
    },
  },
};

export const GitHubMergePR: Story = {
  name: "GitHub: Merge Pull Request",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Merge Pull Request",
    actionType: "github.merge_pr",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      pull_number: 142,
      merge_method: "squash",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "pull_number"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        pull_number: { type: "integer", description: "Pull request number" },
        merge_method: { type: "string", description: "Merge strategy", enum: ["merge", "squash", "rebase"], default: "merge" },
      },
    },
  },
};

export const GitHubCreatePR: Story = {
  name: "GitHub: Create Pull Request",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Create Pull Request",
    actionType: "github.create_pr",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      title: "feat: add dark mode toggle",
      head: "feature/dark-mode",
      base: "main",
      body: "Adds a dark mode toggle to user settings. Closes #87.",
      draft: false,
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "title", "head", "base"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        title: { type: "string", description: "Pull request title" },
        body: { type: "string", description: "Pull request body (Markdown supported)" },
        head: { type: "string", description: "Branch containing the changes" },
        base: { type: "string", description: "Branch to merge into" },
        draft: { type: "boolean", description: "Whether to create as a draft", default: false },
      },
    },
  },
};

export const GitHubAddReviewer: Story = {
  name: "GitHub: Add Reviewer",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Add Reviewer",
    actionType: "github.add_reviewer",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      pull_number: 142,
      reviewers: ["alice", "bob"],
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "pull_number", "reviewers"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        pull_number: { type: "integer", description: "Pull request number" },
        reviewers: { type: "array", description: "GitHub usernames to request reviews from" },
      },
    },
  },
};

export const GitHubCreateRelease: Story = {
  name: "GitHub: Create Release",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Create Release",
    actionType: "github.create_release",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      tag_name: "v2.4.1",
      name: "v2.4.1 — Bug fixes",
      body: "## Changes\n\n- Fix calendar event timezone handling\n- Improve Slack message formatting",
      draft: false,
      prerelease: false,
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "tag_name"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        tag_name: { type: "string", description: "The name of the tag" },
        name: { type: "string", description: "The name of the release" },
        body: { type: "string", description: "Release notes (Markdown supported)" },
        draft: { type: "boolean", description: "Whether to create as draft", default: false },
        prerelease: { type: "boolean", description: "Whether to mark as pre-release", default: false },
      },
    },
  },
};

export const GitHubCloseIssue: Story = {
  name: "GitHub: Close Issue",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Close Issue",
    actionType: "github.close_issue",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      issue_number: 87,
      state_reason: "completed",
      comment: "Closed by #142",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "issue_number"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        issue_number: { type: "integer", description: "Issue number to close" },
        state_reason: { type: "string", description: "Reason for closing", enum: ["completed", "not_planned"], default: "completed" },
        comment: { type: "string", description: "Optional comment to add before closing" },
      },
    },
  },
};

export const GitHubAddComment: Story = {
  name: "GitHub: Add Comment",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Add Comment",
    actionType: "github.add_comment",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      issue_number: 142,
      body: "LGTM! The implementation looks clean. One minor suggestion: consider adding a loading state for the toggle.",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "issue_number", "body"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        issue_number: { type: "integer", description: "Issue or pull request number" },
        body: { type: "string", description: "Comment body (Markdown supported)" },
      },
    },
  },
};

export const GitHubTriggerWorkflow: Story = {
  name: "GitHub: Trigger Workflow",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Trigger Workflow",
    actionType: "github.trigger_workflow",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      workflow_id: "deploy.yml",
      ref: "main",
      inputs: { environment: "staging" },
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "workflow_id", "ref"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        workflow_id: { type: "string", description: "Workflow file name or numeric ID" },
        ref: { type: "string", description: "Branch or tag to run the workflow on" },
        inputs: { type: "object", description: "Input key-value pairs" },
      },
    },
  },
};

export const GitHubCreateOrUpdateFile: Story = {
  name: "GitHub: Create or Update File",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Create or Update File",
    actionType: "github.create_or_update_file",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      path: "docs/setup.md",
      message: "docs: update setup guide for v2.4",
      content: "IyBTZXR1cCBHdWlkZQ==",
      branch: "main",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "path", "message", "content"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        path: { type: "string", description: "File path within the repository" },
        message: { type: "string", description: "Commit message" },
        content: { type: "string", description: "New file content, base64-encoded" },
        branch: { type: "string", description: "Branch to commit to" },
      },
    },
  },
};

export const GitHubCreateWebhook: Story = {
  name: "GitHub: Create Webhook",
  args: {
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    actionName: "Create Webhook",
    actionType: "github.create_webhook",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      url: "https://hooks.example.com/github",
      events: ["push", "pull_request"],
      content_type: "json",
      active: true,
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "url", "events"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        url: { type: "string", description: "Payload URL" },
        events: { type: "array", description: "Events that trigger the webhook" },
        content_type: { type: "string", description: "Payload content type", enum: ["json", "form"], default: "json" },
        active: { type: "boolean", description: "Whether the webhook is active", default: true },
      },
    },
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// SLACK CONNECTOR — all actions
// ═══════════════════════════════════════════════════════════════════════════

export const SlackSendMessage: Story = {
  name: "Slack: Send Message",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Send Message",
    actionType: "slack.send_message",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: {
      layout: "message",
      fields: { to: "channel", body: "message" },
    },
    parameters: {
      channel: "#engineering",
      message: "Deploy v2.4.1 completed successfully :white_check_mark: All health checks passing. Release notes: https://github.com/supersuit-tech/permission-slip/releases/tag/v2.4.1",
    },
    schema: {
      type: "object",
      required: ["channel", "message"],
      properties: {
        channel: { type: "string", description: "Channel name or ID" },
        message: { type: "string", description: "Message text (supports Slack mrkdwn)" },
      },
    },
  },
};

export const SlackCreateChannel: Story = {
  name: "Slack: Create Channel",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Create Channel",
    actionType: "slack.create_channel",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: { name: "project-phoenix", is_private: false },
    schema: {
      type: "object",
      required: ["name"],
      properties: {
        name: { type: "string", description: "Channel name (lowercase, no spaces)" },
        is_private: { type: "boolean", description: "Create as a private channel", default: false },
      },
    },
  },
};

export const SlackCreatePrivateChannel: Story = {
  name: "Slack: Create Private Channel",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Create Channel",
    actionType: "slack.create_channel",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: { name: "security-incidents", is_private: true },
    schema: {
      type: "object",
      required: ["name"],
      properties: {
        name: { type: "string", description: "Channel name" },
        is_private: { type: "boolean", description: "Create as a private channel", default: false },
      },
    },
  },
};

export const SlackListChannels: Story = {
  name: "Slack: List Channels",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "List Channels",
    actionType: "slack.list_channels",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { types: "public_channel", limit: 100, exclude_archived: true },
    schema: {
      type: "object",
      properties: {
        types: { type: "string", description: "Comma-separated channel types", default: "public_channel" },
        limit: { type: "integer", description: "Max channels to return", default: 100 },
        exclude_archived: { type: "boolean", description: "Exclude archived channels", default: true },
      },
    },
  },
};

export const SlackReadChannelMessages: Story = {
  name: "Slack: Read Channel Messages",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Read Channel Messages",
    actionType: "slack.read_channel_messages",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { channel: "C01234567", limit: 20 },
    schema: {
      type: "object",
      required: ["channel"],
      properties: {
        channel: { type: "string", description: "Channel ID" },
        limit: { type: "integer", description: "Max messages to return", default: 20 },
      },
    },
  },
};

export const SlackScheduleMessage: Story = {
  name: "Slack: Schedule Message",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Schedule Message",
    actionType: "slack.schedule_message",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: {
      channel: "#standup",
      message: "Good morning team! Time for our daily standup. What are you working on today?",
      post_at: 1742302800,
    },
    schema: {
      type: "object",
      required: ["channel", "message", "post_at"],
      properties: {
        channel: { type: "string", description: "Channel name or ID" },
        message: { type: "string", description: "Message text" },
        post_at: { type: "integer", description: "Unix timestamp for delivery" },
      },
    },
  },
};

export const SlackSetTopic: Story = {
  name: "Slack: Set Topic",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Set Topic",
    actionType: "slack.set_topic",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: { channel: "C01234567", topic: "Sprint 14 — Focus: API migration & dashboard v2" },
    schema: {
      type: "object",
      required: ["channel", "topic"],
      properties: {
        channel: { type: "string", description: "Channel ID" },
        topic: { type: "string", description: "New channel topic (max 250 characters)" },
      },
    },
  },
};

export const SlackInviteToChannel: Story = {
  name: "Slack: Invite to Channel",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Invite to Channel",
    actionType: "slack.invite_to_channel",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: { channel: "C01234567", users: "U01234567,U09876543" },
    schema: {
      type: "object",
      required: ["channel", "users"],
      properties: {
        channel: { type: "string", description: "Channel ID" },
        users: { type: "string", description: "Comma-separated list of user IDs to invite" },
      },
    },
  },
};

export const SlackSendDM: Story = {
  name: "Slack: Send Direct Message",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Send Direct Message",
    actionType: "slack.send_dm",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { user_id: "U01234567", message: "Hey! The deploy went through. Can you check the staging environment?" },
    schema: {
      type: "object",
      required: ["user_id", "message"],
      properties: {
        user_id: { type: "string", description: "User ID to send the DM to" },
        message: { type: "string", description: "Message text" },
      },
    },
  },
};

export const SlackUpdateMessage: Story = {
  name: "Slack: Update Message",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Update Message",
    actionType: "slack.update_message",
    agentName: "Chiedobot",
    riskLevel: "medium",
    preview: null,
    parameters: { channel: "C01234567", ts: "1742302800.123456", message: "Updated: Deploy v2.4.1 completed with one hotfix applied." },
    schema: {
      type: "object",
      required: ["channel", "ts", "message"],
      properties: {
        channel: { type: "string", description: "Channel ID" },
        ts: { type: "string", description: "Timestamp of the message to update" },
        message: { type: "string", description: "New message text" },
      },
    },
  },
};

export const SlackDeleteMessage: Story = {
  name: "Slack: Delete Message",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Delete Message",
    actionType: "slack.delete_message",
    agentName: "Chiedobot",
    riskLevel: "high",
    preview: null,
    parameters: { channel: "C01234567", ts: "1742302800.123456" },
    schema: {
      type: "object",
      required: ["channel", "ts"],
      properties: {
        channel: { type: "string", description: "Channel ID containing the message" },
        ts: { type: "string", description: "Timestamp of the message to delete" },
      },
    },
  },
};

export const SlackUploadFile: Story = {
  name: "Slack: Upload File",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Upload File",
    actionType: "slack.upload_file",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { channel: "C01234567", filename: "test-report.csv", content: "test,status,duration\nlogin,pass,1.2s\ncheckout,pass,3.4s", title: "CI Test Report" },
    schema: {
      type: "object",
      required: ["channel", "filename", "content"],
      properties: {
        channel: { type: "string", description: "Channel ID" },
        filename: { type: "string", description: "Name of the file" },
        content: { type: "string", description: "File content as text" },
        title: { type: "string", description: "Display title for the file" },
      },
    },
  },
};

export const SlackAddReaction: Story = {
  name: "Slack: Add Reaction",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Add Reaction",
    actionType: "slack.add_reaction",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { channel: "C01234567", timestamp: "1742302800.123456", name: "white_check_mark" },
    schema: {
      type: "object",
      required: ["channel", "timestamp", "name"],
      properties: {
        channel: { type: "string", description: "Channel ID containing the message" },
        timestamp: { type: "string", description: "Timestamp of the message to react to" },
        name: { type: "string", description: "Emoji name without colons" },
      },
    },
  },
};

export const SlackSearchMessages: Story = {
  name: "Slack: Search Messages",
  args: {
    connectorName: "Slack",
    connectorLogo: SLACK_LOGO,
    actionName: "Search Messages",
    actionType: "slack.search_messages",
    agentName: "Chiedobot",
    riskLevel: "low",
    preview: null,
    parameters: { query: "in:#engineering deploy", count: 20, page: 1 },
    schema: {
      type: "object",
      required: ["query"],
      properties: {
        query: { type: "string", description: "Search query (supports Slack search modifiers)" },
        count: { type: "integer", description: "Max results per page", default: 20 },
        page: { type: "integer", description: "Page number", default: 1 },
      },
    },
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// UI CONCEPTS — Parameters toggle + line-height fixes
// ═══════════════════════════════════════════════════════════════════════════

// Shared email params for concepts
const CONCEPT_EMAIL_PARAMS = {
  to: "alice@example.com",
  subject: "Weekly standup notes — March 16",
  body: "Hi Alice,\n\nHere are this week's standup notes. The team made great progress on the API migration and the new dashboard is nearly ready for QA.\n\nBest,\nChiedobot",
};

const CONCEPT_EMAIL_SCHEMA = {
  type: "object" as const,
  required: ["to", "subject", "body"],
  properties: {
    to: { type: "string" as const, description: "Recipient email address" },
    subject: { type: "string" as const, description: "Email subject line" },
    body: { type: "string" as const, description: "Email body (plain text)" },
  },
};

// ---------------------------------------------------------------------------
// Concept A: "Pill Toggle" — small pill button left-aligned, box below
// ---------------------------------------------------------------------------

function ConceptA() {
  const [open, setOpen] = useState(false);
  const params = CONCEPT_EMAIL_PARAMS;
  const schema = CONCEPT_EMAIL_SCHEMA;

  return (
    <Dialog open>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <ConnectorLogo name="Google" logoSvg={GOOGLE_LOGO} size="lg" />
            <div className="min-w-0">
              <DialogTitle className="truncate text-base">Send Email</DialogTitle>
              <p className="text-muted-foreground text-sm">Google</p>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4 sm:space-y-5">
          {/* Agent row */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-violet-100">
                <span className="text-sm font-bold text-violet-700">CB</span>
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">Chiedobot</p>
                <p className="text-muted-foreground text-xs">wants to perform an action</p>
              </div>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">Pending</Badge>
          </div>

          {/* Action header */}
          <div className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="truncate text-sm font-semibold">Send Email</span>
              <div className="flex items-center gap-2">
                <RiskBadge level="medium" />
                <span className="text-muted-foreground inline-flex shrink-0 items-center gap-1 text-xs font-medium">
                  <Clock className="size-3" />
                  9:30
                </span>
              </div>
            </div>

            {/* CONCEPT A: Email preview with line-height fix */}
            <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2">
                <div className="flex size-8 items-center justify-center rounded-lg bg-blue-50 ring-1 ring-blue-200">
                  <svg className="size-4 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
                </div>
                <span className="text-muted-foreground text-xs font-medium">Email</span>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-xs font-medium">To</span>
                  <span className="truncate text-sm font-medium">{params.to}</span>
                </div>
                <p className="truncate text-base font-semibold">{params.subject}</p>
                {/* FIX: whitespace-pre-line preserves \n, leading-relaxed adds breathing room */}
                <p className="text-muted-foreground text-sm leading-relaxed whitespace-pre-line line-clamp-4">
                  {params.body}
                </p>
              </div>
            </div>
          </div>

          {/* CONCEPT A: Parameters — pill button toggle, clean rounded box */}
          <div className="space-y-2">
            <button
              type="button"
              onClick={() => setOpen(!open)}
              className="text-muted-foreground hover:text-foreground hover:bg-muted/50 inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
            >
              <span
                className="transition-transform duration-150"
                style={{ display: "inline-block", transform: open ? "rotate(90deg)" : "rotate(0deg)" }}
                aria-hidden="true"
              >
                ▸
              </span>
              Parameters
            </button>
            {open && (
              <div className="bg-muted/30 overflow-x-auto rounded-xl border p-3 sm:p-4">
                <SchemaParameterDetails parameters={params} schema={schema} />
              </div>
            )}
          </div>

          {/* Buttons */}
          <div className="space-y-2 pt-2">
            <Button size="lg" className="w-full bg-emerald-600 text-white hover:bg-emerald-700">
              <Check className="mr-1 size-4" /> Approve
            </Button>
            <Button variant="outline" size="lg" className="w-full">Deny</Button>
            <button
              className="text-muted-foreground hover:text-foreground flex w-full items-center justify-center gap-1 py-1 text-sm transition-colors"
              type="button"
            >
              <ShieldCheck className="size-3" />
              Always allow this action
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Concept B: "Boxed Section" — whole parameters area is one rounded box
// ---------------------------------------------------------------------------

function ConceptB() {
  const [open, setOpen] = useState(false);
  const params = CONCEPT_EMAIL_PARAMS;
  const schema = CONCEPT_EMAIL_SCHEMA;

  return (
    <Dialog open>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <ConnectorLogo name="Google" logoSvg={GOOGLE_LOGO} size="lg" />
            <div className="min-w-0">
              <DialogTitle className="truncate text-base">Send Email</DialogTitle>
              <p className="text-muted-foreground text-sm">Google</p>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4 sm:space-y-5">
          {/* Agent row */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-violet-100">
                <span className="text-sm font-bold text-violet-700">CB</span>
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">Chiedobot</p>
                <p className="text-muted-foreground text-xs">wants to perform an action</p>
              </div>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">Pending</Badge>
          </div>

          {/* Action header */}
          <div className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="truncate text-sm font-semibold">Send Email</span>
              <div className="flex items-center gap-2">
                <RiskBadge level="medium" />
                <span className="text-muted-foreground inline-flex shrink-0 items-center gap-1 text-xs font-medium">
                  <Clock className="size-3" />
                  9:30
                </span>
              </div>
            </div>

            {/* CONCEPT B: Email preview with line-height fix */}
            <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
              <div className="mb-3 flex items-center gap-2">
                <div className="flex size-8 items-center justify-center rounded-lg bg-blue-50 ring-1 ring-blue-200">
                  <svg className="size-4 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect width="20" height="16" x="2" y="4" rx="2"/><path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"/></svg>
                </div>
                <span className="text-muted-foreground text-xs font-medium">Email</span>
              </div>
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-xs font-medium">To</span>
                  <span className="truncate text-sm font-medium">{params.to}</span>
                </div>
                <p className="truncate text-base font-semibold">{params.subject}</p>
                {/* FIX: whitespace-pre-line + leading-loose for clear paragraph breaks */}
                <p className="text-muted-foreground text-sm leading-loose whitespace-pre-line line-clamp-4">
                  {params.body}
                </p>
              </div>
            </div>
          </div>

          {/* CONCEPT B: Parameters — single rounded box, toggle is the header row */}
          <div className="overflow-hidden rounded-xl border">
            {/* Toggle header — full-width button, rounded corners only if closed */}
            <button
              type="button"
              onClick={() => setOpen(!open)}
              className="bg-muted/40 hover:bg-muted/70 flex w-full items-center justify-between px-4 py-2.5 text-left transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
            >
              <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
                Parameters
              </span>
              <span
                className="text-muted-foreground transition-transform duration-150"
                style={{ display: "inline-block", transform: open ? "rotate(90deg)" : "rotate(0deg)" }}
                aria-hidden="true"
              >
                ▸
              </span>
            </button>
            {/* Content — separated by inner border */}
            {open && (
              <div className="border-t p-3 sm:p-4">
                <SchemaParameterDetails parameters={params} schema={schema} />
              </div>
            )}
          </div>

          {/* Buttons */}
          <div className="space-y-2 pt-2">
            <Button size="lg" className="w-full bg-emerald-600 text-white hover:bg-emerald-700">
              <Check className="mr-1 size-4" /> Approve
            </Button>
            <Button variant="outline" size="lg" className="w-full">Deny</Button>
            <button
              className="text-muted-foreground hover:text-foreground flex w-full items-center justify-center gap-1 py-1 text-sm transition-colors"
              type="button"
            >
              <ShieldCheck className="size-3" />
              Always allow this action
            </button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export const UIConceptA: Story = {
  name: "UI Concept A: Pill Toggle",
  render: () => <ConceptA />,
  args: {} as never,
};

export const UIConceptB: Story = {
  name: "UI Concept B: Boxed Section",
  render: () => <ConceptB />,
  args: {} as never,
};
