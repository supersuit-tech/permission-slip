import { useState } from "react";
import { ChevronDown, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useUpdateActionConfig } from "@/hooks/useUpdateActionConfig";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import {
  connectorInstanceFromParameters,
  instanceSelectLabel,
  mergeConnectorInstanceIntoParameters,
  parametersWithoutConnectorInstance,
  resolveConnectorInstanceAccountLabel,
} from "./connectorInstanceAccount";

interface ActionConfigAccountRescopeCellProps {
  agentId: number;
  config: ActionConfiguration;
  instances: AgentConnectorInstance[];
  onSuccess?: () => void;
}

export function ActionConfigAccountRescopeCell({
  agentId,
  config,
  instances,
  onSuccess,
}: ActionConfigAccountRescopeCellProps) {
  const { updateActionConfig, isPending } = useUpdateActionConfig();
  const [open, setOpen] = useState(false);

  const currentValue = connectorInstanceFromParameters(
    config.parameters as Record<string, unknown>,
  );
  const accountLabel = resolveConnectorInstanceAccountLabel(
    config.parameters.connector_instance,
    instances,
  );

  async function handleSelect(nextValue: string) {
    if (nextValue === currentValue) {
      setOpen(false);
      return;
    }

    try {
      const parameters = mergeConnectorInstanceIntoParameters(
        parametersWithoutConnectorInstance(
          config.parameters as Record<string, unknown>,
        ),
        nextValue,
      );
      await updateActionConfig({
        configId: config.id,
        agentId,
        body: { parameters },
      });
      toast.success("Account scope updated");
      onSuccess?.();
      setOpen(false);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update account scope",
      );
    }
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-auto gap-1 px-1 py-0.5 font-normal"
          disabled={isPending}
          aria-label={`Change account scope for ${config.name}: ${accountLabel}`}
        >
          <span className="text-sm">{accountLabel}</span>
          {isPending ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <ChevronDown className="text-muted-foreground size-3.5" aria-hidden />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-44">
        <DropdownMenuItem onClick={() => void handleSelect("*")}>
          All accounts
        </DropdownMenuItem>
        {instances.map((instance) => (
          <DropdownMenuItem
            key={instance.connector_instance_id}
            onClick={() => void handleSelect(instance.connector_instance_id)}
          >
            {instanceSelectLabel(instance)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
