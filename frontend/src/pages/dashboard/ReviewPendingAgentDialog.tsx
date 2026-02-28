import { Bot, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  ConfirmationCodeBanner,
  useCountdown,
  formatCountdown,
  urgencyColor,
} from "./approval-components";
import { generateVerificationInstructions } from "./agentInstructions";
import { InstructionsBlock } from "./InstructionsBlock";
import { getAgentDisplayName, parseAgentMetadata } from "@/lib/agents";
import type { Agent } from "@/hooks/useAgents";

interface ReviewPendingAgentDialogProps {
  agent: Agent;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ReviewPendingAgentDialog({
  agent,
  open,
  onOpenChange,
}: ReviewPendingAgentDialogProps) {
  const name = getAgentDisplayName(agent);
  const hasRealName = !!parseAgentMetadata(agent.metadata)?.name;

  const origin = window.location.origin;
  const verificationInstructions = agent.confirmation_code
    ? generateVerificationInstructions(
        agent.agent_id,
        agent.confirmation_code,
        origin,
      )
    : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Complete Agent Registration</DialogTitle>
          <DialogDescription>
            This agent is waiting to complete registration. Copy the
            instructions below and send them to the agent.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Agent info */}
          <div className="flex items-center gap-3">
            <div className="bg-muted rounded-full p-2">
              <Bot className="text-muted-foreground size-5" />
            </div>
            <div>
              <p className="text-sm font-medium">{name}</p>
              {hasRealName && (
                <p className="text-muted-foreground font-mono text-xs">
                  ID: {agent.agent_id}
                </p>
              )}
            </div>
          </div>

          {/* Confirmation code */}
          {agent.confirmation_code && (
            <ConfirmationCodeBanner
              code={agent.confirmation_code}
              copyable
              description="Share this code with the agent to complete registration"
            />
          )}

          {/* Expiry countdown */}
          {agent.expires_at && (
            <ExpiryInfo expiresAt={agent.expires_at} />
          )}

          {/* Verification instructions for the agent */}
          {verificationInstructions && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Instructions for your agent</p>
              <InstructionsBlock
                instructions={verificationInstructions}
                buttonLabel="Copy Verification Instructions"
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            data-testid="review-pending-close"
          >
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ExpiryInfo({ expiresAt }: { expiresAt: string }) {
  const remaining = useCountdown(expiresAt);
  const isExpired = remaining <= 0;

  return (
    <div className="flex items-center gap-2">
      <Clock className={`size-4 ${urgencyColor(remaining)}`} />
      <span className={`text-sm font-medium ${urgencyColor(remaining)}`}>
        {isExpired ? "Registration expired" : `Expires in ${formatCountdown(remaining)}`}
      </span>
    </div>
  );
}
