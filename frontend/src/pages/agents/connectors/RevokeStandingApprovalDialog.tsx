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
import type { StandingApproval } from "@/hooks/useStandingApprovals";

interface RevokeStandingApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: StandingApproval;
  onRevoked?: () => void;
}

export function RevokeStandingApprovalDialog({
  open,
  onOpenChange,
  rule,
  onRevoked,
}: RevokeStandingApprovalDialogProps) {
  const { revokeStandingApproval, isPending } = useRevokeStandingApproval();

  async function handleRevoke() {
    try {
      await revokeStandingApproval(rule.standing_approval_id);
      toast.success(
        `Standing approval "${rule.name ?? rule.action_type}" revoked`,
      );
      onOpenChange(false);
      onRevoked?.();
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
            This will revoke the standing approval{" "}
            <strong>{rule.name ?? rule.action_type}</strong> (
            {rule.action_type}). Matching requests will require your approval
            again. This action is irreversible.
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
            onClick={() => void handleRevoke()}
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
