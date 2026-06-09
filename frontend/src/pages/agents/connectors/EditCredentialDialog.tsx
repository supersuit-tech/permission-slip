import { useEffect, useMemo, useState } from "react";
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
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useUpdateCredential } from "@/hooks/useUpdateCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import validation from "@/lib/validation";
import { BridgeTestConnectionButton } from "./BridgeTestConnectionButton";
import { resolveStaticCredentialFields } from "./credentialFields";

interface EditCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credential: RequiredCredential;
  storedCredential: CredentialSummary;
  onSuccess?: () => void;
}

const BLANK_FIELD_HINT = "Leave blank to keep current value";

export function EditCredentialDialog({
  open,
  onOpenChange,
  credential,
  storedCredential,
  onSuccess,
}: EditCredentialDialogProps) {
  const { updateCredential, isLoading } = useUpdateCredential();
  const [label, setLabel] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({});

  const staticFields = useMemo(
    () => resolveStaticCredentialFields(credential),
    [credential],
  );

  const staticFieldSig = useMemo(
    () => staticFields.map((f) => f.key).join("\x1e"),
    [staticFields],
  );

  useEffect(() => {
    if (!open) return;
    setLabel(storedCredential.label ?? "");
    setUsername("");
    setPassword("");
    const next: Record<string, string> = {};
    for (const f of staticFields) {
      next[f.key] = "";
    }
    setFieldValues(next);
  }, [open, storedCredential.id, storedCredential.label, staticFieldSig, staticFields]);

  function resetForm() {
    setLabel(storedCredential.label ?? "");
    setUsername("");
    setPassword("");
    const cleared: Record<string, string> = {};
    for (const f of staticFields) {
      cleared[f.key] = "";
    }
    setFieldValues(cleared);
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const body: {
      label?: string | null;
      credentials?: Record<string, string>;
    } = {};

    const trimmedLabel = label.trim();
    const originalLabel = storedCredential.label ?? "";
    if (trimmedLabel !== originalLabel) {
      body.label = trimmedLabel === "" ? null : trimmedLabel;
    }

    const credentialUpdates = buildCredentialUpdates();
    if (credentialUpdates && Object.keys(credentialUpdates).length > 0) {
      body.credentials = credentialUpdates;
    }

    if (body.label === undefined && body.credentials === undefined) {
      toast.error("Change at least one field to save");
      return;
    }

    try {
      await updateCredential(storedCredential.id, body);
      toast.success(`Credentials updated for ${credential.service}`);
      resetForm();
      onOpenChange(false);
      onSuccess?.();
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : `Failed to update credentials for ${credential.service}`,
      );
    }
  }

  function buildCredentialUpdates(): Record<string, string> | null {
    if (credential.auth_type === "basic") {
      const updates: Record<string, string> = {};
      if (username.trim()) updates.username = username.trim();
      if (password.trim()) updates.password = password.trim();
      return updates;
    }

    const updates: Record<string, string> = {};
    for (const f of staticFields) {
      const v = (fieldValues[f.key] ?? "").trim();
      if (v) {
        updates[f.key] = v;
      }
    }
    return updates;
  }

  const isBasic = credential.auth_type === "basic";

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Credential</DialogTitle>
            <DialogDescription>
              Update credentials for <strong>{credential.service}</strong>.
              Secret values are never shown — leave fields blank to keep the
              current value.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-cred-label">Label (optional)</Label>
              <Input
                id="edit-cred-label"
                placeholder={`e.g. Personal ${credential.service}`}
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                maxLength={validation.credentialLabel.maxLength}
                disabled={isLoading}
              />
              <p className="text-muted-foreground text-xs">
                Clear the label to remove it.
              </p>
            </div>
            {isBasic ? (
              <>
                <div className="space-y-2">
                  <Label htmlFor="edit-cred-username">Username</Label>
                  <Input
                    id="edit-cred-username"
                    placeholder="Username or email"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={isLoading}
                    autoComplete="off"
                  />
                  <p className="text-muted-foreground text-xs">
                    {BLANK_FIELD_HINT}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-cred-password">Password / API Token</Label>
                  <Input
                    id="edit-cred-password"
                    type="password"
                    placeholder="Enter new password or token"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={isLoading}
                    autoComplete="off"
                  />
                  <p className="text-muted-foreground text-xs">
                    {BLANK_FIELD_HINT}
                  </p>
                </div>
              </>
            ) : (
              staticFields.map((f) => (
                <div key={f.key} className="space-y-2">
                  <Label htmlFor={`edit-cred-field-${f.key}`}>{f.label}</Label>
                  <Input
                    id={`edit-cred-field-${f.key}`}
                    type={f.secret ? "password" : "text"}
                    placeholder={f.placeholder || undefined}
                    value={fieldValues[f.key] ?? ""}
                    onChange={(e) =>
                      setFieldValues((prev) => ({
                        ...prev,
                        [f.key]: e.target.value,
                      }))
                    }
                    disabled={isLoading}
                    autoComplete="off"
                  />
                  <p className="text-muted-foreground text-xs">
                    {BLANK_FIELD_HINT}
                  </p>
                  {f.helpText ? (
                    <p className="text-muted-foreground text-xs">{f.helpText}</p>
                  ) : null}
                </div>
              ))
            )}
            <BridgeTestConnectionButton
              service={credential.service}
              credentialId={storedCredential.id}
              buildCredentials={buildCredentialUpdates}
              disabled={isLoading}
            />
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
              Save Changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
