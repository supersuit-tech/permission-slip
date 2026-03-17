import { type FormEvent, useState, useMemo } from "react";
import { Loader2, ChevronLeft, ChevronRight, Check } from "lucide-react";
import { toast } from "sonner";
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
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useUpdateStandingApproval } from "@/hooks/useUpdateStandingApproval";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
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
  /** When provided, the dialog operates in edit mode for the given standing approval. */
  editTarget?: StandingApproval;
  /** Called after a standing approval is successfully created. */
  onCreated?: () => void;
  /** Called after a standing approval is successfully updated. */
  onUpdated?: () => void;
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
  editTarget,
  onCreated,
  onUpdated,
}: CreateStandingApprovalDialogProps) {
  const { createStandingApproval, isPending: isCreatePending } = useCreateStandingApproval();
  const { updateStandingApproval, isPending: isUpdatePending } = useUpdateStandingApproval();
  const isPending = isCreatePending || isUpdatePending;
  const isEditMode = !!editTarget;

  // In edit mode, derive initial values from the target approval.
  const ctxAgentId = isEditMode ? editTarget.agent_id : initialAgentId;
  const ctxActionType = isEditMode ? editTarget.action_type : initialActionType;
  const ctxConstraints = isEditMode
    ? (editTarget.constraints as Record<string, unknown>)
    : initialConstraints;

  // Skip straight to constraints when agent + action are pre-filled
  const hasInitialContext = !!(ctxAgentId && ctxActionType);
  const [step, setStep] = useState<Step>(hasInitialContext ? 3 : 1);
  const [agentId, setAgentId] = useState<number | "">(ctxAgentId ?? "");
  const [selectedConfigId, setSelectedConfigId] = useState<string>(
    isEditMode
      ? (editTarget.source_action_configuration_id ?? CUSTOM_ACTION_SENTINEL)
      : (ctxActionType ? CUSTOM_ACTION_SENTINEL : ""),
  );
  const [customActionType, setCustomActionType] = useState(
    ctxActionType ?? "",
  );
  // Pre-populate constraint form values when initial constraints are provided
  const [paramValues, setParamValues] = useState<Record<string, string>>(() => {
    if (!hasInitialContext || !ctxConstraints) return {};
    const values: Record<string, string> = {};
    for (const [key, value] of Object.entries(ctxConstraints)) {
      if (value === "*") values[key] = "*";
      else if (isPatternWrapper(value)) values[key] = value.$pattern;
      else if (value === null || value === undefined) values[key] = "";
      else values[key] = String(value);
    }
    return values;
  });
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>(() => {
    if (!hasInitialContext || !ctxConstraints) return {};
    const modes: Record<string, ParamMode> = {};
    for (const [key, value] of Object.entries(ctxConstraints)) {
      if (value === "*") modes[key] = "wildcard";
      else if (isPatternWrapper(value)) modes[key] = "pattern";
      else modes[key] = "fixed";
    }
    return modes;
  });
  const [manualConstraintsJson, setManualConstraintsJson] = useState(
    hasInitialContext && ctxConstraints
      ? JSON.stringify(ctxConstraints, null, 2)
      : "",
  );
  const [maxExecutions, setMaxExecutions] = useState(
    isEditMode && editTarget.max_executions != null
      ? String(editTarget.max_executions)
      : "",
  );
  const [expiresAt, setExpiresAt] = useState(() => {
    if (isEditMode && editTarget.expires_at) {
      const d = new Date(editTarget.expires_at);
      const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
      return local.toISOString().slice(0, 16);
    }
    return defaultExpiresAt();
  });

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

  const {
    schema: fetchedSchema,
    isLoading: schemaLoading,
    connectorName,
    actionName,
    connectorLogoSvg,
  } = useActionSchema(effectiveActionType);

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
    setStep(hasInitialContext ? 3 : 1);
    setAgentId(ctxAgentId ?? "");
    setSelectedConfigId(
      isEditMode
        ? (editTarget.source_action_configuration_id ?? CUSTOM_ACTION_SENTINEL)
        : (ctxActionType ? CUSTOM_ACTION_SENTINEL : ""),
    );
    setCustomActionType(ctxActionType ?? "");
    if (hasInitialContext && ctxConstraints) {
      const values: Record<string, string> = {};
      const modes: Record<string, ParamMode> = {};
      for (const [key, value] of Object.entries(ctxConstraints)) {
        if (value === "*") { values[key] = "*"; modes[key] = "wildcard"; }
        else if (isPatternWrapper(value)) { values[key] = value.$pattern; modes[key] = "pattern"; }
        else { values[key] = value === null || value === undefined ? "" : String(value); modes[key] = "fixed"; }
      }
      setParamValues(values);
      setParamModes(modes);
    } else {
      setParamValues({});
      setParamModes({});
    }
    setManualConstraintsJson(
      hasInitialContext && ctxConstraints
        ? JSON.stringify(ctxConstraints, null, 2)
        : "",
    );
    if (isEditMode && editTarget.max_executions != null) {
      setMaxExecutions(String(editTarget.max_executions));
    } else {
      setMaxExecutions("");
    }
    if (isEditMode && editTarget.expires_at) {
      const d = new Date(editTarget.expires_at);
      const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
      setExpiresAt(local.toISOString().slice(0, 16));
    } else {
      setExpiresAt(defaultExpiresAt());
    }
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
      } else if (ctxConstraints && isCustomAction) {
        initConstraintsFromRecord(ctxConstraints);
        setManualConstraintsJson(JSON.stringify(ctxConstraints, null, 2));
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
    const minStep = hasInitialContext ? 3 : 1;
    if (step === 2 && minStep <= 1) setStep(1);
    else if (step === 3 && minStep <= 2) setStep(2);
    else if (step === 4) setStep(3);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (step !== 4) return;

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
      if (isEditMode) {
        await updateStandingApproval(editTarget.standing_approval_id, {
          constraints,
          max_executions: maxExecutions ? Number(maxExecutions) : null,
          expires_at: new Date(expiresAt).toISOString(),
        });
        toast.success("Standing approval updated");
        resetForm();
        onOpenChange(false);
        onUpdated?.();
      } else {
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
        onCreated?.();
      }
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : isEditMode
            ? "Failed to update standing approval"
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
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          {effectiveActionType ? (
            <>
              <div className="flex items-center gap-3">
                <ConnectorLogo
                  name={connectorName ?? effectiveActionType}
                  logoSvg={connectorLogoSvg}
                  size="lg"
                />
                <div className="min-w-0">
                  <DialogTitle className="truncate text-base">
                    {actionName ?? effectiveActionType}
                  </DialogTitle>
                  {connectorName && (
                    <p className="text-muted-foreground text-sm">
                      {connectorName}
                    </p>
                  )}
                </div>
              </div>
              <DialogDescription>
                {hasInitialContext
                  ? `Step ${step - 2} of 2: ${STEP_LABELS[step]}`
                  : `Step ${step} of 4: ${STEP_LABELS[step]}`}
              </DialogDescription>
            </>
          ) : (
            <>
              <DialogTitle>
                {isEditMode ? "Edit Standing Approval" : "Create Standing Approval"}
              </DialogTitle>
              <DialogDescription>
                {hasInitialContext
                  ? `Step ${step - 2} of 2: ${STEP_LABELS[step]}`
                  : `Step ${step} of 4: ${STEP_LABELS[step]}`}
              </DialogDescription>
            </>
          )}
        </DialogHeader>

        <div className="flex items-center gap-1 px-1">
          {(hasInitialContext ? [3, 4] as Step[] : [1, 2, 3, 4] as Step[]).map((s) => (
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
                setCustomActionType(ctxActionType ?? "");
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
            {step > (hasInitialContext ? 3 : 1) && (
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
                {isEditMode ? "Save" : "Create"}
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
