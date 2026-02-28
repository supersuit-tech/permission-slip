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
import { useDeactivateAgent } from "@/hooks/useDeactivateAgent";

interface DeactivateAgentDialogProps {
  agentId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DeactivateAgentDialog({
  agentId,
  open,
  onOpenChange,
}: DeactivateAgentDialogProps) {
  const { deactivateAgent, isLoading } = useDeactivateAgent();

  async function handleConfirm() {
    try {
      await deactivateAgent(agentId);
      toast.success("Agent deactivated");
      onOpenChange(false);
    } catch {
      toast.error("Failed to deactivate agent");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Deactivate Agent</DialogTitle>
          <DialogDescription>
            This will deactivate <strong>{agentId}</strong> and revoke all its
            standing approvals. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={isLoading}
          >
            {isLoading && <Loader2 className="animate-spin" />}
            Deactivate
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
