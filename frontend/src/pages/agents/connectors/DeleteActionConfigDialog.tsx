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
            {config.action_type === "*" ? (
              <>
                This will permanently delete the enable-all configuration{" "}
                <strong>{config.name}</strong>. The agent will lose access to{" "}
                <strong>all actions</strong> on this connector. This action is
                irreversible.
              </>
            ) : (
              <>
                This will permanently delete the configuration{" "}
                <strong>{config.name}</strong> ({config.action_type}). The agent
                will no longer be able to reference this configuration. This
                action is irreversible.
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
