/**
 * CreateStandingApprovalDialog — Storybook stories
 *
 * Renders each step of the "Create Standing Approval" wizard.
 * Step 3 uses the real StepConstraints component (ConstraintScenariosEditor).
 */
import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { ChevronLeft, ChevronRight, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ConnectorLogo } from "@/components/ConnectorLogo";
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
  StepConstraints,
  StepLimits,
} from "./StandingApprovalSteps";
import type { AgentActionOption } from "@/hooks/useAgentConnectorActions";
import type { ParametersSchema } from "@/lib/parameterSchema";
import {
  constraintsToFormState,
  emptyConstraintRow,
  type StructuredConstraintFormState,
} from "@/lib/structuredConstraints";
import { parseDataWindowFormState } from "@/lib/dataWindow";

const GOOGLE_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>Google</title><path d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"/></svg>`;

const GITHUB_LOGO = `<svg role="img" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><title>GitHub</title><path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/></svg>`;

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
];

const MOCK_ACTIONS_BY_CONNECTOR: Record<string, AgentActionOption[]> = {
  "google-calendar": [
    {
      connector_id: "google-calendar",
      action_type: "google_calendar.create_event",
      name: "Create Event",
    },
  ],
  github: [
    {
      connector_id: "github",
      action_type: "github.create_issue",
      name: "Create Issue",
    },
  ],
};

const CALENDAR_SCHEMA: ParametersSchema = {
  type: "object",
  required: ["summary", "start_time", "end_time"],
  properties: {
    summary: { type: "string", description: "Event title/summary", "x-ui": { label: "Title" } },
    start_time: { type: "string", description: "Start time in RFC 3339 format" },
    end_time: { type: "string", description: "End time in RFC 3339 format" },
    calendar_id: { type: "string", description: "Calendar ID" },
    description: { type: "string", description: "Event description" },
    attendees: { type: "array", description: "Attendee emails" },
  },
};

const GITHUB_ISSUE_SCHEMA: ParametersSchema = {
  type: "object",
  required: ["repo", "title"],
  properties: {
    repo: { type: "string", description: "Repository in owner/name format" },
    title: { type: "string", description: "Issue title" },
    body: { type: "string", description: "Issue body" },
    labels: { type: "array", description: "Labels" },
    assignees: { type: "array", description: "Assignees" },
  },
};

