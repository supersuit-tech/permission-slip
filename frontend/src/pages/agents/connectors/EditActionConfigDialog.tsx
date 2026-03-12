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
import { useUpdateActionConfig } from "@/hooks/useUpdateActionConfig";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";
import {
  ActionConfigParameterFields,
  parseParametersSchema,
} from "./ActionConfigParameterFields";
import {
  WILDCARD_ACTION_TYPE,
  NameField,
  DescriptionField,
  CredentialSelect,
  StatusSelect,
  buildParametersFromForm,
  getEmptyRequiredParams,
  isPatternWrapper,
  type ParamMode,
} from "./ActionConfigFormFields";

interface EditActionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  config: ActionConfiguration;
  agentId: number;
  actions: ConnectorAction[];
  credentials: CredentialSummary[];
}

export function EditActionConfigDialog({
  open,
  onOpenChange,
  config,
  agentId,
  actions,
  credentials,
}: EditActionConfigDialogProps) {
  const { updateActionConfig, isPending } = useUpdateActionConfig();

  // No useEffect needed to sync state: the parent conditionally renders this
  // component (`{editTarget && <EditActionConfigDialog>}`), so it always
  // unmounts/remounts when switching configs. useState initializers suffice.
  const [name, setName] = useState(config.name);
  const [description, setDescription] = useState(config.description ?? "");
  const [credentialId, setCredentialId] = useState(
    config.credential_id ?? "",
  );
  const [status, setStatus] = useState<"active" | "disabled">(config.status);
  const [paramValues, setParamValues] = useState<Record<string, string>>(() =>
    toStringRecord(config.parameters),
  );
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>(() =>
    inferModesFromConfig(config.parameters),
  );

  const isWildcard = config.action_type === WILDCARD_ACTION_TYPE;
  const action = useMemo(
    () =>
      isWildcard
        ? null
        : actions.find((a) => a.action_type === config.action_type) ?? null,
    [actions, config.action_type, isWildcard],
  );

  const schema = useMemo(
    // Cast is safe: parameters_schema is typed as `{ [key: string]: unknown }` in
    // the generated OpenAPI types, which is structurally identical to Record<string, unknown>.
    () =>
      parseParametersSchema(
        action?.parameters_schema as Record<string, unknown> | undefined,
      ),
    [action],
  );

  function handleParamChange(key: string, value: string) {
    setParamValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Name is required");
      return;
    }

    const emptyRequired = getEmptyRequiredParams(paramValues, schema?.required);
    if (emptyRequired.length > 0) {
      toast.error(`Required parameters need a value or wildcard: ${emptyRequired.join(", ")}`);
      return;
    }

    try {
      await updateActionConfig({
        configId: config.id,
        agentId,
        body: {
          name: name.trim(),
          description: description.trim() || null,
          credential_id: credentialId || null,
          status,
          parameters: buildParametersFromForm(paramValues, schema?.properties, paramModes),
        },
      });
      toast.success(`Configuration "${name.trim()}" updated`);
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to update action configuration",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Action Configuration</DialogTitle>
            <DialogDescription>
              {isWildcard
                ? "Update the name, description, credential, or status for this enable-all configuration. Parameters cannot be changed on wildcard configurations."
                : <>Update the configuration for{" "}
                  <strong>{action?.name ?? config.action_type}</strong>. The action
                  type and connector cannot be changed.</>}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {/* Action (read-only) */}
            <div className="space-y-2">
              <Label>Action</Label>
              {isWildcard ? (
                <p className="text-sm font-medium">All Actions <span className="text-muted-foreground font-mono text-xs">(*)</span></p>
              ) : (
                <p className="text-sm">
                  {action?.name ?? config.action_type}{" "}
                  <span className="text-muted-foreground font-mono text-xs">
                    ({config.action_type})
                  </span>
                </p>
              )}
            </div>

            <NameField
              id="edit-config-name"
              value={name}
              onChange={setName}
              disabled={isPending}
            />

            <DescriptionField
              id="edit-config-description"
              value={description}
              onChange={setDescription}
              disabled={isPending}
            />

            <StatusSelect
              id="edit-config-status"
              value={status}
              onChange={setStatus}
              disabled={isPending}
            />

            <CredentialSelect
              id="edit-config-credential"
              value={credentialId}
              onChange={setCredentialId}
              credentials={credentials}
              disabled={isPending}
            />

            {isWildcard ? (
              <div className="space-y-2">
                <Label>Parameters</Label>
                <p className="text-muted-foreground text-sm italic">
                  All parameters — agent chooses freely. Parameters cannot be modified on wildcard configurations.
                </p>
              </div>
            ) : action ? (
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
            ) : null}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending && <Loader2 className="animate-spin" />}
              Save Changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** Convert stored parameter values to a flat string map for form inputs.
 *  Unwraps `{"$pattern": "<glob>"}` objects into plain strings. */
function toStringRecord(
  params: Record<string, unknown>,
): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value === null || value === undefined) {
      result[key] = "";
    } else if (isPatternWrapper(value)) {
      result[key] = value.$pattern;
    } else if (typeof value === "object") {
      result[key] = JSON.stringify(value);
    } else {
      result[key] = String(value);
    }
  }
  return result;
}

/** Infer ParamMode for each key from stored config parameters. */
function inferModesFromConfig(
  params: Record<string, unknown>,
): Record<string, ParamMode> {
  const modes: Record<string, ParamMode> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value === "*") {
      modes[key] = "wildcard";
    } else if (isPatternWrapper(value)) {
      modes[key] = "pattern";
    } else {
      modes[key] = "fixed";
    }
  }
  return modes;
}
