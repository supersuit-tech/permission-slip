import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { Loader2 } from "lucide-react";
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
import { Label } from "@/components/ui/label";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import { useStandingApprovalTemplates } from "@/hooks/useStandingApprovalTemplates";
import type { StandingApprovalTemplate } from "@/hooks/useStandingApprovalTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import {
  ConstraintParameterFields,
  parseParametersSchema,
} from "./ConstraintParameterFields";
import {
  ActionSelect,
  NameField,
  DescriptionField,
  buildParametersFromForm,
  getEmptyRequiredParams,
  isPatternWrapper,
  type ParamMode,
} from "./StandingApprovalFormFields";
import { TemplatePicker } from "./TemplatePicker";
import {
  RiskBadge,
  riskBlurb,
  riskCardClass,
  riskDialogAccentClass,
} from "./RiskBadge";
import { ConnectorInstanceAccountSelect } from "./ConnectorInstanceAccountSelect";
import {
  mergeConnectorInstanceIntoParameters,
} from "./connectorInstanceAccount";
import { StepLimits } from "@/pages/dashboard/StandingApprovalSteps";

interface AddStandingApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  initialTemplate?: StandingApprovalTemplate | null;
  onCreated?: () => void;
}

export function AddStandingApprovalDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  initialTemplate = null,
  onCreated,
}: AddStandingApprovalDialogProps) {
  const { createStandingApproval, isPending } = useCreateStandingApproval();
  const { templates, isLoading: templatesLoading } =
    useStandingApprovalTemplates(connectorId);
  const { instances } = useAgentConnectorInstances(agentId, connectorId);
  const showAccountSelect = instances.length > 1;

  const [selectedActionType, setSelectedActionType] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>({});
  const [connectorInstance, setConnectorInstance] = useState("*");
  const [appliedTemplateId, setAppliedTemplateId] = useState<string | null>(
    null,
  );
  const [noExpiry, setNoExpiry] = useState(true);
  const [expiresAt, setExpiresAt] = useState(() => defaultExpiresAtLocal());

  const prevOpenRef = useRef(false);

  const selectedAction = useMemo(
    () => actions.find((a) => a.action_type === selectedActionType) ?? null,
    [actions, selectedActionType],
  );

  const riskLevel = selectedAction?.risk_level;

  const schema = useMemo(
    () =>
      parseParametersSchema(
        selectedAction?.parameters_schema as
          | Record<string, unknown>
          | undefined,
      ),
    [selectedAction],
  );

  function resetForm() {
    setSelectedActionType("");
    setName("");
    setDescription("");
    setParamValues({});
    setParamModes({});
    setConnectorInstance("*");
    setAppliedTemplateId(null);
    setNoExpiry(true);
    setExpiresAt(defaultExpiresAtLocal());
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  function handleActionChange(actionType: string) {
    setSelectedActionType(actionType);
    setParamValues({});
    setParamModes({});
    setConnectorInstance("*");
    setAppliedTemplateId(null);
  }

  const handleTemplateSelect = useCallback(
    (
      template: StandingApprovalTemplate,
      options?: { fromInitial?: boolean },
    ) => {
      setSelectedActionType(template.action_type);
      setName(template.name);
      setDescription(template.description ?? "");

      const values: Record<string, string> = {};
      const modes: Record<string, ParamMode> = {};
      for (const [key, value] of Object.entries(template.constraints)) {
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
      setConnectorInstance("*");
      setAppliedTemplateId(template.id);

      if (template.duration_days != null) {
        setNoExpiry(false);
        const d = new Date();
        d.setUTCDate(d.getUTCDate() + template.duration_days);
        const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
        setExpiresAt(local.toISOString().slice(0, 16));
      }

      if (!options?.fromInitial) {
        toast.success(`Template "${template.name}" applied`);
      }
    },
    [],
  );

  useEffect(() => {
    const wasOpen = prevOpenRef.current;
    prevOpenRef.current = open;
    if (open && !wasOpen && initialTemplate) {
      handleTemplateSelect(initialTemplate, { fromInitial: true });
    }
  }, [open, initialTemplate, handleTemplateSelect]);

  function handleParamChange(key: string, value: string) {
    setParamValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!selectedActionType) {
      toast.error("Please select an action");
      return;
    }
    if (!name.trim()) {
      toast.error("Please enter a name for this standing approval");
      return;
    }

    const emptyRequired = getEmptyRequiredParams(
      paramValues,
      schema?.required,
      schema?.properties,
    );
    if (emptyRequired.length > 0) {
      toast.error(
        `Required parameters need a value or wildcard: ${emptyRequired.join(", ")}`,
      );
      return;
    }

    try {
      let builtConstraints = buildParametersFromForm(
        paramValues,
        schema?.properties,
        paramModes,
      );
      if (showAccountSelect) {
        builtConstraints = mergeConnectorInstanceIntoParameters(
          builtConstraints,
          connectorInstance,
        );
      }

      const activeTemplate =
        appliedTemplateId != null
          ? (templates.find((t) => t.id === appliedTemplateId) ?? null)
          : null;
      const constraints = standingApprovalConstraintsForCreate(
        builtConstraints as Record<string, unknown>,
        activeTemplate ?? initialTemplate,
      );

      await createStandingApproval({
        agent_id: agentId,
        action_type: selectedActionType,
        action_version: "1",
        name: name.trim(),
        description: description.trim() || null,
        constraints,
        ...(noExpiry ? {} : { expires_at: new Date(expiresAt).toISOString() }),
      });

      toast.success(`Standing approval "${name.trim()}" created`);
      resetForm();
      onOpenChange(false);
      onCreated?.();
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create standing approval",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className={`max-h-[85dvh] overflow-y-auto sm:max-w-lg ${riskDialogAccentClass(riskLevel)}`}
      >
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Standing Approval</DialogTitle>
            <DialogDescription>
              Pre-authorize matching requests so this agent can run them
              automatically. Set constraint boundaries for each parameter.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <ActionSelect
              id="standing-approval-action"
              value={selectedActionType}
              onChange={handleActionChange}
              actions={actions}
              disabled={isPending}
            />

            {selectedAction && (
              <div
                className={`flex items-start gap-3 rounded-lg border p-3 ${riskCardClass(riskLevel)}`}
              >
                <RiskBadge level={riskLevel} />
                <p className="text-muted-foreground text-sm">
                  {riskBlurb(riskLevel)}
                </p>
              </div>
            )}

            {selectedActionType && (
              <TemplatePicker
                templates={templates}
                isLoading={templatesLoading}
                actionType={selectedActionType}
                onSelect={handleTemplateSelect}
                disabled={isPending}
                selectedTemplateId={appliedTemplateId}
              />
            )}

            <NameField
              id="standing-approval-name"
              value={name}
              onChange={setName}
              disabled={isPending}
            />

            <DescriptionField
              id="standing-approval-description"
              value={description}
              onChange={setDescription}
              disabled={isPending}
            />

            {showAccountSelect && (
              <ConnectorInstanceAccountSelect
                id="standing-approval-account"
                value={connectorInstance}
                onChange={setConnectorInstance}
                instances={instances}
                disabled={isPending}
              />
            )}

            {selectedAction && (
              <div className="space-y-2">
                <Label>Constraints</Label>
                <ConstraintParameterFields
                  parametersSchema={schema}
                  values={paramValues}
                  onValueChange={handleParamChange}
                  modes={paramModes}
                  onModeChange={(key, mode) =>
                    setParamModes((prev) => ({ ...prev, [key]: mode }))
                  }
                  disabled={isPending}
                  agentId={agentId}
                  connectorId={connectorId}
                />
              </div>
            )}

            <StepLimits
              expiresAt={expiresAt}
              onExpiresAtChange={setExpiresAt}
              noExpiry={noExpiry}
              onNoExpiryChange={setNoExpiry}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={() => handleOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !selectedActionType}>
              {isPending && <Loader2 className="animate-spin" />}
              Create Standing Approval
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function defaultExpiresAtLocal(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
  template?: StandingApprovalTemplate | null,
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0 || template) {
    return params;
  }
  const allBareWildcard = entries.every(([, v]) => v === "*");
  if (allBareWildcard) {
    return {};
  }
  return params;
}
