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
import { useUpdateStandingApproval } from "@/hooks/useUpdateStandingApproval";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import {
  connectorInstanceFromStandingApprovalId,
  instanceSelectLabel,
  standingApprovalConnectorInstanceIdForUpdate,
} from "./connectorInstanceAccount";

interface StandingApprovalAccountRescopeCellProps {
  agentId: number;
  rule: StandingApproval;
  instances: AgentConnectorInstance[];
  onSuccess?: () => void;
}

export function StandingApprovalAccountRescopeCell({
  rule,
  instances,
  onSuccess,
}: StandingApprovalAccountRescopeCellProps) {
  const { updateStandingApproval, isPending } = useUpdateStandingApproval();
  const [open, setOpen] = useState(false);

  const currentValue = connectorInstanceFromStandingApprovalId(
    rule.connector_instance_id,
  );
  const accountLabel =
    currentValue === "*"
      ? "All accounts"
      : (instances.find((i) => i.connector_instance_id === currentValue)
          ?.display?.trim() ?? currentValue);

  async function handleSelect(nextValue: string) {
    if (nextValue === currentValue) {
      setOpen(false);
      return;
    }

    try {
      await updateStandingApproval(rule.standing_approval_id, {
        connector_instance_id:
          standingApprovalConnectorInstanceIdForUpdate(nextValue),
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
          aria-label={`Change account scope for ${rule.name ?? rule.action_type}: ${accountLabel}`}
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
            onClick={() =>
              void handleSelect(instance.connector_instance_id)
            }
          >
            {instanceSelectLabel(instance)}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
