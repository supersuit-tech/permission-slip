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
import { useDeleteActionConfig } from "@/hooks/useDeleteActionConfig";
import { useLinkedStandingApprovalsForConfig } from "@/hooks/useLinkedStandingApprovalsForConfig";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";

interface DeleteActionConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  config: ActionConfiguration;
  agentId: number;
}

export function DeleteActionConfigDialog({
  open,
  onOpenChange,
  config,
  agentId,
}: DeleteActionConfigDialogProps) {
  const { deleteActionConfig, isPending } = useDeleteActionConfig();
  const embeddedCount = config.linked_standing_approvals?.length ?? 0;
  const needsFetch = embeddedCount === 0;
  const { data: linkedStandingData } = useLinkedStandingApprovalsForConfig(
    config.id,
    needsFetch && open,
  );
  const linkedCount =
    embeddedCount > 0
      ? embeddedCount
      : (linkedStandingData?.data?.length ?? 0);

  async function handleDelete() {
    try {
      await deleteActionConfig({ configId: config.id, agentId });
      toast.success(`Configuration "${config.name}" deleted`);
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to delete action configuration",
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Action Configuration</DialogTitle>
          <DialogDescription>
            This will permanently delete the configuration{" "}
            <strong>{config.name}</strong> ({config.action_type}). The agent
            will no longer be able to reference this configuration. This
            action is irreversible.
            {linkedCount > 0 && (
              <>
                {" "}
                This action configuration has {linkedCount} active standing
                approval{linkedCount === 1 ? "" : "s"}. Deleting it will also
                revoke them.
              </>
            )}
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
            onClick={handleDelete}
            disabled={isPending}
          >
            {isPending && <Loader2 className="animate-spin" />}
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
