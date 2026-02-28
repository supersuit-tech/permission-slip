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
import { useRevokeStandingApproval } from "@/hooks/useRevokeStandingApproval";

interface RevokeStandingApprovalDialogProps {
  standingApprovalId: string;
  actionType: string;
  agentId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RevokeStandingApprovalDialog({
  standingApprovalId,
  actionType,
  agentId,
  open,
  onOpenChange,
}: RevokeStandingApprovalDialogProps) {
  const { revokeStandingApproval, isPending } = useRevokeStandingApproval();

  async function handleConfirm() {
    try {
      await revokeStandingApproval(standingApprovalId);
      toast.success("Standing approval revoked");
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to revoke standing approval",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke Standing Approval</DialogTitle>
          <DialogDescription>
            This will revoke the <strong>{actionType}</strong> standing approval
            for agent <strong>{agentId}</strong>. The agent will need per-request
            approval for this action going forward. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={isPending}
          >
            {isPending && <Loader2 className="animate-spin" />}
            Revoke
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
