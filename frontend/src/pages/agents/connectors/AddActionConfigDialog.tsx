import { useState, useMemo } from "react";
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
import { useCreateActionConfig } from "@/hooks/useCreateActionConfig";
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";
import {
  ActionConfigParameterFields,
  parseParametersSchema,
} from "./ActionConfigParameterFields";
import {
  ActionSelect,
  NameField,
  DescriptionField,
  CredentialSelect,
  buildParametersFromForm,
  getEmptyRequiredParams,
  isPatternWrapper,
  type ParamMode,
} from "./ActionConfigFormFields";
import { TemplatePicker } from "./TemplatePicker";

interface AddActionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  credentials: CredentialSummary[];
}

export function AddActionConfigDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  credentials,
}: AddActionConfigDialogProps) {
  const { createActionConfig, isPending } = useCreateActionConfig();
  const { templates, isLoading: templatesLoading } =
    useActionConfigTemplates(connectorId);

  const [selectedActionType, setSelectedActionType] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [credentialId, setCredentialId] = useState("");
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>({});
  const [appliedTemplateId, setAppliedTemplateId] = useState<string | null>(null);

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
    setCredentialId("");
    setParamValues({});
    setParamModes({});
    setAppliedTemplateId(null);
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

  function handleTemplateSelect(template: ActionConfigTemplate) {
    // Pre-fill form fields from the template
    setName(template.name);
    setDescription(template.description ?? "");

    // Convert template parameters to form values and modes
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

    toast.success(`Template "${template.name}" applied`);
  }

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

    const emptyRequired = getEmptyRequiredParams(paramValues, schema?.required);
    if (emptyRequired.length > 0) {
      toast.error(`Required parameters need a value or wildcard: ${emptyRequired.join(", ")}`);
      return;
    }

    try {
      await createActionConfig({
        agent_id: agentId,
        connector_id: connectorId,
        action_type: selectedActionType,
        name: name.trim(),
        description: description.trim() || undefined,
        credential_id: credentialId || undefined,
        parameters: buildParametersFromForm(paramValues, schema?.properties, paramModes),
      });
      toast.success(`Configuration "${name.trim()}" created`);
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
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
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

            <CredentialSelect
              id="config-credential"
              value={credentialId}
              onChange={setCredentialId}
              credentials={credentials}
              disabled={isPending}
              helpText="A credential is required before the agent can execute actions through this configuration."
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
