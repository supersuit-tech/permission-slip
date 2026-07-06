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
import { useUpdateStandingApproval } from "@/hooks/useUpdateStandingApproval";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import {
  ConstraintParameterFields,
  parseParametersSchema,
} from "./ConstraintParameterFields";
import { MetaConstraintFields } from "./MetaConstraintFields";
import {
  NameField,
  DescriptionField,
  buildParametersFromForm,
  getEmptyRequiredParams,
  isPatternWrapper,
  type ParamMode,
} from "./StandingApprovalFormFields";
import {
  buildMetaConstraintsFromForm,
  mergeStandingApprovalConstraints,
  metaModesFromConstraints,
  metaValuesFromConstraints,
  preservedNamespacesFromConstraints,
} from "@/lib/standingApprovalConstraints";
import { ConnectorInstanceAccountSelect } from "./ConnectorInstanceAccountSelect";
import {
  connectorInstanceFromStandingApprovalId,
  standingApprovalConnectorInstanceIdForUpdate,
} from "./connectorInstanceAccount";
import { StepLimits } from "@/pages/dashboard/StandingApprovalSteps";

interface EditStandingApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: StandingApproval;
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  onUpdated?: () => void;
}

export function EditStandingApprovalDialog({
  open,
  onOpenChange,
  rule,
  agentId,
  connectorId,
  actions,
  onUpdated,
}: EditStandingApprovalDialogProps) {
  const { updateStandingApproval, isPending } = useUpdateStandingApproval();
  const { instances } = useAgentConnectorInstances(agentId, connectorId);
  const showAccountSelect = instances.length >= 1;

  const [name, setName] = useState(rule.name ?? "");
  const [description, setDescription] = useState(rule.description ?? "");
  const [paramValues, setParamValues] = useState<Record<string, string>>(() =>
    toStringRecord((rule.constraints ?? {}) as Record<string, unknown>),
  );
  const [paramModes, setParamModes] = useState<Record<string, ParamMode>>(() =>
    inferModesFromConstraints((rule.constraints ?? {}) as Record<string, unknown>),
  );
  const [metaValues, setMetaValues] = useState<Record<string, string>>(() =>
    metaValuesFromConstraints((rule.constraints ?? {}) as Record<string, unknown>),
  );
  const [metaModes, setMetaModes] = useState<Record<string, ParamMode>>(() =>
    metaModesFromConstraints((rule.constraints ?? {}) as Record<string, unknown>),
  );
  const preservedNamespaces = useMemo(
    () =>
      preservedNamespacesFromConstraints(
        (rule.constraints ?? {}) as Record<string, unknown>,
      ),
    [rule.constraints],
  );
  const [connectorInstance, setConnectorInstance] = useState(() =>
    connectorInstanceFromStandingApprovalId(rule.connector_instance_id),
  );
  const [noExpiry, setNoExpiry] = useState(!rule.expires_at);
  const [expiresAt, setExpiresAt] = useState(() => {
    if (!rule.expires_at) return defaultExpiresAtLocal();
    const d = new Date(rule.expires_at);
    const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
  });

  const action = useMemo(
    () => actions.find((a) => a.action_type === rule.action_type) ?? null,
    [actions, rule.action_type],
  );

  const schema = useMemo(
    () =>
      parseParametersSchema(
        action?.parameters_schema as Record<string, unknown> | undefined,
      ),
    [action],
  );

  const metaFields = action?.meta_constraint_fields ?? [];

  function handleParamChange(key: string, value: string) {
    setParamValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Name is required");
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
      const constraints = mergeStandingApprovalConstraints(
        buildParametersFromForm(paramValues, schema?.properties, paramModes),
        buildMetaConstraintsFromForm(metaValues, metaModes),
        preservedNamespaces,
      );

      await updateStandingApproval(rule.standing_approval_id, {
        name: name.trim(),
        description: description.trim() || null,
        constraints,
        expires_at: noExpiry ? null : new Date(expiresAt).toISOString(),
        connector_instance_id:
          standingApprovalConnectorInstanceIdForUpdate(connectorInstance),
      });
      toast.success(`Standing approval "${name.trim()}" updated`);
      onOpenChange(false);
      onUpdated?.();
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to update standing approval",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Standing Approval</DialogTitle>
            <DialogDescription>
              Update the standing approval for{" "}
              <strong>{action?.name ?? rule.action_type}</strong>. The action
              type cannot be changed.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Action</Label>
              <p className="text-sm">
                {action?.name ?? rule.action_type}{" "}
                <span className="text-muted-foreground font-mono text-xs">
                  ({rule.action_type})
                </span>
              </p>
            </div>

            <NameField
              id="edit-standing-approval-name"
              value={name}
              onChange={setName}
              disabled={isPending}
            />

            <DescriptionField
              id="edit-standing-approval-description"
              value={description}
              onChange={setDescription}
              disabled={isPending}
            />

            {showAccountSelect && (
              <ConnectorInstanceAccountSelect
                id="edit-standing-approval-account"
                value={connectorInstance}
                onChange={setConnectorInstance}
                instances={instances}
                disabled={isPending}
              />
            )}

            {action ? (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Parameter constraints</Label>
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
                {metaFields.length > 0 && (
                  <div className="space-y-2">
                    <Label>Verified metadata</Label>
                    <MetaConstraintFields
                      fields={metaFields}
                      values={metaValues}
                      onValueChange={(key, value) =>
                        setMetaValues((prev) => ({ ...prev, [key]: value }))
                      }
                      modes={metaModes}
                      onModeChange={(key, mode) =>
                        setMetaModes((prev) => ({ ...prev, [key]: mode }))
                      }
                      disabled={isPending}
                    />
                  </div>
                )}
              </div>
            ) : null}

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

function defaultExpiresAtLocal(): string {
  const d = new Date();
  d.setDate(d.getDate() + 30);
  const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 16);
}

function toStringRecord(
  constraints: Record<string, unknown> | null | undefined,
): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(constraints ?? {})) {
    if (key === "$meta" || key === "$data_window") continue;
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

function inferModesFromConstraints(
  constraints: Record<string, unknown> | null | undefined,
): Record<string, ParamMode> {
  const modes: Record<string, ParamMode> = {};
  for (const [key, value] of Object.entries(constraints ?? {})) {
    if (key === "$meta" || key === "$data_window") continue;
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
