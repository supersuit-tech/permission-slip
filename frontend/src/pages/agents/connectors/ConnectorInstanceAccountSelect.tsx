import { Label } from "@/components/ui/label";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import { instanceSelectLabel } from "./connectorInstanceAccount";

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

interface ConnectorInstanceAccountSelectProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  instances: AgentConnectorInstance[];
  disabled?: boolean;
}

export function ConnectorInstanceAccountSelect({
  id,
  value,
  onChange,
  instances,
  disabled,
}: ConnectorInstanceAccountSelectProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Account</Label>
      <select
        id={id}
        className={selectClassName}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      >
        <option value="*">All accounts</option>
        {instances.map((instance) => (
          <option
            key={instance.connector_instance_id}
            value={instance.connector_instance_id}
          >
            {instanceSelectLabel(instance)}
          </option>
        ))}
      </select>
      <p className="text-muted-foreground text-xs">
        Which connected account this configuration applies to.
      </p>
    </div>
  );
}
