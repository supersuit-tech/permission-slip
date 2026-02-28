import { useState, type FormEvent } from "react";
import { Eye, EyeOff, Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useStoreCredential } from "@/hooks/useStoreCredential";
import { FormError } from "@/components/FormError";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface AddCredentialDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface CredentialField {
  key: string;
  value: string;
}

const SERVICE_PATTERN = /^[a-z][a-z0-9_.-]*$/;
const CREDENTIAL_KEY_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_.-]*$/;

export function AddCredentialDialog({
  open,
  onOpenChange,
}: AddCredentialDialogProps) {
  const { storeCredential, isLoading } = useStoreCredential();
  const [service, setService] = useState("");
  const [label, setLabel] = useState("");
  const [fields, setFields] = useState<CredentialField[]>([
    { key: "", value: "" },
  ]);
  const [showValues, setShowValues] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  function resetForm() {
    setService("");
    setLabel("");
    setFields([{ key: "", value: "" }]);
    setShowValues(false);
    setFormError(null);
  }

  function handleClose(nextOpen: boolean) {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  }

  function addField() {
    setFields((prev) => [...prev, { key: "", value: "" }]);
  }

  function removeField(index: number) {
    setFields((prev) => prev.filter((_, i) => i !== index));
  }

  function updateField(index: number, prop: "key" | "value", val: string) {
    setFields((prev) =>
      prev.map((f, i) => (i === index ? { ...f, [prop]: val } : f)),
    );
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);

    const trimmedService = service.trim();
    if (!trimmedService) return;

    if (!SERVICE_PATTERN.test(trimmedService)) {
      setFormError(
        "Service name must start with a lowercase letter and contain only lowercase letters, numbers, dots, hyphens, or underscores.",
      );
      return;
    }

    // Build credentials object from key-value fields.
    const credentials: Record<string, string> = {};
    for (const field of fields) {
      const key = field.key.trim();
      const value = field.value.trim();
      if (!key && !value) continue;
      if (!key || !value) {
        setFormError("Each credential field must have both a key and a value.");
        return;
      }
      if (!CREDENTIAL_KEY_PATTERN.test(key)) {
        setFormError(
          `Invalid key "${key}". Keys must start with a letter or underscore and contain only letters, numbers, underscores, dots, or hyphens.`,
        );
        return;
      }
      credentials[key] = value;
    }

    if (Object.keys(credentials).length === 0) {
      setFormError("Add at least one credential field.");
      return;
    }

    try {
      await storeCredential({
        service: trimmedService,
        credentials,
        label: label.trim() || undefined,
      });
      toast.success(`Credential for ${trimmedService} stored successfully.`);
      handleClose(false);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to store credential.";
      setFormError(message);
    }
  }

  const hasValidField = fields.some(
    (f) => f.key.trim() !== "" && f.value.trim() !== "",
  );

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Credential</DialogTitle>
          <DialogDescription>
            Store API credentials for an external service. Credentials are
            encrypted at rest and only used during action execution.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="cred-service">Service</Label>
              <Input
                id="cred-service"
                placeholder="github"
                value={service}
                onChange={(e) => {
                  setService(e.target.value.toLowerCase());
                  setFormError(null);
                }}
                pattern="^[a-z][a-z0-9_.\-]*$"
                title="Lowercase letters, numbers, dots, hyphens, and underscores. Must start with a letter."
                required
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                Must match the service name expected by the connector (e.g.
                github, slack, stripe).
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="cred-label">
                Label{" "}
                <span className="text-muted-foreground font-normal">
                  (optional)
                </span>
              </Label>
              <Input
                id="cred-label"
                placeholder="Personal Access Token"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                maxLength={256}
              />
              <p className="text-xs text-muted-foreground">
                Helps distinguish multiple credentials for the same service.
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Credential Fields</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowValues((prev) => !prev)}
                className="h-auto px-2 py-1 text-xs"
                aria-label={showValues ? "Hide values" : "Show values"}
              >
                {showValues ? (
                  <EyeOff className="mr-1 size-3" />
                ) : (
                  <Eye className="mr-1 size-3" />
                )}
                {showValues ? "Hide" : "Show"}
              </Button>
            </div>
            <div className="space-y-2">
              {fields.map((field, index) => (
                <div key={index} className="flex items-center gap-2">
                  <Input
                    placeholder="key (e.g. api_key)"
                    value={field.key}
                    onChange={(e) => {
                      updateField(index, "key", e.target.value);
                      setFormError(null);
                    }}
                    className="flex-1"
                    aria-label={`Credential key ${index + 1}`}
                  />
                  <Input
                    placeholder="value"
                    type={showValues ? "text" : "password"}
                    value={field.value}
                    onChange={(e) =>
                      updateField(index, "value", e.target.value)
                    }
                    className="flex-1"
                    aria-label={`Credential value ${index + 1}`}
                  />
                  {fields.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeField(index)}
                      aria-label={`Remove field ${index + 1}`}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={addField}
              className="mt-1"
            >
              <Plus className="size-4" />
              Add Field
            </Button>
          </div>

          <FormError error={formError} />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleClose(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !service.trim() || !hasValidField}
            >
              {isLoading && <Loader2 className="animate-spin" />}
              Store Credential
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
