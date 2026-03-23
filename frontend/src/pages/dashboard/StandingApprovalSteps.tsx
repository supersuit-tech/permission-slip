import { Loader2, Info } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { Agent } from "@/hooks/useAgents";
import { getAgentDisplayName } from "@/lib/agents";
import type { ParametersSchema } from "@/lib/parameterSchema";
import { ActionConfigParameterFields } from "@/pages/agents/connectors/ActionConfigParameterFields";
import type { ParamMode } from "@/pages/agents/connectors/ActionConfigFormFields";

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
  selectedConfigId,
  onConfigChange,
  configsByConnector,
  configsLoading,
}: {
  selectedConfigId: string;
  onConfigChange: (id: string) => void;
  configsByConnector: Record<string, ActionConfiguration[]>;
  configsLoading: boolean;
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
      ) : connectorIds.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
          <p className="text-muted-foreground text-xs leading-relaxed">
            No active action configurations found for this agent. Configure an
            action in the agent settings before creating a standing approval.
          </p>
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
          </select>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              Select an action configuration to pre-populate constraints.
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
  paramValues,
  paramModes,
  onParamValueChange,
  onParamModeChange,
  manualConstraintsJson,
  onManualConstraintsJsonChange,
  isPending,
}: {
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

  return (
    <div className="space-y-3">
      <div className="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info className="text-muted-foreground mt-0.5 size-4 shrink-0" />
        <p className="text-muted-foreground text-sm">
          Standing approvals require parameter constraints. At least one
          parameter must be Fixed or Pattern — not all wildcards.
        </p>
      </div>

      {configSchema?.properties ? (
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
      ) : (
        <div className="space-y-2">
          <Label htmlFor="sa-manual-constraints">Constraints (JSON)</Label>
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              No parameter schema found for this action. Enter constraints
              manually as a JSON object.
            </p>
          </div>
          <textarea
            id="sa-manual-constraints"
            className="border-input bg-background ring-offset-background focus-visible:ring-ring flex min-h-[100px] w-full rounded-md border px-3 py-2 font-mono text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            placeholder={'{\n  "recipient": "*@mycompany.com",\n  "subject": "*"\n}'}
            value={manualConstraintsJson}
            onChange={(e) => onManualConstraintsJsonChange(e.target.value)}
            disabled={isPending}
          />
          <div className="rounded-lg border border-dashed bg-muted/40 px-3 py-2">
            <p className="text-muted-foreground text-xs leading-relaxed">
              Use <code className="rounded bg-muted px-1 font-mono text-foreground/70">&quot;*&quot;</code> for wildcard
              parameters, but at least one must be non-wildcard.
            </p>
          </div>
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
  currentExecutionCount,
  noExpiry,
  onNoExpiryChange,
}: {
  maxExecutions: string;
  onMaxExecutionsChange: (value: string) => void;
  expiresAt: string;
  onExpiresAtChange: (value: string) => void;
  /** When editing an existing approval, the number of times it has already been used. */
  currentExecutionCount?: number;
  noExpiry?: boolean;
  onNoExpiryChange?: (value: boolean) => void;
}) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="sa-max-executions">Max Executions</Label>
        <Input
          id="sa-max-executions"
          type="number"
          min={currentExecutionCount != null ? String(currentExecutionCount) : "1"}
          step="1"
          placeholder="Unlimited"
          value={maxExecutions}
          onChange={(e) => onMaxExecutionsChange(e.target.value)}
        />
        {currentExecutionCount != null && currentExecutionCount > 0 ? (
          <p className="text-muted-foreground text-sm">
            Already used {currentExecutionCount} time{currentExecutionCount !== 1 ? "s" : ""} — minimum is {currentExecutionCount}. Leave empty for unlimited.
          </p>
        ) : (
          <p className="text-muted-foreground text-sm">
            Leave empty for unlimited executions.
          </p>
        )}
      </div>

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
            : "Set a specific expiration date, or check \"Until revoked\" for no expiry."}
        </p>
      </div>
    </div>
  );
}
