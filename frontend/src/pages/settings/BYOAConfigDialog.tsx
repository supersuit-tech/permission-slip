import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useSaveOAuthProviderConfig } from "@/hooks/useSaveOAuthProviderConfig";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface BYOAConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider: string;
  providerLabel: string;
}

export function BYOAConfigDialog({
  open,
  onOpenChange,
  provider,
  providerLabel,
}: BYOAConfigDialogProps) {
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const { save, isLoading } = useSaveOAuthProviderConfig();

  function handleClose() {
    setClientId("");
    setClientSecret("");
    onOpenChange(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const trimmedId = clientId.trim();
    const trimmedSecret = clientSecret.trim();
    if (!trimmedId || !trimmedSecret) {
      toast.error("Both Client ID and Client Secret are required.");
      return;
    }

    try {
      await save({
        provider,
        clientId: trimmedId,
        clientSecret: trimmedSecret,
      });
      toast.success(
        `OAuth credentials saved for ${providerLabel}. You can now connect your account.`,
      );
      handleClose();
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to save credentials.";
      toast.error(message);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              Configure {providerLabel} OAuth App
            </DialogTitle>
            <DialogDescription>
              Provide your own OAuth client credentials for{" "}
              {providerLabel}. These are encrypted and stored securely in the
              vault.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="byoa-client-id">Client ID</Label>
              <Input
                id="byoa-client-id"
                placeholder="Your OAuth client ID"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                disabled={isLoading}
                autoComplete="off"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="byoa-client-secret">Client Secret</Label>
              <Input
                id="byoa-client-secret"
                type="password"
                placeholder="Your OAuth client secret"
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
                disabled={isLoading}
                autoComplete="off"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="size-4 animate-spin" />
                  Saving...
                </>
              ) : (
                "Save Credentials"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
