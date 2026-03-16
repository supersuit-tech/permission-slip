/**
 * Approval Dialog — Design Concept Variants
 *
 * Three alternative designs for the approval dialog addressing:
 * 1. The sun icon (removed / replaced with meaningful alternatives)
 * 2. The dark preview card (light-mode appropriate)
 * 3. The "Raw parameters" toggle (polished, intentional)
 */
import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import {
  Check,
  Clock,
  ShieldCheck,
  Mail,
  Bot,
  ChevronDown,
  ChevronRight,

  Eye,
  EyeOff,
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
import { SchemaParameterDetails } from "@/components/SchemaParameterDetails";
import { RiskBadge } from "./approval-components";
import { getInitials } from "@/components/ui/avatar";
import type { ParametersSchema } from "@/lib/parameterSchema";

const meta: Meta = {
  title: "Approval Dialog/Design Concepts",
  parameters: { layout: "centered" },
};
export default meta;

const GOOGLE_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>Google</title><path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/></svg>`;

const emailParams = {
  to: "alice@example.com",
  subject: "Weekly standup notes — March 16",
  body: "Hi Alice,\n\nHere are this week's standup notes.\n\nThe team made great progress on the API migration. All endpoints are now versioned and the legacy routes have been deprecated with a 90-day sunset window.\n\nThe new dashboard is ready for QA — please review it before Thursday's release window.\n\nBest,\nChiedobot",
};

const emailSchema: ParametersSchema = {
  type: "object",
  required: ["to", "subject", "body"],
  properties: {
    to: { type: "string", description: "Recipient email address" },
    subject: { type: "string", description: "Email subject line" },
    body: { type: "string", description: "Email body (plain text)" },
  },
};

// ---------------------------------------------------------------------------
// Shared sub-components used by all three concepts
// ---------------------------------------------------------------------------

/** Approve / Deny / Always-allow button cluster shared across all concepts. */
function DialogActionButtons() {
  return (
    <div className="space-y-2 pt-2">
      <Button size="lg" className="w-full bg-emerald-600 text-white hover:bg-emerald-700">
        <Check className="mr-1 size-4" />
        Approve
      </Button>
      <Button variant="outline" size="lg" className="w-full">Deny</Button>
      <button
        type="button"
        className="text-muted-foreground hover:text-foreground flex w-full items-center justify-center gap-1 py-1 text-sm transition-colors"
      >
        <ShieldCheck className="size-3" />
        Always allow this action
      </button>
    </div>
  );
}

/** Action name + risk badge + countdown row shared across all concepts. */
function ActionHeader({ actionName, riskLevel, countdown }: {
  actionName: string;
  riskLevel: "low" | "medium" | "high";
  countdown: string;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <span className="text-sm font-semibold">{actionName}</span>
      <div className="flex items-center gap-2">
        <RiskBadge level={riskLevel} />
        <span className="text-muted-foreground inline-flex items-center gap-1 text-xs font-medium">
          <Clock className="size-3" />
          {countdown}
        </span>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// CONCEPT A — "Clean Card"
// Agent shown as a subtle inline badge. Preview card uses a soft bordered
// container with proper light-mode colors. Raw params uses a styled
// button-like toggle with icon.
// ---------------------------------------------------------------------------

function ConceptA() {
  const [showRaw, setShowRaw] = useState(false);
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
          {/* Agent identity — clean inline row, no avatar */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <div className="bg-primary/10 flex size-8 shrink-0 items-center justify-center rounded-full">
                <Bot className="size-4 text-primary" />
              </div>
              <div>
                <p className="text-sm font-semibold">Chiedobot</p>
                <p className="text-muted-foreground text-xs">wants to perform an action</p>
              </div>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">Pending</Badge>
          </div>

          <ActionHeader actionName="Send Email" riskLevel="medium" countdown="9:30" />

          {/* Preview card — light, bordered, clean */}
          <div className="overflow-hidden rounded-xl border bg-gradient-to-b from-slate-50 to-white p-4 dark:from-slate-900 dark:to-slate-950 dark:border-slate-800">
            <div className="mb-3 flex items-center gap-2">
              <div className="flex size-8 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30">
                <Mail className="size-4 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground text-xs font-medium">To</span>
                <span className="truncate text-sm font-medium">{emailParams.to}</span>
              </div>
              <p className="truncate text-base font-semibold">{emailParams.subject}</p>
              <p className="text-muted-foreground line-clamp-2 text-sm leading-relaxed">
                Hi Alice, Here are this week's standup notes. The team made great progress on the API migration...
              </p>
            </div>
          </div>

          {/* Raw parameters — styled toggle button */}
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowRaw(!showRaw)}
              aria-expanded={showRaw}
              className="bg-background text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 rounded border px-3 py-1.5 text-xs font-medium transition-colors"
            >
              {showRaw ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
              {showRaw ? "Hide parameters" : "Show parameters"}
            </button>
          </div>
          {showRaw && (
            <div className="rounded-lg border bg-muted/30 p-3 sm:p-4">
              <SchemaParameterDetails parameters={emailParams} schema={emailSchema} />
            </div>
          )}

          <DialogActionButtons />
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// CONCEPT B — "Minimal"
// No separate agent row — agent name is part of the header. Preview card
// uses a white card with left accent border. Raw params is a pill toggle.
// ---------------------------------------------------------------------------

function ConceptB() {
  const [showRaw, setShowRaw] = useState(false);
  return (
    <Dialog open>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <ConnectorLogo name="Google" logoSvg={GOOGLE_LOGO} size="lg" />
            <div className="min-w-0 flex-1">
              <DialogTitle className="truncate text-base">Send Email</DialogTitle>
              <p className="text-muted-foreground text-sm">Google · requested by <span className="font-medium text-foreground">Chiedobot</span></p>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">Pending</Badge>
          </div>
        </DialogHeader>

        <div className="space-y-4 sm:space-y-5">
          <ActionHeader actionName="Send Email" riskLevel="medium" countdown="9:30" />

          {/* Preview card — left accent border, clean white bg */}
          <div className="overflow-hidden rounded-lg border-l-4 border-l-blue-500 border border-border bg-card p-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <Mail className="size-4 text-blue-500" />
                <span className="text-muted-foreground text-xs font-medium uppercase tracking-wide">Email Preview</span>
              </div>
              <div className="flex items-baseline gap-2 pt-1">
                <span className="text-muted-foreground text-xs font-medium shrink-0">To</span>
                <span className="truncate text-sm">{emailParams.to}</span>
              </div>
              <p className="text-base font-semibold">{emailParams.subject}</p>
              <p className="text-muted-foreground line-clamp-3 text-sm leading-relaxed">
                Hi Alice, Here are this week's standup notes. The team made great progress on the API migration...
              </p>
            </div>
          </div>

          {/* Raw parameters — pill toggle */}
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowRaw(!showRaw)}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${showRaw ? "bg-foreground text-background border-foreground" : "bg-background text-muted-foreground border-border hover:border-foreground/30 hover:text-foreground"}`}
            >
              {showRaw ? <EyeOff className="size-3" /> : <Eye className="size-3" />}
              {showRaw ? "Hide parameters" : "Show parameters"}
            </button>
          </div>
          {showRaw && (
            <div className="rounded-lg border bg-muted/30 p-3 sm:p-4">
              <SchemaParameterDetails parameters={emailParams} schema={emailSchema} />
            </div>
          )}

          <DialogActionButtons />
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// CONCEPT C — "Elevated"
// Agent shown with a colored initial avatar. Preview uses an elevated
// card with subtle shadow. Raw params integrated inline with a clean
// section divider and label.
// ---------------------------------------------------------------------------

