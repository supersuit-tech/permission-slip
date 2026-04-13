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
import { SegmentedControl } from "@/components/ui/segmented-control";
import { useCreateActionConfig } from "@/hooks/useCreateActionConfig";
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import {
  ActionConfigParameterFields,
  parseParametersSchema,
} from "./ActionConfigParameterFields";
import {
  ActionSelect,
  NameField,
  DescriptionField,
  buildParametersFromForm,
  getEmptyRequiredParams,
  isPatternWrapper,
  type ParamMode,
} from "./ActionConfigFormFields";
import { TemplatePicker } from "./TemplatePicker";
import type { ApprovalMode } from "./RecommendedTemplatesDialog";

const approvalModeOptions: { label: string; value: ApprovalMode }[] = [
  { label: "Auto-approve", value: "auto_approve" },
  { label: "Requires approval", value: "requires_approval" },
];

interface AddActionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  /** When the dialog opens, apply this template (action type, fields, parameters). */
  initialTemplate?: ActionConfigTemplate | null;
  /** Override the template's default approval mode. */
  initialApprovalMode?: ApprovalMode;
}

export function AddActionConfigDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  initialTemplate = null,
  initialApprovalMode,
}: AddActionConfigDialogProps) {
  const { createActionConfig, isPending: isCreatingConfig } =
    useCreateActionConfig();
  const { createStandingApproval, isPending: isCreatingStanding } =
    useCreateStandingApproval();
  const isPending = isCreatingConfig || isCreatingStanding;
  const { templates, isLoading: templatesLoading } =
    useActionConfigTemplates(connectorId);

  const [selectedActionType, setSelectedActionType] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>({});
  const [appliedTemplateId, setAppliedTemplateId] = useState<string | null>(null);
  const [approvalMode, setApprovalMode] = useState<ApprovalMode>("requires_approval");

  const prevOpenRef = useRef(false);

  const selectedAction = useMemo(
    () => actions.find((a) => a.action_type === selectedActionType) ?? null,
    [actions, selectedActionType],
  );

  const schema = useMemo(
    // Cast is safe: parameters_schema is typed as `{ [key: string]: unknown }` in
    // the generated OpenAPI types, which is structurally identical to Record<string, unknown>.
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
    setAppliedTemplateId(null);
    setApprovalMode("requires_approval");
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  function handleActionChange(actionType: string) {
    setSelectedActionType(actionType);
    setParamValues({});
    setParamModes({});
    setAppliedTemplateId(null);
  }

  const handleTemplateSelect = useCallback(
    (
      template: ActionConfigTemplate,
      options?: { fromInitial?: boolean },
    ) => {
      setSelectedActionType(template.action_type);
      setName(template.name);
      setDescription(template.description ?? "");

      const values: Record<string, string> = {};
      const modes: Record<string, ParamMode> = {};
      for (const [key, value] of Object.entries(template.parameters)) {
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
      setAppliedTemplateId(template.id);

      // When selecting a new template via the picker (not from initial),
      // reset approval mode to the template's default.
      if (!options?.fromInitial) {
        setApprovalMode(
          template.standing_approval != null ? "auto_approve" : "requires_approval",
        );
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
      setApprovalMode(
        initialApprovalMode ??
          (initialTemplate.standing_approval != null ? "auto_approve" : "requires_approval"),
      );
    }
  }, [open, initialTemplate, initialApprovalMode, handleTemplateSelect]);

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
      toast.error("Please enter a name for this configuration");
      return;
    }

    const emptyRequired = getEmptyRequiredParams(paramValues, schema?.required, schema?.properties);
    if (emptyRequired.length > 0) {
      toast.error(`Required parameters need a value or wildcard: ${emptyRequired.join(", ")}`);
      return;
    }

    try {
      const builtParams = buildParametersFromForm(
        paramValues,
        schema?.properties,
        paramModes,
      );
      const ac = await createActionConfig({
        agent_id: agentId,
        connector_id: connectorId,
        action_type: selectedActionType,
        name: name.trim(),
        description: description.trim() || undefined,
        parameters: builtParams,
      });

      if (approvalMode === "auto_approve" && ac?.id) {
        const activeTemplate =
          appliedTemplateId != null
            ? templates.find((t) => t.id === appliedTemplateId) ?? null
            : null;
        const standingSpec =
          activeTemplate?.standing_approval ?? initialTemplate?.standing_approval;
        const startsAt = new Date();
        let expiresAt: string | null = null;
        if (standingSpec?.duration_days != null) {
          const exp = new Date(startsAt);
          exp.setUTCDate(exp.getUTCDate() + standingSpec.duration_days);
          expiresAt = exp.toISOString();
        }
        const constraints = standingApprovalConstraintsForCreate(
          builtParams as Record<string, unknown>,
        );
        await createStandingApproval({
          agent_id: agentId,
          action_type: selectedActionType,
          action_version: "1",
          constraints,
          source_action_configuration_id: ac.id,
          max_executions: standingSpec?.max_executions ?? null,
          starts_at: startsAt.toISOString(),
          expires_at: expiresAt,
        });
      }

      toast.success(
        approvalMode === "auto_approve"
          ? `Configuration "${name.trim()}" created with auto-approval`
          : `Configuration "${name.trim()}" created`,
      );
      resetForm();
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to create action configuration",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Action Configuration</DialogTitle>
            <DialogDescription>
              Define how this agent can use an action. Lock in parameter values
              or mark them as wildcards to give the agent freedom.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <ActionSelect
              id="config-action"
              value={selectedActionType}
              onChange={handleActionChange}
              actions={actions}
              disabled={isPending}
            />

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

            <div className="space-y-2">
              <Label>Approval mode</Label>
              <SegmentedControl
                options={approvalModeOptions}
                value={approvalMode}
                onChange={setApprovalMode}
                ariaLabel="Approval mode"
              />
              <p className="text-muted-foreground text-xs">
                {approvalMode === "auto_approve"
                  ? "A standing approval will be created so matching requests run automatically."
                  : "Each request will require your explicit approval before it runs."}
              </p>
            </div>

            <NameField
              id="config-name"
              value={name}
              onChange={setName}
              disabled={isPending}
            />

            <DescriptionField
              id="config-description"
              value={description}
              onChange={setDescription}
              disabled={isPending}
            />

            {selectedAction && (
              <div className="space-y-2">
                <Label>Parameters</Label>
                <ActionConfigParameterFields
                  parametersSchema={schema}
                  values={paramValues}
                  onValueChange={handleParamChange}
                  modes={paramModes}
                  onModeChange={(key, mode) => setParamModes((prev) => ({ ...prev, [key]: mode }))}
                  disabled={isPending}
                  agentId={agentId}
                  connectorId={connectorId}
                />
              </div>
            )}
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
              Create Configuration
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** With source_action_configuration_id, `{}` means match-all (backend stores NULL constraints). */
function standingApprovalConstraintsForCreate(
  params: Record<string, unknown>,
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return {};
  }
  const allBareWildcard = entries.every(([, v]) => v === "*");
  if (allBareWildcard) {
    return {};
  }
  return params;
}
