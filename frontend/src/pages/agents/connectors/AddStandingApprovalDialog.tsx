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
import { useCreateStandingApproval } from "@/hooks/useCreateStandingApproval";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import {
  ConstraintScenariosEditor,
  ensureScenarioFieldRows,
} from "./ConstraintScenariosEditor";
import { parseParametersSchema } from "@/lib/parameterSchema";
import {
  ActionSelect,
  NameField,
  DescriptionField,
} from "./StandingApprovalFormFields";
import {
  buildStructuredConstraintsFromForm,
  constraintsToFormState,
  type StructuredConstraintFormState,
} from "@/lib/structuredConstraints";
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
  onCreated?: () => void;
}

export function AddStandingApprovalDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  actions,
  onCreated,
}: AddStandingApprovalDialogProps) {
  const { createStandingApproval, isPending } = useCreateStandingApproval();
  const { instances } = useAgentConnectorInstances(agentId, connectorId);
  const showAccountSelect = instances.length > 1;

  const [selectedActionType, setSelectedActionType] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [constraintForm, setConstraintForm] =
    useState<StructuredConstraintFormState>(() =>
      constraintsToFormState(null),
    );
  const [connectorInstance, setConnectorInstance] = useState("*");
  const [noExpiry, setNoExpiry] = useState(true);
  const [expiresAt, setExpiresAt] = useState(() => defaultExpiresAtLocal());

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

  const metaFields = selectedAction?.meta_constraint_fields ?? [];

  const paramKeys = useMemo(
    () => (schema?.properties ? Object.keys(schema.properties) : []),
    [schema],
  );

  const preparedConstraintForm = useMemo(
    () => ensureScenarioFieldRows(constraintForm, paramKeys, metaFields),
    [constraintForm, paramKeys, metaFields],
  );

  function resetForm() {
    setSelectedActionType("");
    setName("");
    setDescription("");
    setConstraintForm(constraintsToFormState(null));
    setConnectorInstance("*");
    setNoExpiry(true);
    setExpiresAt(defaultExpiresAtLocal());
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  function handleActionChange(actionType: string) {
    setSelectedActionType(actionType);
    setConstraintForm(constraintsToFormState(null));
    setConnectorInstance("*");
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

    try {
      let builtConstraints = buildStructuredConstraintsFromForm(
        preparedConstraintForm,
      ) as Record<string, unknown>;
      if (showAccountSelect) {
        builtConstraints = mergeConnectorInstanceIntoParameters(
          builtConstraints,
          connectorInstance,
        );
      }

      const constraints = standingApprovalConstraintsForCreate(
        builtConstraints as Record<string, unknown>,
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
                <ConstraintScenariosEditor
                  form={preparedConstraintForm}
                  onChange={setConstraintForm}
                  parametersSchema={schema}
                  metaFields={metaFields}
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
): Record<string, unknown> {
  const entries = Object.entries(params);
  if (entries.length === 0) {
    return params;
  }
  const allBareWildcard = entries.every(([, v]) => v === "*");
  if (allBareWildcard) {
    return {};
  }
  return params;
}
