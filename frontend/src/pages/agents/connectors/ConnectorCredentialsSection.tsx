import { useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  Circle,
  ExternalLink,
  Loader2,
  LogIn,
  Plus,
  Trash2,
  Unplug,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
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
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { providerLabel, getOAuthAuthorizeUrl } from "@/lib/oauth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { serviceLabel, authTypeLabel } from "@/lib/labels";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

/** Providers that require a shop subdomain for per-shop OAuth URLs. */
const SHOP_REQUIRED_PROVIDERS = new Set(["shopify"]);

export interface ConnectorCredentialsSectionProps {
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  connectorId,
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasExplicitOAuth = requiredCredentials.some(
    (c) => c.auth_type === "oauth2",
  );
  const hasStatic = requiredCredentials.some((c) => c.auth_type !== "oauth2");
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasStatic,
  });
  // Also fetch OAuth data when there's a matching built-in provider (e.g.
  // Shopify declares api_key in manifest but has a built-in OAuth provider).
  const {
    connections,
    isLoading: connectionsLoading,
    error: oauthError,
  } = useOAuthConnections({ enabled: true });
  const { providers, isLoading: providersLoading } = useOAuthProviders({
    enabled: true,
  });

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  // Check if there's a matching built-in OAuth provider for this connector
  // (covers cases like Shopify where the manifest says api_key but OAuth is available).
  const matchingProvider = providers.find((p) => p.id === connectorId);
  const hasImplicitOAuth = !hasExplicitOAuth && !!matchingProvider;
  const hasOAuth = hasExplicitOAuth || hasImplicitOAuth;

  // Sort credentials: OAuth first, then static
  const sorted = [...requiredCredentials].sort((a, b) => {
    if (a.auth_type === "oauth2" && b.auth_type !== "oauth2") return -1;
    if (a.auth_type !== "oauth2" && b.auth_type === "oauth2") return 1;
    return 0;
  });

  const anyLoading =
    isLoading || connectionsLoading || providersLoading;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials</CardTitle>
        {hasOAuth && hasStatic && (
          <CardDescription>
            Connect via OAuth (recommended) or use an API key as an
            alternative.
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {!hasRequiredCredentials && !matchingProvider ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            This connector does not require any credentials.
          </p>
        ) : anyLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2
              className="text-muted-foreground size-5 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error || oauthError ? (
          <p className="text-destructive text-sm">{error ?? oauthError}</p>
        ) : (
          <div className="space-y-3">
            {/* Implicit OAuth row for connectors with a matching built-in
                provider but no explicit oauth2 credential in their manifest */}
            {hasImplicitOAuth && (
              <>
                <OAuthCredentialRow
                  requiredCredential={{
                    service: connectorId,
                    auth_type: "oauth2",
                    oauth_provider: connectorId,
                  }}
                  connections={connections}
                  providers={providers}
                />
                {sorted.length > 0 && (
                  <div className="flex items-center gap-3 py-1">
                    <div className="bg-border h-px flex-1" />
                    <span className="text-muted-foreground text-xs font-medium uppercase">
                      or
                    </span>
                    <div className="bg-border h-px flex-1" />
                  </div>
                )}
              </>
            )}

            {sorted.map((cred, idx) => {
              const prevCred = idx > 0 ? sorted[idx - 1] : undefined;
              const showOrSeparator =
                prevCred != null &&
                prevCred.auth_type === "oauth2" &&
                cred.auth_type !== "oauth2";

              return (
                <div key={cred.service}>
                  {showOrSeparator && (
                    <div className="flex items-center gap-3 py-1">
                      <div className="bg-border h-px flex-1" />
                      <span className="text-muted-foreground text-xs font-medium uppercase">
                        or
                      </span>
                      <div className="bg-border h-px flex-1" />
                    </div>
                  )}
                  {cred.auth_type === "oauth2" && cred.oauth_provider ? (
                    <OAuthCredentialRow
                      requiredCredential={cred}
                      connections={connections}
                      providers={providers}
                    />
                  ) : (
                    <StaticCredentialRow
                      requiredCredential={cred}
                      storedCredentials={
                        storedByService.get(cred.service) ?? []
                      }
                      isAlternative={hasOAuth}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function OAuthCredentialRow({
  requiredCredential,
  connections,
  providers,
}: {
  requiredCredential: RequiredCredential;
  connections: {
    provider: string;
    status: string;
    scopes: string[];
    connected_at: string;
  }[];
  providers: { id: string; has_credentials: boolean }[];
}) {
  const { session } = useAuth();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const [shopDialogOpen, setShopDialogOpen] = useState(false);

  const providerId = requiredCredential.oauth_provider ?? "";
  const connection = connections.find((c) => c.provider === providerId);
  const provider = providers.find((p) => p.id === providerId);
  const isConnected = connection?.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token || !providerId) return;
    if (SHOP_REQUIRED_PROVIDERS.has(providerId)) {
      setShopDialogOpen(true);
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
    );
  }

  async function handleDisconnect() {
    try {
      await disconnect(providerId);
      toast.success(`${providerLabel(providerId)} disconnected.`);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to disconnect.";
      toast.error(message);
    }
  }

  const scopes = requiredCredential.oauth_scopes ?? [];

  return (
    <>
      <div className="rounded-lg border p-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : needsReauth ? (
              <AlertTriangle className="size-5 shrink-0 text-amber-500" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div>
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium">
                  {providerLabel(providerId)}
                </p>
                <Badge variant="secondary" className="text-xs">
                  OAuth
                </Badge>
                <Badge variant="outline" className="text-xs">
                  Recommended
                </Badge>
              </div>
              <p className="text-muted-foreground text-xs">
                {isConnected
                  ? `Connected ${new Date(connection.connected_at).toLocaleDateString()}`
                  : needsReauth
                    ? "Connection expired or was revoked \u2014 re-authorize to restore access"
                    : `Connect your ${providerLabel(providerId)} account via OAuth for automatic token management`}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isConnected ? (
              <>
                <span className="text-xs font-medium text-green-600 dark:text-green-400">
                  Connected
                </span>
                <InlineConfirmButton
                  confirmLabel="Disconnect"
                  isProcessing={isDisconnecting}
                  onConfirm={handleDisconnect}
                >
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Disconnect ${providerLabel(providerId)}`}
                  >
                    <Unplug className="text-muted-foreground size-4" />
                  </Button>
                </InlineConfirmButton>
              </>
            ) : needsReauth ? (
              <>
                <Badge variant="destructive" className="gap-1 text-xs">
                  <AlertTriangle className="size-3" />
                  Needs Re-auth
                </Badge>
                <Button variant="outline" size="sm" onClick={handleConnect}>
                  <LogIn className="size-3" />
                  Re-authorize
                </Button>
              </>
            ) : (
              <>
                <span className="text-muted-foreground text-xs font-medium">
                  Not connected
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleConnect}
                  disabled={!provider?.has_credentials}
                >
                  <LogIn className="size-3" />
                  Connect {providerLabel(providerId)}
                </Button>
              </>
            )}
          </div>
        </div>
        {scopes.length > 0 && !isConnected && (
          <div className="mt-2 pl-8">
            <p className="text-muted-foreground text-[11px]">
              Permissions requested: {scopes.join(", ")}
            </p>
          </div>
        )}
        {!isConnected && !needsReauth && !provider?.has_credentials && (
          <p className="text-muted-foreground mt-2 text-xs">
            OAuth is not available yet — ask your admin to configure{" "}
            {providerLabel(providerId)} OAuth credentials.
          </p>
        )}
      </div>

      {SHOP_REQUIRED_PROVIDERS.has(providerId) && (
        <ShopDomainDialog
          open={shopDialogOpen}
          onOpenChange={setShopDialogOpen}
          providerId={providerId}
        />
      )}
    </>
  );
}

function ShopDomainDialog({
  open,
  onOpenChange,
  providerId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  providerId: string;
}) {
  const { session } = useAuth();
  const [shop, setShop] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!session?.access_token) return;
    const trimmed = shop.trim().toLowerCase();
    if (!trimmed) return;
    const subdomain = trimmed.replace(/\.myshopify\.com$/, "");
    const url = getOAuthAuthorizeUrl(providerId, session.access_token);
    window.location.href = `${url}&shop=${encodeURIComponent(subdomain)}`;
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Connect {providerLabel(providerId)} Store</DialogTitle>
          <DialogDescription>
            Enter your {providerLabel(providerId)} store subdomain to begin the
            OAuth connection.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-2 py-4">
            <Label htmlFor="shop-domain">Store subdomain</Label>
            <div className="flex items-center gap-2">
              <Input
                id="shop-domain"
                placeholder="mystore"
                value={shop}
                onChange={(e) => setShop(e.target.value)}
                autoFocus
              />
              <span className="text-muted-foreground whitespace-nowrap text-sm">
                .myshopify.com
              </span>
            </div>
            <p className="text-muted-foreground text-xs">
              e.g. if your store URL is mystore.myshopify.com, enter
              &quot;mystore&quot;
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
            <Button type="submit" disabled={!shop.trim()}>
              <LogIn className="size-4" />
              Continue to {providerLabel(providerId)}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function StaticCredentialRow({
  requiredCredential,
  storedCredentials,
  isAlternative,
}: {
  requiredCredential: RequiredCredential;
  storedCredentials: CredentialSummary[];
  isAlternative: boolean;
}) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<CredentialSummary | null>(
    null,
  );

  const isConnected = storedCredentials.length > 0;

  return (
    <>
      <div className="rounded-lg border p-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div>
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium">
                  {serviceLabel(requiredCredential.service)}
                </p>
                {isAlternative && (
                  <Badge variant="outline" className="text-xs">
                    Alternative
                  </Badge>
                )}
              </div>
              <p className="text-muted-foreground text-xs">
                {authTypeLabel(requiredCredential.auth_type)}
              </p>
              {requiredCredential.instructions_url && (
                <a
                  href={requiredCredential.instructions_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground mt-0.5 inline-flex items-center gap-1 text-xs"
                >
                  <ExternalLink className="size-3" />
                  How to get this credential
                </a>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`text-xs font-medium ${
                isConnected
                  ? "text-green-600 dark:text-green-400"
                  : "text-muted-foreground"
              }`}
            >
              {isConnected ? "Connected" : "Not configured"}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAddDialogOpen(true)}
            >
              <Plus className="size-3" />
              {isConnected ? "Add Another" : "Connect"}
            </Button>
          </div>
        </div>

        {storedCredentials.length > 0 && (
          <div className="mt-3 space-y-2 border-t pt-3">
            {storedCredentials.map((cred) => (
              <div
                key={cred.id}
                className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm">
                    {cred.label ?? cred.service}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    Added {new Date(cred.created_at).toLocaleDateString()}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => setRemoveTarget(cred)}
                  aria-label={`Remove credential ${cred.label ?? cred.service}`}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      <AddCredentialDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        credential={requiredCredential}
      />

      {removeTarget && (
        <RemoveCredentialDialog
          open={!!removeTarget}
          onOpenChange={(open) => {
            if (!open) setRemoveTarget(null);
          }}
          credential={removeTarget}
        />
      )}
    </>
  );
}
