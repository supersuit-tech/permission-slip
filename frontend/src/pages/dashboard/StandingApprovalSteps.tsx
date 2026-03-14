import { Loader2, Info } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { Agent } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { ActionConfigParameterFields } from "@/pages/agents/connectors/ActionConfigParameterFields";
import type { ParamMode } from "@/pages/agents/connectors/ActionConfigFormFields";

export const CUSTOM_ACTION_SENTINEL = "__custom__";

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
      <p className="text-muted-foreground text-xs">
        Choose which agent this standing approval applies to.
      </p>
    </div>
  );
}

export function StepPickAction({
  selectedConfigId,
  onConfigChange,
  customActionType,
  onCustomActionTypeChange,
  configsByConnector,
  configsLoading,
  isCustomAction,
}: {
  selectedConfigId: string;
  onConfigChange: (id: string) => void;
  customActionType: string;
  onCustomActionTypeChange: (value: string) => void;
  configsByConnector: Record<string, ActionConfiguration[]>;
  configsLoading: boolean;
  isCustomAction: boolean;
}) {
  const connectorIds = Object.keys(configsByConnector);

  return (
    <div className="space-y-3">
      <Label htmlFor="sa-config">Action Configuration</Label>
      {configsLoading ? (
        <div className="flex items-center gap-2 py-2">
          <Loader2 className="size-4 animate-spin" />
          <span className="text-muted-foreground text-sm">
            Loading configurations...
          </span>
        </div>
      ) : (
        <>
          <select
            id="sa-config"
            value={selectedConfigId}
            onChange={(e) => onConfigChange(e.target.value)}
            className={selectClassName}
          >
            <option value="">Select an action configuration...</option>
            {connectorIds.map((connId) => (
              <optgroup key={connId} label={connId}>
                {configsByConnector[connId]?.map((config) => (
                  <option key={config.id} value={config.id}>
                    {config.name} ({config.action_type})
                  </option>
                ))}
              </optgroup>
            ))}
            <option value={CUSTOM_ACTION_SENTINEL}>
              Custom action type...
            </option>
          </select>
          <p className="text-muted-foreground text-xs">
            Select an action configuration to pre-populate constraints, or
            choose &quot;Custom action type&quot; for manual entry.
          </p>
        </>
      )}

      {isCustomAction && (
        <div className="space-y-2">
          <Label htmlFor="sa-custom-action">Action Type</Label>
          <Input
            id="sa-custom-action"
            placeholder="e.g. github.create_issue"
            value={customActionType}
            onChange={(e) => onCustomActionTypeChange(e.target.value)}
          />
        </div>
      )}
    </div>
  );
}

export function StepConstraints({
  isCustomAction,
  configSchema,
  schemaLoading,
  paramValues,
  paramModes,
  onParamValueChange,
  onParamModeChange,
  manualConstraintsJson,
  onManualConstraintsJsonChange,
  isPending,
}: {
  isCustomAction: boolean;
  configSchema: ParametersSchema | null;
  schemaLoading: boolean;
  paramValues: Record<string, string>;
  paramModes: Record<string, ParamMode>;
  onParamValueChange: (key: string, value: string) => void;
  onParamModeChange: (key: string, mode: ParamMode) => void;
  manualConstraintsJson: string;
  onManualConstraintsJsonChange: (value: string) => void;
  isPending: boolean;
}) {
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

  const showSchemaFields = configSchema?.properties && !isCustomAction;
  const showManualJson = isCustomAction && !configSchema?.properties;

  return (
    <div className="space-y-3">
      <div className="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <p className="text-muted-foreground text-xs">
          Standing approvals require parameter constraints. At least one
          parameter must be Fixed or Pattern — not all wildcards.
        </p>
      </div>

      {showSchemaFields ? (
        <div className="space-y-2">
          <Label>Constraints</Label>
          <ActionConfigParameterFields
            parametersSchema={configSchema}
            values={paramValues}
            onValueChange={onParamValueChange}
            modes={paramModes}
            onModeChange={onParamModeChange}
            disabled={isPending}
          />
        </div>
      ) : showManualJson ? (
        <div className="space-y-2">
          <Label htmlFor="sa-manual-constraints">Constraints (JSON)</Label>
          <textarea
            id="sa-manual-constraints"
            className="border-input bg-background ring-offset-background focus-visible:ring-ring flex min-h-[100px] w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={'{\n  "recipient": "*@mycompany.com",\n  "subject": "*"\n}'}
            value={manualConstraintsJson}
            onChange={(e) => onManualConstraintsJsonChange(e.target.value)}
            disabled={isPending}
          />
          <p className="text-muted-foreground text-xs">
            Enter parameter constraints as a JSON object. Use{" "}
            <code className="font-mono">&quot;*&quot;</code> for wildcard
            parameters, but at least one must be non-wildcard.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          <Label>Constraints</Label>
          {configSchema?.properties ? (
            <ActionConfigParameterFields
              parametersSchema={configSchema}
              values={paramValues}
              onValueChange={onParamValueChange}
              modes={paramModes}
              onModeChange={onParamModeChange}
              disabled={isPending}
            />
          ) : (
            <p className="text-muted-foreground text-sm">
              No parameter schema found for this action. Enter constraints
              manually as JSON below.
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export function StepLimits({
  maxExecutions,
  onMaxExecutionsChange,
  expiresAt,
  onExpiresAtChange,
}: {
  maxExecutions: string;
  onMaxExecutionsChange: (value: string) => void;
  expiresAt: string;
  onExpiresAtChange: (value: string) => void;
}) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="sa-max-executions">Max Executions</Label>
        <Input
          id="sa-max-executions"
          type="number"
          min="1"
          step="1"
          placeholder="Unlimited"
          value={maxExecutions}
          onChange={(e) => onMaxExecutionsChange(e.target.value)}
        />
        <p className="text-muted-foreground text-xs">
          Leave empty for unlimited executions.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="sa-expires-at">Expires At</Label>
        <Input
          id="sa-expires-at"
          type="datetime-local"
          value={expiresAt}
          onChange={(e) => onExpiresAtChange(e.target.value)}
          required
        />
        <p className="text-muted-foreground text-xs">
          Maximum 90 days from now.
        </p>
      </div>
    </div>
  );
}
