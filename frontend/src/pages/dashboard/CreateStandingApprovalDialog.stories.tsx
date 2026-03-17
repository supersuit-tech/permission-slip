/**
 * CreateStandingApprovalDialog — Storybook stories
 *
 * Renders each step of the "Create Standing Approval" wizard.
 * Steps 1, 2, and 4 use the real presentational sub-components.
 * Step 3 uses ParameterFieldWidget directly (leaf component, no Radix context issues).
 */
import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ChevronLeft, ChevronRight, Check, Info, Asterisk } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { Label } from "@/components/ui/label";
import { ParameterFieldWidget } from "@/pages/agents/connectors/ParameterFieldWidget";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  StepPickAgent,
  StepPickAction,
  StepLimits,
  CUSTOM_ACTION_SENTINEL,
} from "./StandingApprovalSteps";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ParametersSchema, SchemaProperty } from "@/lib/parameterSchema";

// ---------------------------------------------------------------------------
// SVG logos (embedded to avoid API calls)
// ---------------------------------------------------------------------------

const GOOGLE_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>Google</title><path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/></svg>`;

const GITHUB_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>GitHub</title><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`;

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

const MOCK_AGENTS = [
  {
    agent_id: 1,
    status: "registered" as const,
    metadata: { name: "Calendar Assistant" },
    created_at: "2026-03-01T00:00:00Z",
  },
  {
    agent_id: 2,
    status: "registered" as const,
    metadata: { name: "GitHub Bot" },
    created_at: "2026-03-02T00:00:00Z",
  },
  {
    agent_id: 3,
    status: "registered" as const,
    metadata: { name: "Slack Notifier" },
    created_at: "2026-03-03T00:00:00Z",
  },
];

const MOCK_CONFIGS_BY_CONNECTOR: Record<string, ActionConfiguration[]> = {
  "google-calendar": [
    {
      id: "cfg-1",
      agent_id: 1,
      name: "Create Event",
      action_type: "google_calendar.create_event",
      connector_id: "google-calendar",
      status: "active",
      parameters: {
        summary: "Team Standup",
        calendar_id: "primary",
        attendees: "*",
      },
      created_at: "2026-03-01T00:00:00Z",
      updated_at: "2026-03-01T00:00:00Z",
    },
    {
      id: "cfg-2",
      agent_id: 1,
      name: "Delete Event",
      action_type: "google_calendar.delete_event",
      connector_id: "google-calendar",
      status: "active",
      parameters: { calendar_id: "primary", event_id: "*" },
      created_at: "2026-03-01T00:00:00Z",
      updated_at: "2026-03-01T00:00:00Z",
    },
  ],
  github: [
    {
      id: "cfg-3",
      agent_id: 2,
      name: "Create Issue",
      action_type: "github.create_issue",
      connector_id: "github",
      status: "active",
      parameters: { repo: "supersuit-tech/permission-slip", title: "*" },
      created_at: "2026-03-02T00:00:00Z",
      updated_at: "2026-03-02T00:00:00Z",
    },
  ],
};

const CALENDAR_SCHEMA: ParametersSchema = {
  type: "object",
  required: ["summary", "start_time", "end_time"],
  properties: {
    attendees: {
      type: "array",
      description: "List of attendee email addresses",
    },
    calendar_id: {
      type: "string",
      description: "Calendar ID (defaults to 'primary')",
    },
    description: {
      type: "string",
      description: "Event description",
    },
    end_time: {
      type: "string",
      description:
        "End time in RFC 3339 format (e.g. '2024-01-15T10:00:00-05:00')",
    },
    start_time: {
      type: "string",
      description:
        "Start time in RFC 3339 format (e.g. '2024-01-15T09:00:00-05:00')",
    },
    summary: {
      type: "string",
      description: "Event title/summary",
      "x-ui": { label: "Title" },
    },
  },
};

const GITHUB_ISSUE_SCHEMA: ParametersSchema = {
  type: "object",
  required: ["repo", "title"],
  properties: {
    repo: {
      type: "string",
      description: "Repository in owner/name format",
    },
    title: {
      type: "string",
      description: "Issue title",
    },
    body: {
      type: "string",
      description: "Issue body (markdown)",
    },
    labels: {
      type: "array",
      description: "Labels to apply to the issue",
    },
    assignees: {
      type: "array",
      description: "GitHub usernames to assign",
    },
  },
};

