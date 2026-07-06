import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

type DeclineAllApprovalsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pendingCount: number;
  onConfirm: () => Promise<void>;
  isPending: boolean;
};

export function DeclineAllApprovalsDialog({
  open,
  onOpenChange,
  pendingCount,
  onConfirm,
  isPending,
}: DeclineAllApprovalsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Decline all pending requests?</DialogTitle>
          <DialogDescription>
            Decline all {pendingCount} pending request
            {pendingCount !== 1 ? "s" : ""}? Agents will be notified that these
            requests were denied.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={isPending}
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={isPending}
            onClick={() => {
              void onConfirm();
            }}
          >
            {isPending ? "Declining…" : "Decline all"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
