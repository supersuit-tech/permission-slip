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
import { useDeleteCredential } from "@/hooks/useDeleteCredential";
import type { CredentialSummary } from "@/hooks/useCredentials";

interface RemoveCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credential: CredentialSummary;
}

export function RemoveCredentialDialog({
  open,
  onOpenChange,
  credential,
}: RemoveCredentialDialogProps) {
  const { deleteCredential, isLoading } = useDeleteCredential();

  async function handleRemove() {
    try {
      await deleteCredential(credential.id);
      toast.success(
        `Credential for ${credential.service}${credential.label ? ` (${credential.label})` : ""} removed`,
      );
      onOpenChange(false);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : `Failed to remove credential for ${credential.service}`,
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Remove Credential</DialogTitle>
          <DialogDescription>
            This will permanently delete the stored credential for{" "}
            <strong>{credential.service}</strong>
            {credential.label && (
              <>
                {" "}
                (<strong>{credential.label}</strong>)
              </>
            )}
            . Actions requiring this credential will fail until new credentials
            are stored. This action is irreversible.
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
            onClick={handleRemove}
            disabled={isLoading}
          >
            {isLoading && <Loader2 className="animate-spin" />}
            Remove
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
