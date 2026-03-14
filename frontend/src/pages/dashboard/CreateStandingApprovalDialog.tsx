import { type FormEvent, useState, useMemo } from "react";
import { Loader2, ChevronLeft, ChevronRight, Check } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useActionConfigs } from "@/hooks/useActionConfigs";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { Agent } from "@/hooks/useAgents";
import {
  buildParametersFromForm,
  isPatternWrapper,
  type ParamMode,
} from "@/pages/agents/connectors/ActionConfigFormFields";
import {
  CUSTOM_ACTION_SENTINEL,
  StepPickAgent,
  StepPickAction,
  StepConstraints,
  StepLimits,
} from "./StandingApprovalSteps";

export interface CreateStandingApprovalDialogProps {
  agents: Agent[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialAgentId?: number;
  initialActionType?: string;
  initialConstraints?: Record<string, unknown>;
}

type Step = 1 | 2 | 3 | 4;
const STEP_LABELS: Record<Step, string> = {
  1: "Pick Agent",
  2: "Pick Action",
  3: "Set Constraints",
  4: "Set Limits",
};

function defaultExpiresAt(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

/**
 * Returns true when at least one parameter constraint is non-wildcard.
 */
function hasNonWildcardConstraint(
  paramValues: Record<string, string>,
  paramModes: Record<string, ParamMode>,
): boolean {
  for (const key of Object.keys(paramValues)) {
    const mode = paramModes[key] ?? "fixed";
    if (mode !== "wildcard") {
      const value = paramValues[key];
      if (value !== undefined && value !== "") {
        return true;
      }
    }
  }
  return false;
}

export function CreateStandingApprovalDialog({
  agents,
  open,
  onOpenChange,
  initialAgentId,
  initialActionType,
  initialConstraints,
}: CreateStandingApprovalDialogProps) {
  const { createStandingApproval, isPending } = useCreateStandingApproval();

  const [step, setStep] = useState<Step>(1);
  const [agentId, setAgentId] = useState<number | "">(initialAgentId ?? "");
  const [selectedConfigId, setSelectedConfigId] = useState<string>("");
  const [customActionType, setCustomActionType] = useState(
    initialActionType ?? "",
  );
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>({});
  const [manualConstraintsJson, setManualConstraintsJson] = useState("");
  const [maxExecutions, setMaxExecutions] = useState("");
  const [expiresAt, setExpiresAt] = useState(defaultExpiresAt);

  const activeAgents = agents.filter((a) => a.status !== "deactivated");

  const { configs, isLoading: configsLoading } = useActionConfigs(
    typeof agentId === "number" ? agentId : 0,
  );

  const activeConfigs = useMemo(
    () => configs.filter((c) => c.status === "active"),
    [configs],
  );

  const selectedConfig = useMemo(
    () =>
      selectedConfigId && selectedConfigId !== CUSTOM_ACTION_SENTINEL
        ? activeConfigs.find((c) => c.id === selectedConfigId) ?? null
        : null,
    [activeConfigs, selectedConfigId],
  );

  const isCustomAction = selectedConfigId === CUSTOM_ACTION_SENTINEL;

  const effectiveActionType = selectedConfig
    ? selectedConfig.action_type
    : customActionType;

  const { schema: fetchedSchema, isLoading: schemaLoading } =
    useActionSchema(effectiveActionType);

  const configSchema = fetchedSchema;

  const configsByConnector = useMemo(() => {
    const groups: Record<string, ActionConfiguration[]> = {};
    for (const config of activeConfigs) {
      const key = config.connector_id;
      if (!groups[key]) groups[key] = [];
      groups[key].push(config);
    }
    return groups;
  }, [activeConfigs]);

  function resetForm() {
    setStep(1);
    setAgentId(initialAgentId ?? "");
    setSelectedConfigId("");
    setCustomActionType(initialActionType ?? "");
    setParamValues({});
    setParamModes({});
    setManualConstraintsJson("");
    setMaxExecutions("");
    setExpiresAt(defaultExpiresAt());
  }

  function initConstraintsFromRecord(record: Record<string, unknown>) {
    const values: Record<string, string> = {};
    const modes: Record<string, ParamMode> = {};
    for (const [key, value] of Object.entries(record)) {
      if (value === "*") {
        values[key] = "*";
        modes[key] = "wildcard";
      } else if (isPatternWrapper(value)) {
        values[key] = value.$pattern;
        modes[key] = "pattern";
      } else if (value === null || value === undefined) {
        values[key] = "";
        modes[key] = "fixed";
      } else {
        values[key] = String(value);
        modes[key] = "fixed";
      }
    }
    setParamValues(values);
    setParamModes(modes);
  }

  function handleNext() {
    if (step === 1) {
      if (!agentId) {
        toast.error("Please select an agent");
        return;
      }
      setStep(2);
    } else if (step === 2) {
      if (configsLoading) {
        toast.error("Please wait for configurations to finish loading");
        return;
      }
      if (!selectedConfigId) {
        toast.error("Please select an action configuration or choose custom");
        return;
      }
      if (isCustomAction && !customActionType.trim()) {
        toast.error("Please enter an action type");
        return;
      }
      if (selectedConfig) {
        initConstraintsFromRecord(selectedConfig.parameters);
        setManualConstraintsJson("");
      } else if (initialConstraints && isCustomAction) {
        initConstraintsFromRecord(initialConstraints);
        setManualConstraintsJson(JSON.stringify(initialConstraints, null, 2));
      } else {
        setParamValues({});
        setParamModes({});
        setManualConstraintsJson("");
      }
      setStep(3);
    } else if (step === 3) {
      if (schemaLoading) {
        toast.error("Please wait for the parameter schema to finish loading");
        return;
      }
      // Use manual JSON path when no schema properties are available —
      // matches the rendering condition in StepConstraints.
      const useManualJson = !configSchema?.properties;
      if (useManualJson) {
        try {
          const parsed = JSON.parse(manualConstraintsJson) as Record<
            string,
            unknown
          >;
          if (
            parsed === null ||
            typeof parsed !== "object" ||
            Array.isArray(parsed)
          ) {
            toast.error("Constraints must be a JSON object");
            return;
          }
          const allWildcard = Object.values(parsed).every((v) => v === "*");
          if (Object.keys(parsed).length === 0 || allWildcard) {
            toast.error(
              "At least one parameter constraint must be non-wildcard",
            );
            return;
          }
        } catch {
          toast.error("Constraints must be valid JSON");
          return;
        }
      } else {
        if (!hasNonWildcardConstraint(paramValues, paramModes)) {
          toast.error(
            "At least one parameter constraint must be non-wildcard",
          );
          return;
        }
      }
      setStep(4);
    }
  }

  function handleBack() {
    if (step === 2) setStep(1);
    else if (step === 3) setStep(2);
    else if (step === 4) setStep(3);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();

    if (!agentId || !effectiveActionType || !expiresAt) {
      toast.error("Please fill in all required fields");
      return;
    }

    let constraints: Record<string, unknown>;

    // Use manual JSON path when no schema properties are available —
    // matches the rendering condition in StepConstraints.
    const useManualJson = !configSchema?.properties;
    if (useManualJson) {
      try {
        constraints = JSON.parse(manualConstraintsJson) as Record<
          string,
          unknown
        >;
      } catch {
        toast.error("Constraints must be valid JSON");
        return;
      }
      if (
        constraints === null ||
        typeof constraints !== "object" ||
        Array.isArray(constraints)
      ) {
        toast.error("Constraints must be a JSON object");
        return;
      }
      const allWildcard = Object.values(constraints).every((v) => v === "*");
      if (Object.keys(constraints).length === 0 || allWildcard) {
        toast.error("At least one parameter constraint must be non-wildcard");
        return;
      }
    } else {
      constraints = buildParametersFromForm(
        paramValues,
        configSchema?.properties,
        paramModes,
      );
    }

    try {
      await createStandingApproval({
        agent_id: agentId,
        action_type: effectiveActionType,
        action_version: "1",
        constraints,
        source_action_configuration_id: selectedConfig?.id,
        max_executions: maxExecutions ? Number(maxExecutions) : null,
        expires_at: new Date(expiresAt).toISOString(),
      });
      toast.success("Standing approval created");
      resetForm();
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create standing approval",
      );
    }
  }

  const canCreate = !isPending && !!agentId && !!effectiveActionType && !!expiresAt;

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) resetForm();
        onOpenChange(v);
      }}
    >
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create Standing Approval</DialogTitle>
          <DialogDescription>
            Step {step} of 4: {STEP_LABELS[step]}
          </DialogDescription>
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

        <form onSubmit={handleSubmit} className="space-y-4">
          {step === 1 && (
            <StepPickAgent
              agentId={agentId}
              onAgentChange={(id) => {
                setAgentId(id);
                setSelectedConfigId("");
                setCustomActionType(initialActionType ?? "");
                setParamValues({});
                setParamModes({});
              }}
              activeAgents={activeAgents}
            />
          )}

          {step === 2 && (
            <StepPickAction
              selectedConfigId={selectedConfigId}
              onConfigChange={setSelectedConfigId}
              customActionType={customActionType}
              onCustomActionTypeChange={setCustomActionType}
              configsByConnector={configsByConnector}
              configsLoading={configsLoading}
              isCustomAction={isCustomAction}
            />
          )}

          {step === 3 && (
            <StepConstraints
              configSchema={configSchema}
              schemaLoading={schemaLoading}
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
              isPending={isPending}
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
                onClick={handleBack}
                disabled={isPending}
              >
                <ChevronLeft className="size-4" />
                Back
              </Button>
            )}
            <div className="flex-1" />
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                resetForm();
                onOpenChange(false);
              }}
              disabled={isPending}
            >
              Cancel
            </Button>
            {step < 4 ? (
              <Button
                type="button"
                onClick={handleNext}
                disabled={
                  (step === 2 && configsLoading) ||
                  (step === 3 && schemaLoading)
                }
              >
                Next
                <ChevronRight className="size-4" />
              </Button>
            ) : (
              <Button type="submit" disabled={!canCreate}>
                {isPending ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Check className="size-4" />
                )}
                Create
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
