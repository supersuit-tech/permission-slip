import { useState } from "react";
import { Loader2, Shield } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useApproveStandingApprovalRequest } from "@/hooks/useApproveStandingApprovalRequest";
import { useDenyStandingApprovalRequest } from "@/hooks/useDenyStandingApprovalRequest";
import { useActionSchema } from "@/hooks/useActionSchema";
import type { StandingApprovalRequestSummary } from "@/hooks/useStandingApprovalRequests";
import { formatConnectorDisplayName } from "./approvalConnectorLabel";
import { ConstraintsSummary } from "./ConstraintsSummary";
import { useStandingApprovalInstanceAmbiguityWarning } from "@/hooks/useStandingApprovalInstanceAmbiguityWarning";
import { StandingApprovalInstanceAmbiguityWarning } from "./StandingApprovalInstanceAmbiguityWarning";

interface ReviewStandingApprovalRequestDialogProps {
  request: StandingApprovalRequestSummary;
  agentDisplayName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ReviewStandingApprovalRequestDialog({
  request,
  agentDisplayName,
  open,
  onOpenChange,
}: ReviewStandingApprovalRequestDialogProps) {
  const { approveRequest, isPending: isApproving } = useApproveStandingApprovalRequest();
  const { denyRequest, isPending: isDenying } = useDenyStandingApprovalRequest();
  const { connectorName } = useActionSchema(request.action_type);
  const { showWarning: showInstanceAmbiguityWarning } =
    useStandingApprovalInstanceAmbiguityWarning(request);
  const [done, setDone] = useState<"approved" | "denied" | null>(null);

  const connectorDisplayName = formatConnectorDisplayName({
    connectorName: request.connector_name ?? connectorName,
    actionType: request.action_type,
    instanceDisplay: request.connector_instance_display,
  });
  const constraints =
    request.constraints && typeof request.constraints === "object"
      ? (request.constraints as Record<string, unknown>)
      : null;
  const busy = isApproving || isDenying;

  async function handleApprove() {
    try {
      await approveRequest(request.request_id);
      setDone("approved");
      toast.success("Auto-approve rule activated");
      setTimeout(() => onOpenChange(false), 1500);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to approve rule";
      toast.error(message);
    }
  }

  async function handleDeny() {
    try {
      await denyRequest(request.request_id);
      setDone("denied");
      toast.success("Rule proposal denied");
      setTimeout(() => onOpenChange(false), 1500);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to deny rule";
      toast.error(message);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2">
            <Shield className="size-5" />
            Rule proposal
            <Badge variant="secondary">Rule proposal</Badge>
          </DialogTitle>
          <DialogDescription>
            Review constraints before approving this standing auto-approve rule.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 text-sm">
          <p>
            <span className="font-medium">{agentDisplayName}</span> wants a standing
            auto-approve rule for{" "}
            <span className="font-medium">{connectorDisplayName}</span> (
            <span className="font-mono">{request.action_type}</span>).
          </p>

          {showInstanceAmbiguityWarning && (
            <StandingApprovalInstanceAmbiguityWarning />
          )}

          <div>
            <p className="text-muted-foreground mb-1 text-xs font-medium uppercase tracking-wide">
              Constraints
            </p>
            <ConstraintsSummary constraints={constraints} />
            <p className="text-muted-foreground mt-2 text-xs">
              Verified fields (<span className="font-mono">$meta</span>) match
              server-fetched envelope data, not agent-supplied parameters.
            </p>
          </div>

          {done === "approved" && (
            <p className="text-green-600 dark:text-green-400">Rule approved and active.</p>
          )}
          {done === "denied" && (
            <p className="text-muted-foreground">Proposal denied.</p>
          )}
        </div>

        {!done && (
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={handleDeny} disabled={busy}>
              {isDenying ? <Loader2 className="size-4 animate-spin" /> : "Deny"}
            </Button>
            <Button onClick={handleApprove} disabled={busy}>
              {isApproving ? <Loader2 className="size-4 animate-spin" /> : "Approve rule"}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
