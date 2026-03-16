import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useStoreCredential } from "@/hooks/useStoreCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import validation from "@/lib/validation";

interface AddCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credential: RequiredCredential;
  /** Override the credential key used when storing (default: "api_key"). */
  credentialKey?: string;
  /** Override the input field label (default: "API Key"). */
  fieldLabel?: string;
  /** Override the input placeholder (default: "Enter API key or token"). */
  fieldPlaceholder?: string;
  /** Override the dialog title (default: "Add Credential"). */
  title?: string;
  /** Called after credentials are successfully stored, with the new credential ID. */
  onSuccess?: (credentialId: string) => void;
}

export function AddCredentialDialog({
  open,
  onOpenChange,
  credential,
  credentialKey,
  fieldLabel,
  fieldPlaceholder,
  title,
  onSuccess,
}: AddCredentialDialogProps) {
  const { storeCredential, isLoading } = useStoreCredential();
  const [label, setLabel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  function resetForm() {
    setLabel("");
    setApiKey("");
    setUsername("");
    setPassword("");
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const credentials = buildCredentials();
    if (!credentials) return;

    try {
      const result = await storeCredential({
        service: credential.service,
        credentials,
        label: label.trim() || undefined,
      });
      toast.success(`Credentials stored for ${credential.service}`);
      resetForm();
      onOpenChange(false);
      onSuccess?.(result?.id ?? "");
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : `Failed to store credentials for ${credential.service}`,
      );
    }
  }

  const resolvedKey = credentialKey ?? "api_key";
  const resolvedFieldLabel = fieldLabel ?? "API Key";
  const resolvedPlaceholder = fieldPlaceholder ?? "Enter API key or token";
  const resolvedTitle = title ?? "Add Credential";

  function buildCredentials(): Record<string, string> | null {
    if (credential.auth_type === "basic") {
      if (!username.trim() || !password.trim()) {
        toast.error("Username and password are required");
        return null;
      }
      return { username: username.trim(), password: password.trim() };
    }
    if (!apiKey.trim()) {
      toast.error(`${resolvedFieldLabel} is required`);
      return null;
    }
    return { [resolvedKey]: apiKey.trim() };
  }

  const isBasic = credential.auth_type === "basic";

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{resolvedTitle}</DialogTitle>
            <DialogDescription>
              Store credentials for <strong>{credential.service}</strong>.
              Credentials are encrypted at rest and never exposed to agents.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="cred-label">Label (optional)</Label>
              <Input
                id="cred-label"
                placeholder={`e.g. Personal ${credential.service}`}
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                maxLength={validation.credentialLabel.maxLength}
                disabled={isLoading}
              />
            </div>
            {isBasic ? (
              <>
                <div className="space-y-2">
                  <Label htmlFor="cred-username">Username</Label>
                  <Input
                    id="cred-username"
                    placeholder="Username or email"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={isLoading}
                    required
                    autoComplete="off"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="cred-password">Password / API Token</Label>
                  <Input
                    id="cred-password"
                    type="password"
                    placeholder="Enter password or token"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={isLoading}
                    required
                    autoComplete="off"
                  />
                </div>
              </>
            ) : (
              <div className="space-y-2">
                <Label htmlFor="cred-api-key">{resolvedFieldLabel}</Label>
                <Input
                  id="cred-api-key"
                  type="password"
                  placeholder={resolvedPlaceholder}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  disabled={isLoading}
                  required
                  autoComplete="off"
                />
              </div>
            )}
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="secondary"
              onClick={() => handleOpenChange(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="animate-spin" />}
              Store Credential
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