function defaultExpiresAt(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

function calendarEditFormState(): StructuredConstraintFormState {
  const form = constraintsToFormState(null);
  const scenario = form.scenarios[0];
  if (scenario) {
    scenario.paramRows = {
      summary: [{ ...emptyConstraintRow(), value: "Team Standup", mode: "fixed" }],
      calendar_id: [{ ...emptyConstraintRow(), value: "primary", mode: "fixed" }],
      attendees: [{ ...emptyConstraintRow(), value: "*", mode: "wildcard" }],
    };
  }
  return form;
}

function multiScenarioFormState(): StructuredConstraintFormState {
  const form = constraintsToFormState({
    $version: 2,
    match: "any",
    groups: [
      {
        match: "all",
        conditions: [
          { field: "repo", op: "matches", value: "supersuit-tech/webapp" },
          { field: "title", op: "matches", value: { $pattern: "*bug*" } },
        ],
      },
      {
        match: "all",
        conditions: [
          { field: "channel", op: "matches", value: "#incidents" },
        ],
      },
    ],
  });
  return form;
}

type Step = 1 | 2 | 3 | 4;
const STEP_LABELS: Record<Step, string> = {
  1: "Pick Agent",
  2: "Pick Action",
  3: "Set Constraints",
  4: "Expiry",
};

function CreateStandingApprovalWizard({
  initialStep = 1,
  connectorName,
  connectorLogo,
  schema,
  initialConstraintForm,
  metaFields = [],
}: {
  initialStep?: Step;
  connectorName?: string;
  connectorLogo?: string;
  schema?: ParametersSchema | null;
  initialConstraintForm?: StructuredConstraintFormState;
  metaFields?: string[];
}) {
  const [step, setStep] = useState<Step>(initialStep);
  const [agentId, setAgentId] = useState<number | "">(initialStep > 1 ? 1 : "");
  const [selectedActionType, setSelectedActionType] = useState(
    initialStep > 1 ? "google_calendar.create_event" : "",
  );
  const [constraintForm, setConstraintForm] = useState<StructuredConstraintFormState>(
    () => initialConstraintForm ?? constraintsToFormState(null),
  );
  const [manualConstraintsJson, setManualConstraintsJson] = useState("");
  const [dataWindowForm, setDataWindowForm] = useState(parseDataWindowFormState(null));
  const [noExpiry, setNoExpiry] = useState(true);
  const [expiresAt, setExpiresAt] = useState(defaultExpiresAt);

  const effectiveSchema = schema ?? (step >= 3 ? CALENDAR_SCHEMA : null);
  const effectiveActionType = selectedActionType;

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
              selectedActionType={selectedActionType}
              onActionChange={setSelectedActionType}
              actionsByConnector={MOCK_ACTIONS_BY_CONNECTOR}
              actionsLoading={false}
            />
          )}

          {step === 3 && (
            <StepConstraints
              configSchema={effectiveSchema}
              schemaLoading={false}
              constraintForm={constraintForm}
              onConstraintFormChange={setConstraintForm}
              metaFields={metaFields}
              manualConstraintsJson={manualConstraintsJson}
              onManualConstraintsJsonChange={setManualConstraintsJson}
              dataWindowForm={dataWindowForm}
              onDataWindowFormChange={setDataWindowForm}
              isPending={false}
            />
          )}

          {step === 4 && (
            <StepLimits
              expiresAt={expiresAt}
              onExpiresAtChange={setExpiresAt}
              noExpiry={noExpiry}
              onNoExpiryChange={setNoExpiry}
            />
          )}

          <DialogFooter className="gap-2 sm:gap-0">
            {step > 1 && (
              <Button type="button" variant="outline" onClick={() => setStep((step - 1) as Step)}>
                <ChevronLeft className="size-4" />
                Back
              </Button>
            )}
            <div className="flex-1" />
            <Button type="button" variant="secondary">
              Cancel
            </Button>
            {step < 4 ? (
              <Button type="button" onClick={() => setStep((step + 1) as Step)}>
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

const meta: Meta<typeof CreateStandingApprovalWizard> = {
  title: "Dashboard/CreateStandingApprovalDialog",
  component: CreateStandingApprovalWizard,
  parameters: { layout: "centered" },
};
export default meta;

type Story = StoryObj<typeof CreateStandingApprovalWizard>;

export const FullWizard: Story = { args: { initialStep: 1 } };

export const Step1PickAgent: Story = {
  args: { initialStep: 1 },
  name: "Step 1 – Pick Agent",
};

export const Step2PickAction: Story = {
  args: { initialStep: 2 },
  name: "Step 2 – Pick Action",
};

export const Step3ConstraintsWithSchema: Story = {
  args: {
    initialStep: 3,
    schema: CALENDAR_SCHEMA,
    connectorName: "Google Calendar",
    connectorLogo: GOOGLE_LOGO,
  },
  name: "Step 3 – Constraints (Schema)",
};

export const Step3ConstraintsGitHub: Story = {
  args: {
    initialStep: 3,
    schema: GITHUB_ISSUE_SCHEMA,
    connectorName: "GitHub",
    connectorLogo: GITHUB_LOGO,
    initialConstraintForm: multiScenarioFormState(),
  },
  name: "Step 3 – Constraints (Multi-scenario)",
};

export const Step3ConstraintsManualJson: Story = {
  args: {
    initialStep: 3,
    schema: null,
    connectorName: "Custom Connector",
  },
  name: "Step 3 – Constraints (Manual JSON)",
};

export const Step4Limits: Story = {
  args: {
    initialStep: 4,
    connectorName: "Google Calendar",
    connectorLogo: GOOGLE_LOGO,
  },
  name: "Step 4 – Expiry",
};

function EditStandingApprovalWizard() {
  const [step, setStep] = useState<3 | 4>(3);
  const [constraintForm, setConstraintForm] = useState(calendarEditFormState);
  const [noExpiry, setNoExpiry] = useState(false);
  const [expiresAt, setExpiresAt] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() + 14);
    const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
  });

  return (
    <Dialog open>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <ConnectorLogo name="Google Calendar" logoSvg={GOOGLE_LOGO} size="lg" />
            <div className="min-w-0">
              <DialogTitle className="truncate text-base">Create Event</DialogTitle>
              <p className="text-muted-foreground text-sm">Google Calendar</p>
            </div>
          </div>
          <DialogDescription>
            Step {step - 2} of 2: {STEP_LABELS[step]}
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-1 px-1">
          {([3, 4] as const).map((s) => (
            <div
              key={s}
              className={`h-1.5 flex-1 rounded-full transition-colors ${
                s <= step ? "bg-primary" : "bg-muted"
              }`}
            />
          ))}
        </div>

        <div className="space-y-4">
          {step === 3 && (
            <StepConstraints
              configSchema={CALENDAR_SCHEMA}
              schemaLoading={false}
              constraintForm={constraintForm}
              onConstraintFormChange={setConstraintForm}
              metaFields={[]}
              manualConstraintsJson=""
              onManualConstraintsJsonChange={() => {}}
              dataWindowForm={parseDataWindowFormState(null)}
              onDataWindowFormChange={() => {}}
              isPending={false}
            />
          )}

          {step === 4 && (
            <StepLimits
              expiresAt={expiresAt}
              onExpiresAtChange={setExpiresAt}
              noExpiry={noExpiry}
              onNoExpiryChange={setNoExpiry}
            />
          )}

          <DialogFooter className="gap-2 sm:gap-0">
            {step === 4 && (
              <Button type="button" variant="outline" onClick={() => setStep(3)}>
                <ChevronLeft className="size-4" />
                Back
              </Button>
            )}
            <div className="flex-1" />
            <Button type="button" variant="secondary">
              Cancel
            </Button>
            {step === 3 ? (
              <Button type="button" onClick={() => setStep(4)}>
                Next
                <ChevronRight className="size-4" />
              </Button>
            ) : (
              <Button type="button">
                <Check className="size-4" />
                Save
              </Button>
            )}
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export const EditMode: StoryObj = {
  render: () => <EditStandingApprovalWizard />,
  name: "Edit Mode",
};
