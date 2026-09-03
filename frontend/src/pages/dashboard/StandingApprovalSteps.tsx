import { useMemo } from "react";
import { Loader2, Info } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Agent } from "@/hooks/useAgents";
import type { AgentActionOption } from "@/hooks/useAgentConnectorActions";
import { getAgentDisplayName } from "@/lib/agents";
import type { ParametersSchema } from "@/lib/parameterSchema";
import {
  ConstraintScenariosEditor,
  ensureScenarioFieldRows,
} from "@/pages/agents/connectors/ConstraintScenariosEditor";
import { DataWindowPicker } from "@/components/DataWindowPicker";
import type { DataWindowFormState } from "@/lib/dataWindow";
import type { StructuredConstraintFormState } from "@/lib/structuredConstraints";

const selectClassName =
  "border-input bg-background ring-offset-background focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50";

export function StepPickAgent({
  agentId,
  onAgentChange,
  activeAgents,
}: {
  agentId: number | "";
  onAgentChange: (id: number | "") => void;
  activeAgents: Agent[];
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="sa-agent">Agent</Label>
      <select
        id="sa-agent"
        value={agentId}
        onChange={(e) =>
          onAgentChange(e.target.value === "" ? "" : Number(e.target.value))
        }
        className={selectClassName}
      >
        <option value="">Select an agent</option>
        {activeAgents.map((a) => (
          <option key={a.agent_id} value={a.agent_id}>
            {getAgentDisplayName(a)}
          </option>
        ))}
      </select>
      <p className="text-muted-foreground text-sm">
        Choose which agent this standing approval applies to.
      </p>
    </div>
  );
}

export function StepPickAction({
  selectedActionType,
  onActionChange,
  actionsByConnector,
  actionsLoading,
}: {
  selectedActionType: string;
  onActionChange: (actionType: string) => void;
  actionsByConnector: Record<string, AgentActionOption[]>;
  actionsLoading: boolean;
}) {
  const connectorIds = Object.keys(actionsByConnector);

  return (
    <div className="space-y-3">
      <Label htmlFor="sa-action">Action</Label>
      {actionsLoading ? (
        <div className="flex items-center gap-2 py-2">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-muted-foreground text-sm">
            Loading actions...
          </span>
        </div>
      ) : connectorIds.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
          <p className="text-muted-foreground text-xs leading-relaxed">
            No enabled connectors found for this agent. Enable a connector
            before creating a standing approval.
          </p>
        </div>
      ) : (
        <>
          <select
            id="sa-action"
            value={selectedActionType}
            onChange={(e) => onActionChange(e.target.value)}
            className={selectClassName}
          >
            <option value="">Select an action...</option>
            {connectorIds.map((connId) => (
              <optgroup key={connId} label={connId}>
                {actionsByConnector[connId]?.map((action) => (
                  <option key={action.action_type} value={action.action_type}>
                    {action.name} ({action.action_type})
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              Choose the action type this standing approval will pre-authorize.
            </p>
          </div>
        </>
      )}
    </div>
  );
}

export function StepConstraints({
  configSchema,
  schemaLoading,
  constraintForm,
  onConstraintFormChange,
  metaFields,
  manualConstraintsJson,
  onManualConstraintsJsonChange,
  dataWindowSupported,
  dataWindowForm,
  onDataWindowFormChange,
  isPending,
}: {
  configSchema: ParametersSchema | null;
  schemaLoading: boolean;
  constraintForm: StructuredConstraintFormState;
  onConstraintFormChange: (form: StructuredConstraintFormState) => void;
  metaFields: string[];
  manualConstraintsJson: string;
  onManualConstraintsJsonChange: (value: string) => void;
  dataWindowSupported?: boolean;
  dataWindowForm: DataWindowFormState;
  onDataWindowFormChange: (value: DataWindowFormState) => void;
  isPending: boolean;
}) {
  const paramKeys = useMemo(
    () => (configSchema?.properties ? Object.keys(configSchema.properties) : []),
    [configSchema],
  );

  const preparedForm = useMemo(
    () => ensureScenarioFieldRows(constraintForm, paramKeys, metaFields),
    [constraintForm, paramKeys, metaFields],
  );

  if (schemaLoading) {
    return (
      <div className="flex items-center gap-2 py-4">
        <Loader2 className="size-4 animate-spin" />
        <span className="text-muted-foreground text-sm">
          Loading parameter schema...
        </span>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <p className="text-muted-foreground text-sm">
          Standing approvals can constrain parameters, or you can confirm an
          unrestricted rule that auto-approves any parameters for this action.
        </p>
      </div>

      {configSchema?.properties ? (
        <div className="space-y-2">
          <Label>Constraints</Label>
          <ConstraintScenariosEditor
            form={preparedForm}
            onChange={onConstraintFormChange}
            parametersSchema={configSchema}
            metaFields={metaFields}
            disabled={isPending}
          />
        </div>
      ) : (
        <div className="space-y-2">
          <Label htmlFor="sa-manual-constraints">Constraints (JSON)</Label>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              No parameter schema found for this action. Enter constraints
              manually as a JSON object (legacy flat map or v2 structured
              format with <code className="font-mono">$version: 2</code>).
            </p>
          </div>
          <textarea
            id="sa-manual-constraints"
            className="border-input bg-background ring-offset-background focus-visible:ring-ring flex min-h-[100px] w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={'{\n  "$version": 2,\n  "match": "any",\n  "groups": [...]\n}'}
            value={manualConstraintsJson}
            onChange={(e) => onManualConstraintsJsonChange(e.target.value)}
            disabled={isPending}
          />
        </div>
      )}

      {dataWindowSupported ? (
        <DataWindowPicker
          value={dataWindowForm}
          onChange={onDataWindowFormChange}
          disabled={isPending}
        />
      ) : null}
    </div>
  );
}

export function StepLimits({
  expiresAt,
  onExpiresAtChange,
  noExpiry,
  onNoExpiryChange,
}: {
  expiresAt: string;
  onExpiresAtChange: (value: string) => void;
  noExpiry?: boolean;
  onNoExpiryChange?: (value: boolean) => void;
}) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label htmlFor="sa-expires-at">Expires At</Label>
          {onNoExpiryChange && (
            <div className="flex items-center gap-2">
              <Checkbox
                id="sa-no-expiry"
                checked={noExpiry ?? false}
                onCheckedChange={(checked) => onNoExpiryChange(checked === true)}
              />
              <Label htmlFor="sa-no-expiry" className="text-sm font-normal">
                Until revoked
              </Label>
            </div>
          )}
        </div>
        {!noExpiry && (
          <Input
            id="sa-expires-at"
            type="datetime-local"
            value={expiresAt}
            onChange={(e) => onExpiresAtChange(e.target.value)}
            required
          />
        )}
        <p className="text-muted-foreground text-sm">
          {noExpiry
            ? "This standing approval will remain active until you revoke it."
            : 'Set a specific expiration date, or check "Until revoked" for no expiry.'}
        </p>
      </div>
    </div>
  );
}
