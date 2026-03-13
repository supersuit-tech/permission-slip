import { Loader2, AlertTriangle } from "lucide-react";
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
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";

interface DisconnectOAuthDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  connectionId: string;
  providerName: string;
  displayName?: string;
}

export function DisconnectOAuthDialog({
  open,
  onOpenChange,
  connectionId,
  providerName,
  displayName,
}: DisconnectOAuthDialogProps) {
  const { disconnect, isLoading } = useDisconnectOAuth();

  async function handleDisconnect() {
    try {
      await disconnect(connectionId);
      toast.success(`Disconnected ${providerName}`);
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to disconnect",
      );
    }
  }

  const label = displayName
    ? `${providerName} (${displayName})`
    : providerName;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertTriangle className="text-destructive size-5" />
            Disconnect {label}?
          </DialogTitle>
          <DialogDescription>
            Disconnecting will remove this account from{" "}
            <strong>all agents and connectors</strong> that use it. Actions
            requiring this credential will fail until a new connection is
            established.
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
            onClick={handleDisconnect}
            disabled={isLoading}
          >
            {isLoading && <Loader2 className="animate-spin" />}
            Disconnect
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
