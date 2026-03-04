import { useState } from "react";
import { KeyRound, Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useCredentials } from "@/hooks/useCredentials";
import { useDeleteCredential } from "@/hooks/useDeleteCredential";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { LimitBadge } from "@/components/LimitBadge";
import { UpgradePrompt } from "@/components/UpgradePrompt";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AddCredentialDialog } from "./AddCredentialDialog";

export function CredentialSection() {
  const { credentials, isLoading, error } = useCredentials();
  const { deleteCredential, isLoading: isDeleting } = useDeleteCredential();
  const { billingPlan } = useBillingPlan();
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const maxCredentials = billingPlan?.plan?.max_credentials ?? null;
  const credentialCount = billingPlan?.usage?.credentials ?? credentials.length;
  const atLimit = maxCredentials != null && credentialCount >= maxCredentials;

  async function handleDelete(credentialId: string, service: string) {
    try {
      await deleteCredential(credentialId);
      toast.success(`Credential for ${service} deleted.`);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to delete credential.";
      toast.error(message);
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <KeyRound className="text-muted-foreground size-5" />
              <CardTitle>Credential Vault</CardTitle>
              {billingPlan?.plan ? (
                <LimitBadge
                  current={credentialCount}
                  max={maxCredentials}
                  resource="credentials"
                />
              ) : credentials.length > 0 ? (
                <Badge variant="outline" className="ml-1">
                  {credentials.length}
                </Badge>
              ) : null}
            </div>
            {!atLimit && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setAddDialogOpen(true)}
              >
                <Plus className="size-4" />
                Add Credential
              </Button>
            )}
          </div>
          <CardDescription>
            Service credentials stored in your encrypted vault. These are used by
            connectors to execute actions on your behalf.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading credentials"
            >
              <Loader2 className="text-muted-foreground size-5 animate-spin" />
            </div>
          ) : error ? (
            <p className="text-destructive text-sm">{error}</p>
          ) : credentials.length === 0 ? (
            <p className="text-muted-foreground py-4 text-center text-sm">
              No credentials stored yet. Click &ldquo;Add Credential&rdquo; to
              store API keys or tokens for your connected services.
            </p>
          ) : (
            <div className="space-y-3">
              {credentials.map((cred) => (
                <div
                  key={cred.id}
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium">{cred.service}</p>
                    <p className="text-xs text-muted-foreground">
                      {cred.label ?? "Credential"} &middot; Added{" "}
                      {new Date(cred.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">Active</Badge>
                    <InlineConfirmButton
                      confirmLabel="Delete"
                      isProcessing={isDeleting}
                      onConfirm={() => handleDelete(cred.id, cred.service)}
                    >
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Delete ${cred.service} credential`}
                      >
                        <Trash2 className="text-muted-foreground size-4" />
                      </Button>
                    </InlineConfirmButton>
                  </div>
                </div>
              ))}
            </div>
          )}
          {atLimit && (
            <div className="mt-4">
              <UpgradePrompt feature="Upgrade to store more credentials." />
            </div>
          )}
        </CardContent>
      </Card>

      <AddCredentialDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
      />
    </>
  );
}