function defaultExpiresAt(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

// ---------------------------------------------------------------------------
// Inline constraint field — mirrors ActionConfigParameterFields for Storybook
// ---------------------------------------------------------------------------

type ParamMode = "fixed" | "pattern" | "wildcard";

function StoryConstraintField({
  paramKey,
  property,
  isRequired,
  value,
  mode,
  onValueChange,
  onModeChange,
}: {
  paramKey: string;
  property: SchemaProperty;
  isRequired: boolean;
  value: string;
  mode: ParamMode;
  onValueChange: (key: string, value: string) => void;
  onModeChange: (key: string, mode: ParamMode) => void;
}) {
  const isWildcard = mode === "wildcard";
  // Use a clean human-readable label: snake_case → Title Case, with common acronym fixes
  const label = paramKey
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .replace(/\bId\b/g, "ID")
    .replace(/\bUrl\b/g, "URL");

  // In constraint context, datetime fields render as plain text — users enter
  // patterns like "2026-*" or RFC 3339 values, not pick from a calendar.
  const constraintProperty: typeof property =
    property.format === "date-time" ||
    property["x-ui"]?.widget === "datetime" ||
    (property.type === "string" && property.description &&
      (() => {
        const d = property.description.toLowerCase();
        return (d.includes("rfc 3339") || d.includes("rfc3339") || d.includes("iso 8601")) && !d.includes("epoch");
      })())
      ? { ...property, "x-ui": { ...(property["x-ui"] ?? {}), widget: "text" } }
      : property;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Label htmlFor={`param-${paramKey}`} className="text-sm font-medium">
            {label}
          </Label>
          {isRequired && (
            <Badge variant="secondary" className="text-xs">
              required
            </Badge>
          )}
        </div>
        <label className="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs whitespace-nowrap">
          <Checkbox
            checked={isWildcard}
            onCheckedChange={(checked) => {
              if (checked === true) {
                onModeChange(paramKey, "wildcard");
                onValueChange(paramKey, "*");
              } else if (checked === false) {
                onModeChange(paramKey, "fixed");
                onValueChange(paramKey, "");
              }
            }}
          />
          <Asterisk className="size-3" />
          Any value
        </label>
      </div>
      <div>
        <ParameterFieldWidget
          paramKey={paramKey}
          property={constraintProperty}
          value={isWildcard ? "" : value}
          onChange={(v) => onValueChange(paramKey, v)}
          disabled={isWildcard}
          placeholder={isWildcard ? "Agent can use any value" : undefined}
          className={isWildcard ? "bg-muted" : ""}
        />
      </div>
      {!isWildcard && value.includes("*") && (
        <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
          <p className="text-muted-foreground text-xs leading-relaxed">
            <code className="rounded bg-muted px-1 font-mono text-foreground/70">*</code> matches
            any text
          </p>
        </div>
      )}
    </div>
  );
}

