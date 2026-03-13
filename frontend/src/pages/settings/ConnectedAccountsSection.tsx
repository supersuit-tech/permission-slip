import { useState } from "react";
import {
  AlertTriangle,
  Link2,
  Loader2,
  LogIn,
  Plus,
  Unplug,
} from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
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
import { DisconnectOAuthDialog } from "@/pages/agents/connectors/DisconnectOAuthDialog";

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

  const [instanceDialogState, setInstanceDialogState] = useState<{
    provider: string;
    replaceId?: string;
  } | null>(null);

  const [disconnectTarget, setDisconnectTarget] = useState<{
    id: string;
    provider: string;
    displayName?: string;
  } | null>(null);

  function handleConnect(providerId: string) {
    if (!session?.access_token) return;
    if (providerId in INSTANCE_SUBDOMAIN_PROVIDERS) {
      setInstanceDialogState({ provider: providerId });
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
    );
  }

  function handleReconnect(connectionId: string, providerId: string) {
    if (!session?.access_token) return;
    if (providerId in INSTANCE_SUBDOMAIN_PROVIDERS) {
      setInstanceDialogState({ provider: providerId, replaceId: connectionId });
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
      { replaceId: connectionId },
    );
  }

  // Providers that are ready to connect (have credentials configured)
  const configuredProviders = providers.filter((p) => p.has_credentials);

  // Group connections by provider for display
  const connectedProviderIds = new Set(connections.map((c) => c.provider));

  // Providers with no connections yet — show as initial "Connect" buttons
  const unconnectedProviders = configuredProviders.filter(
    (p) => !connectedProviderIds.has(p.id),
  );

  // Providers that already have connections — show "Add another account" buttons
  const connectedProviderIdsWithCredentials = configuredProviders
    .filter((p) => connectedProviderIds.has(p.id))
    .map((p) => p.id);

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
                key={conn.id}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="space-y-0.5">
                  <p className="text-sm font-medium">
                    {providerLabel(conn.provider)}
                    {conn.display_name && (
                      <span className="text-muted-foreground ml-1.5 font-normal">
                        ({conn.display_name})
                      </span>
                    )}
                    {!conn.display_name && conn.instance && (
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
                      onClick={() => handleReconnect(conn.id, conn.provider)}
                    >
                      <LogIn className="size-4" />
                      Re-authorize
                    </Button>
                  ) : (
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          handleReconnect(conn.id, conn.provider)
                        }
                      >
                        <LogIn className="size-4" />
                        Reconnect
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Disconnect ${providerLabel(conn.provider)}`}
                        onClick={() =>
                          setDisconnectTarget({
                            id: conn.id,
                            provider: conn.provider,
                            displayName: conn.display_name,
                          })
                        }
                      >
                        <Unplug className="text-muted-foreground size-4" />
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}

            {/* "Add another account" for providers that already have connections */}
            {connectedProviderIdsWithCredentials.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2">
                {connectedProviderIdsWithCredentials.map((providerId) => (
                  <Button
                    key={`add-${providerId}`}
                    variant="outline"
                    size="sm"
                    onClick={() => handleConnect(providerId)}
                  >
                    <Plus className="size-4" />
                    Add another {providerLabel(providerId)} account
                  </Button>
                ))}
              </div>
            )}

            {/* "Connect" for providers with no connections yet */}
            {unconnectedProviders.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2">
                {unconnectedProviders.map((provider) => (
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

            {connections.length === 0 && configuredProviders.length === 0 && (
              <p className="text-muted-foreground py-4 text-center text-sm">
                No OAuth providers are configured yet. Set up client credentials
                to enable OAuth connections.
              </p>
            )}
          </div>
        )}
      </CardContent>

      {instanceDialogState &&
        session?.access_token &&
        instanceDialogState.provider in INSTANCE_SUBDOMAIN_PROVIDERS && (
          <InstanceSubdomainDialog
            open
            config={INSTANCE_SUBDOMAIN_PROVIDERS[instanceDialogState.provider]!}
            onOpenChange={(open) => {
              if (!open) setInstanceDialogState(null);
            }}
            onSubmit={(value) => {
              if (!session?.access_token || !instanceDialogState) return;
              const cfg =
                INSTANCE_SUBDOMAIN_PROVIDERS[instanceDialogState.provider]!;
              const url = getOAuthAuthorizeUrl(
                instanceDialogState.provider,
                session.access_token,
                instanceDialogState.replaceId
                  ? { replaceId: instanceDialogState.replaceId }
                  : undefined,
              );
              window.location.href = `${url}&${cfg.queryParam}=${encodeURIComponent(value)}`;
              setInstanceDialogState(null);
            }}
          />
        )}

      {disconnectTarget && (
        <DisconnectOAuthDialog
          open
          onOpenChange={(open) => {
            if (!open) setDisconnectTarget(null);
          }}
          connectionId={disconnectTarget.id}
          providerName={providerLabel(disconnectTarget.provider)}
          displayName={disconnectTarget.displayName}
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
