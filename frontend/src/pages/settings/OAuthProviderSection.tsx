import { useState } from "react";
import { KeyRound, Loader2, Settings2, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { useOAuthProviderConfigs } from "@/hooks/useOAuthProviderConfigs";
import { useDeleteOAuthProviderConfig } from "@/hooks/useDeleteOAuthProviderConfig";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { providerLabel } from "@/lib/providerLabels";
import { BYOAConfigDialog } from "./BYOAConfigDialog";

export function OAuthProviderSection() {
  const { providers, isLoading: providersLoading } = useOAuthProviders();
  const { configs, isLoading: configsLoading } = useOAuthProviderConfigs();
  const { deleteConfig, isLoading: isDeleting } =
    useDeleteOAuthProviderConfig();
  const [byoaDialog, setBYOADialog] = useState<{
    provider: string;
    label: string;
  } | null>(null);

  const isLoading = providersLoading || configsLoading;

  // Providers that need BYOA setup: registered but lacking client credentials
  const unconfiguredProviders = providers.filter((p) => !p.has_credentials);

  // BYOA configs that have been saved
  const configuredByoaProviders = configs;

  async function handleDeleteConfig(provider: string) {
    try {
      await deleteConfig(provider);
      toast.success(`OAuth credentials removed for ${providerLabel(provider)}.`);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to remove credentials.";
      toast.error(message);
    }
  }

  // Don't render this section if there are no BYOA configs and no unconfigured providers
  if (
    !isLoading &&
    configuredByoaProviders.length === 0 &&
    unconfiguredProviders.length === 0
  ) {
    return null;
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Settings2 className="text-muted-foreground size-5" />
            <CardTitle>OAuth App Credentials</CardTitle>
          </div>
          <CardDescription>
            Configure your own OAuth client credentials for providers that
            aren&apos;t pre-configured. Required for self-hosted deployments or
            custom providers.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading OAuth provider configs"
            >
              <Loader2 className="text-muted-foreground size-5 animate-spin" />
            </div>
          ) : (
            <div className="space-y-3">
              {configuredByoaProviders.map((config) => (
                <div
                  key={config.provider}
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-2">
                      <KeyRound className="text-muted-foreground size-4" />
                      <p className="text-sm font-medium">
                        {providerLabel(config.provider)}
                      </p>
                    </div>
                    <p className="text-muted-foreground text-xs">
                      Custom credentials configured &middot; Added{" "}
                      {new Date(config.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">BYOA</Badge>
                    <InlineConfirmButton
                      confirmLabel="Remove"
                      isProcessing={isDeleting}
                      onConfirm={() => handleDeleteConfig(config.provider)}
                    >
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove ${providerLabel(config.provider)} credentials`}
                      >
                        <Trash2 className="text-muted-foreground size-4" />
                      </Button>
                    </InlineConfirmButton>
                  </div>
                </div>
              ))}

              {unconfiguredProviders.map((provider) => (
                <div
                  key={provider.id}
                  className="flex items-center justify-between rounded-lg border border-dashed p-4"
                >
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium">
                      {providerLabel(provider.id)}
                    </p>
                    <p className="text-muted-foreground text-xs">
                      Needs OAuth client credentials to enable connections
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      setBYOADialog({
                        provider: provider.id,
                        label: providerLabel(provider.id),
                      })
                    }
                  >
                    Configure
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {byoaDialog && (
        <BYOAConfigDialog
          open
          onOpenChange={() => setBYOADialog(null)}
          provider={byoaDialog.provider}
          providerLabel={byoaDialog.label}
        />
      )}
    </>
  );
}