function ConceptC() {
  const [showRaw, setShowRaw] = useState(false);
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
          {/* Agent identity — initial avatar */}
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30">
                <span className="text-sm font-bold text-violet-700 dark:text-violet-300">{getInitials("Chiedobot")}</span>
              </div>
              <div>
                <p className="text-sm font-semibold">Chiedobot</p>
                <p className="text-muted-foreground text-xs">wants to perform an action</p>
              </div>
            </div>
            <Badge variant="warning-soft" className="shrink-0 uppercase">Pending</Badge>
          </div>

          <ActionHeader actionName="Send Email" riskLevel="medium" countdown="9:30" />

          {/* Preview card — elevated with shadow */}
          <div className="overflow-hidden rounded-xl border bg-card p-4 shadow-sm">
            <div className="mb-3 flex items-center gap-2">
              <div className="flex size-8 items-center justify-center rounded-lg bg-blue-50 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800">
                <Mail className="size-4 text-blue-600 dark:text-blue-400" />
              </div>
              <span className="text-muted-foreground text-xs font-medium">Email</span>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground text-xs font-medium">To</span>
                <span className="truncate text-sm font-medium">{emailParams.to}</span>
              </div>
              <p className="truncate text-base font-semibold">{emailParams.subject}</p>
              <p className="text-muted-foreground line-clamp-2 text-sm leading-relaxed">
                Hi Alice, Here are this week's standup notes. The team made great progress on the API migration...
              </p>
            </div>
          </div>

          {/* Raw parameters — section divider with inline toggle */}
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="border-border w-full border-t" />
            </div>
            <div className="relative flex justify-center">
              <button
                type="button"
                onClick={() => setShowRaw(!showRaw)}
                className="bg-background text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 px-3 text-xs font-medium transition-colors"
              >
                {showRaw ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
                Parameters
              </button>
            </div>
          </div>
          {showRaw && (
            <div className="rounded-lg border bg-muted/30 p-3 sm:p-4">
              <SchemaParameterDetails parameters={emailParams} schema={emailSchema} />
            </div>
          )}

          <DialogActionButtons />
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Stories
// ---------------------------------------------------------------------------

export const Concept_A_CleanCard: StoryObj = {
  name: "Concept A — Clean Card",
  render: () => <ConceptA />,
};

export const Concept_B_Minimal: StoryObj = {
  name: "Concept B — Minimal",
  render: () => <ConceptB />,
};

export const Concept_C_Elevated: StoryObj = {
  name: "Concept C — Elevated",
  render: () => <ConceptC />,
};
