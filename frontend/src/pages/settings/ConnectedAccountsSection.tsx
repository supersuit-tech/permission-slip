import { useState } from "react";
import {
  AlertTriangle,
  Link2,
  Loader2,
  LogIn,
  Unplug,
} from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { providerLabel, getOAuthAuthorizeUrl } from "@/lib/oauth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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

/**
 * Config for providers that need a per-instance subdomain before the OAuth
 * redirect can be constructed. Each entry maps a provider ID to the dialog
 * display config and the query parameter name to attach to the authorize URL.
 */
const INSTANCE_SUBDOMAIN_PROVIDERS: Record<
  string,
  {
    title: string;
    domainSuffix: string;
    placeholder: string;
    inputLabel: string;
    inputId: string;
    stripSuffix: RegExp;
    queryParam: string;
    buttonLabel: string;
  }
> = {
  shopify: {
    title: "Connect Shopify Store",
    domainSuffix: ".myshopify.com",
    placeholder: "mystore",
    inputLabel: "Store subdomain",
    inputId: "settings-shop-domain",
    stripSuffix: /\.myshopify\.com$/,
    queryParam: "shop",
    buttonLabel: "Continue to Shopify",
  },
  zendesk: {
    title: "Connect Zendesk",
    domainSuffix: ".zendesk.com",
    placeholder: "mycompany",
    inputLabel: "Subdomain",
    inputId: "settings-zendesk-subdomain",
    stripSuffix: /\.zendesk\.com$/,
    queryParam: "subdomain",
    buttonLabel: "Continue to Zendesk",
  },
};

function statusBadge(status: string) {
  switch (status) {
    case "active":
      return <Badge variant="secondary">Connected</Badge>;
    case "needs_reauth":
      return (
        <Badge variant="destructive" className="gap-1">
          <AlertTriangle className="size-3" />
          Needs Re-auth
        </Badge>
      );
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

export function ConnectedAccountsSection() {
  const { session } = useAuth();
  const { connections, isLoading, error } = useOAuthConnections();
  const { providers } = useOAuthProviders();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();

  async function handleDisconnect(provider: string) {
    try {
      await disconnect(provider);
      toast.success(`${providerLabel(provider)} disconnected.`);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to disconnect.";
      toast.error(message);
    }
  }

  const [instanceDialogProvider, setInstanceDialogProvider] = useState<
    string | null
  >(null);

  function handleConnect(providerId: string) {
    if (!session?.access_token) return;
    if (providerId in INSTANCE_SUBDOMAIN_PROVIDERS) {
      setInstanceDialogProvider(providerId);
      return;
    }
    // Open in same window — the callback redirects back to settings
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
    );
  }

  // Providers that are ready to connect but don't have an active connection
  const connectedProviderIds = new Set(connections.map((c) => c.provider));
  const availableProviders = providers.filter(
    (p) => p.has_credentials && !connectedProviderIds.has(p.id),
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Link2 className="text-muted-foreground size-5" />
          <CardTitle>Connected Accounts</CardTitle>
          {connections.length > 0 && (
            <Badge variant="outline" className="ml-1">
              {connections.length}
            </Badge>
          )}
        </div>
        <CardDescription>
          Connect your accounts to enable connectors that use OAuth for
          authentication. Tokens are encrypted and automatically refreshed.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading connected accounts"
          >
            <Loader2 className="text-muted-foreground size-5 animate-spin" />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : (
          <div className="space-y-3">
            {connections.map((conn) => (
              <div
                key={conn.provider}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="space-y-0.5">
                  <p className="text-sm font-medium">
                    {providerLabel(conn.provider)}
                    {conn.instance && (
                      <span className="text-muted-foreground ml-1.5 font-normal">
                        ({conn.instance})
                      </span>
                    )}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {conn.scopes.length} scope
                    {conn.scopes.length !== 1 ? "s" : ""} granted &middot;
                    Connected{" "}
                    {new Date(conn.connected_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {statusBadge(conn.status)}
                  {conn.status === "needs_reauth" ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleConnect(conn.provider)}
                    >
                      <LogIn className="size-4" />
                      Re-authorize
                    </Button>
                  ) : (
                    <InlineConfirmButton
                      confirmLabel="Disconnect"
                      isProcessing={isDisconnecting}
                      onConfirm={() => handleDisconnect(conn.provider)}
                    >
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Disconnect ${providerLabel(conn.provider)}`}
                      >
                        <Unplug className="text-muted-foreground size-4" />
                      </Button>
                    </InlineConfirmButton>
                  )}
                </div>
              </div>
            ))}

            {availableProviders.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2">
                {availableProviders.map((provider) => (
                  <Button
                    key={provider.id}
                    variant="outline"
                    size="sm"
                    onClick={() => handleConnect(provider.id)}
                  >
                    <LogIn className="size-4" />
                    Connect {providerLabel(provider.id)}
                  </Button>
                ))}
              </div>
            )}

            {connections.length === 0 && availableProviders.length === 0 && (
              <p className="text-muted-foreground py-4 text-center text-sm">
                No OAuth providers are configured yet. Set up client credentials
                to enable OAuth connections.
              </p>
            )}
          </div>
        )}
      </CardContent>

      {instanceDialogProvider &&
        session?.access_token &&
        instanceDialogProvider in INSTANCE_SUBDOMAIN_PROVIDERS && (
          <InstanceSubdomainDialog
            open
            config={INSTANCE_SUBDOMAIN_PROVIDERS[instanceDialogProvider]!}
            onOpenChange={(open) => {
              if (!open) setInstanceDialogProvider(null);
            }}
            onSubmit={(value) => {
              if (!session?.access_token || !instanceDialogProvider) return;
              const cfg =
                INSTANCE_SUBDOMAIN_PROVIDERS[instanceDialogProvider]!;
              const url = getOAuthAuthorizeUrl(
                instanceDialogProvider,
                session.access_token,
              );
              window.location.href = `${url}&${cfg.queryParam}=${encodeURIComponent(value)}`;
              setInstanceDialogProvider(null);
            }}
          />
        )}
    </Card>
  );
}

/**
 * Generic dialog for providers that require a per-instance subdomain before
 * an OAuth redirect can be constructed (e.g. Zendesk, Shopify). Driven by
 * INSTANCE_SUBDOMAIN_PROVIDERS config — no provider-specific JSX needed.
 */
function InstanceSubdomainDialog({
  open,
  config,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  config: (typeof INSTANCE_SUBDOMAIN_PROVIDERS)[string];
  onOpenChange: (open: boolean) => void;
  onSubmit: (value: string) => void;
}) {
  const [value, setValue] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = value.trim().toLowerCase();
    if (!trimmed) return;
    onSubmit(trimmed.replace(config.stripSuffix, ""));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{config.title}</DialogTitle>
          <DialogDescription>
            Enter your {config.inputLabel.toLowerCase()} to begin the OAuth
            connection.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-2 py-4">
            <Label htmlFor={config.inputId}>{config.inputLabel}</Label>
            <div className="flex items-center gap-2">
              <Input
                id={config.inputId}
                placeholder={config.placeholder}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                autoFocus
              />
              <span className="text-muted-foreground whitespace-nowrap text-sm">
                {config.domainSuffix}
              </span>
            </div>
            <p className="text-muted-foreground text-xs">
              e.g. if your URL is {config.placeholder}
              {config.domainSuffix}, enter &quot;{config.placeholder}&quot;
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!value.trim()}>
              <LogIn className="size-4" />
              {config.buttonLabel}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