function StoryStepConstraints({
  schema,
  paramValues,
  paramModes,
  onParamValueChange,
  onParamModeChange,
  manualConstraintsJson,
  onManualConstraintsJsonChange,
}: {
  schema: ParametersSchema | null;
  paramValues: Record<string, string>;
  paramModes: Record<string, ParamMode>;
  onParamValueChange: (key: string, value: string) => void;
  onParamModeChange: (key: string, mode: ParamMode) => void;
  manualConstraintsJson: string;
  onManualConstraintsJsonChange: (value: string) => void;
}) {
  const properties = schema?.properties;
  const requiredFields = schema?.required ?? [];

  return (
    <div className="space-y-3">
      <div className="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <p className="text-muted-foreground text-sm">
          At least one field must have a fixed value or pattern — not all wildcards.
        </p>
      </div>

      {properties ? (
        <div className="space-y-3">
          <Label>Constraints</Label>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              Use{" "}
              <code className="rounded bg-muted px-1 font-mono text-foreground/70">*</code> as a
              wildcard in any value (e.g.{" "}
              <code className="rounded bg-muted px-1 font-mono text-foreground/70">
                *@mycompany.com
              </code>
              ). Check <strong className="text-foreground/70">Any value</strong> to let the agent choose freely.
            </p>
          </div>
          {Object.entries(properties)
            .sort(([a], [b]) => {
              const aReq = requiredFields.includes(a) ? 0 : 1;
              const bReq = requiredFields.includes(b) ? 0 : 1;
              return aReq - bReq;
            })
            .map(([key, prop]) => (
              <StoryConstraintField
                key={key}
                paramKey={key}
                property={prop}
                isRequired={requiredFields.includes(key)}
                value={paramValues[key] ?? ""}
                mode={paramModes[key] ?? "fixed"}
                onValueChange={onParamValueChange}
                onModeChange={onParamModeChange}
              />
            ))}
        </div>
      ) : (
        <div className="space-y-2">
          <Label htmlFor="sa-manual-constraints">Constraints (JSON)</Label>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              No parameter schema found for this action. Enter constraints
              manually as a JSON object.
            </p>
          </div>
          <textarea
            id="sa-manual-constraints"
            className="border-input bg-background ring-offset-background focus-visible:ring-ring flex min-h-[100px] w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={
              '{\n  "recipient": "*@mycompany.com",\n  "subject": "*"\n}'
            }
            value={manualConstraintsJson}
            onChange={(e) => onManualConstraintsJsonChange(e.target.value)}
          />
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              Use{" "}
              <code className="rounded bg-muted px-1 font-mono text-foreground/70">&quot;*&quot;</code>{" "}
              for wildcard parameters, but at least one must be non-wildcard.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Interactive wizard story — renders all four steps with navigation
// ---------------------------------------------------------------------------

type Step = 1 | 2 | 3 | 4;
const STEP_LABELS: Record<Step, string> = {
  1: "Pick Agent",
  2: "Pick Action",
  3: "Set Constraints",
  4: "Set Limits",
};

function CreateStandingApprovalWizard({
  initialStep = 1,
  connectorName,
  connectorLogo,
  schema,
}: {
  initialStep?: Step;
  connectorName?: string;
  connectorLogo?: string;
  schema?: ParametersSchema | null;
}) {
  const [step, setStep] = useState<Step>(initialStep);
  const [agentId, setAgentId] = useState<number | "">(
    initialStep > 1 ? 1 : "",
  );
  const [selectedConfigId, setSelectedConfigId] = useState(
    initialStep > 1 ? "cfg-1" : "",
  );
  const [customActionType, setCustomActionType] = useState("");
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>({});
  const [manualConstraintsJson, setManualConstraintsJson] = useState("");
  const [maxExecutions, setMaxExecutions] = useState("");
  const [expiresAt, setExpiresAt] = useState(defaultExpiresAt);

  const isCustomAction = selectedConfigId === CUSTOM_ACTION_SENTINEL;
  const effectiveSchema = schema ?? (step >= 3 ? CALENDAR_SCHEMA : null);

  const effectiveActionType =
    selectedConfigId === "cfg-1"
      ? "google_calendar.create_event"
      : selectedConfigId === "cfg-3"
        ? "github.create_issue"
        : customActionType;

  return (
    <Dialog open>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          {effectiveActionType && step >= 2 ? (
            <>
              <div className="flex items-center gap-3">
                <ConnectorLogo
                  name={connectorName ?? "Google Calendar"}
                  logoSvg={connectorLogo ?? GOOGLE_LOGO}
                  size="lg"
                />
                <div className="min-w-0">
                  <DialogTitle className="truncate text-base">
                    {effectiveActionType === "google_calendar.create_event"
                      ? "Create Event"
                      : effectiveActionType}
                  </DialogTitle>
                  <p className="text-muted-foreground text-sm">
                    {connectorName ?? "Google Calendar"}
                  </p>
                </div>
              </div>
              <DialogDescription>
                Step {step} of 4: {STEP_LABELS[step]}
              </DialogDescription>
            </>
          ) : (
            <>
              <DialogTitle>Create Standing Approval</DialogTitle>
              <DialogDescription>
                Step {step} of 4: {STEP_LABELS[step]}
              </DialogDescription>
            </>
          )}
        </DialogHeader>

        <div className="flex items-center gap-1 px-1">
          {([1, 2, 3, 4] as Step[]).map((s) => (
            <div
              key={s}
              className={`h-1.5 flex-1 rounded-full transition-colors ${
                s <= step ? "bg-primary" : "bg-muted"
              }`}
            />
          ))}
        </div>

        <div className="space-y-4">
          {step === 1 && (
            <StepPickAgent
              agentId={agentId}
              onAgentChange={setAgentId}
              activeAgents={MOCK_AGENTS}
            />
          )}

          {step === 2 && (
            <StepPickAction
              selectedConfigId={selectedConfigId}
              onConfigChange={setSelectedConfigId}
              customActionType={customActionType}
              onCustomActionTypeChange={setCustomActionType}
              configsByConnector={MOCK_CONFIGS_BY_CONNECTOR}
              configsLoading={false}
              isCustomAction={isCustomAction}
            />
          )}

          {step === 3 && (
            <StoryStepConstraints
              schema={effectiveSchema}
              paramValues={paramValues}
              paramModes={paramModes}
              onParamValueChange={(key, value) =>
                setParamValues((prev) => ({ ...prev, [key]: value }))
              }
              onParamModeChange={(key, mode) =>
                setParamModes((prev) => ({ ...prev, [key]: mode }))
              }
              manualConstraintsJson={manualConstraintsJson}
              onManualConstraintsJsonChange={setManualConstraintsJson}
            />
          )}

          {step === 4 && (
            <StepLimits
              maxExecutions={maxExecutions}
              onMaxExecutionsChange={(value) => {
                if (value === "") {
                  setMaxExecutions("");
                  return;
                }
                const intValue = parseInt(value, 10);
                if (Number.isNaN(intValue) || intValue < 1) return;
                setMaxExecutions(String(intValue));
              }}
              expiresAt={expiresAt}
              onExpiresAtChange={setExpiresAt}
            />
          )}

          <DialogFooter className="gap-2 sm:gap-0">
            {step > 1 && (
              <Button
                type="button"
                variant="outline"
                onClick={() => setStep((step - 1) as Step)}
              >
                <ChevronLeft className="size-4" />
                Back
              </Button>
            )}
            <div className="flex-1" />
            <Button type="button" variant="secondary">
              Cancel
            </Button>
            {step < 4 ? (
              <Button
                type="button"
                onClick={() => setStep((step + 1) as Step)}
              >
                Next
                <ChevronRight className="size-4" />
              </Button>
            ) : (
              <Button type="button">
                <Check className="size-4" />
                Create
              </Button>
            )}
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Storybook meta
// ---------------------------------------------------------------------------

const meta: Meta<typeof CreateStandingApprovalWizard> = {
  title: "Dashboard/CreateStandingApprovalDialog",
  component: CreateStandingApprovalWizard,
  parameters: {
    layout: "centered",
  },
};
export default meta;

type Story = StoryObj<typeof CreateStandingApprovalWizard>;

// ---------------------------------------------------------------------------
// Stories — one per step, plus a full interactive wizard
// ---------------------------------------------------------------------------

/** Full interactive wizard starting from Step 1 (Pick Agent). */
export const FullWizard: Story = {
  args: { initialStep: 1 },
};

/** Step 1: Pick Agent — agent selection dropdown. */
export const Step1PickAgent: Story = {
  args: { initialStep: 1 },
  name: "Step 1 – Pick Agent",
};

/** Step 2: Pick Action — action configuration selection. */
export const Step2PickAction: Story = {
  args: { initialStep: 2 },
  name: "Step 2 – Pick Action",
};

/** Step 3: Set Constraints — with Google Calendar schema (Fixed/Pattern/Wildcard). */
export const Step3ConstraintsWithSchema: Story = {
  args: {
    initialStep: 3,
    schema: CALENDAR_SCHEMA,
    connectorName: "Google Calendar",
    connectorLogo: GOOGLE_LOGO,
  },
  name: "Step 3 – Constraints (Schema)",
};

/** Step 3: Set Constraints — GitHub issue schema variant. */
export const Step3ConstraintsGitHub: Story = {
  args: {
    initialStep: 3,
    schema: GITHUB_ISSUE_SCHEMA,
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
  },
  name: "Step 3 – Constraints (GitHub)",
};

/** Step 3: Set Constraints — manual JSON fallback (no schema). */
export const Step3ConstraintsManualJson: Story = {
  args: {
    initialStep: 3,
    schema: null,
    connectorName: "Custom Connector",
  },
  name: "Step 3 – Constraints (Manual JSON)",
};

/** Step 4: Set Limits — max executions and expiry. */
export const Step4Limits: Story = {
  args: {
    initialStep: 4,
    connectorName: "Google Calendar",
    connectorLogo: GOOGLE_LOGO,
  },
  name: "Step 4 – Limits",
};
