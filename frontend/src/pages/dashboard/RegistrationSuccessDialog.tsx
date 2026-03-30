import { Bot, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { generatePostRegistrationInstructions } from "./agentInstructions";
import { InstructionsBlock } from "./InstructionsBlock";
import { getAgentDisplayName } from "@/lib/agents";
import type { Agent } from "@/hooks/useAgents";

interface RegistrationSuccessDialogProps {
  agent: Agent;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RegistrationSuccessDialog({
  agent,
  open,
  onOpenChange,
}: RegistrationSuccessDialogProps) {
  const name = getAgentDisplayName(agent);
  const origin = window.location.origin;
  const instructions = generatePostRegistrationInstructions(
    agent.agent_id,
    origin,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Openclaw Connected Successfully</DialogTitle>
          <DialogDescription>
            Openclaw is ready. Tell it to run this command to discover what it can do.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Success banner */}
          <div className="flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 px-4 py-3 dark:border-green-800 dark:bg-green-950/50">
            <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            <div className="flex items-center gap-2">
              <Bot className="text-muted-foreground size-4" />
              <span className="text-sm font-medium">{name}</span>
              <span className="text-muted-foreground text-xs">is now registered</span>
            </div>
          </div>

          <InstructionsBlock
            instructions={instructions}
            buttonLabel="Copy Command"
          />
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
