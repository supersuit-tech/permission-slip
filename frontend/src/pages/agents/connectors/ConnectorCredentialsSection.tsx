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
import {
  providerLabel,
  getOAuthAuthorizeUrl,
  SHOP_REQUIRED_PROVIDERS,
} from "@/lib/oauth";
import {
  useAgentConnectorCredential,
  useAssignAgentConnectorCredential,
} from "@/hooks/useAgentConnectorCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { serviceLabel, authTypeLabel } from "@/lib/labels";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";
import { DisconnectOAuthDialog } from "./DisconnectOAuthDialog";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";

export interface ConnectorCredentialsSectionProps {
  agentId: number;
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  agentId,
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
        <AgentCredentialBinding
          agentId={agentId}
          connectorId={connectorId}
          credentials={credentials}
          connections={connections}
          anyLoading={anyLoading}
        />

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
  connections: OAuthConnection[];
  providers: { id: string; has_credentials: boolean }[];
}) {
  const { session } = useAuth();
  const [shopDialogState, setShopDialogState] = useState<{
    replaceId?: string;
  } | null>(null);
  const [disconnectTarget, setDisconnectTarget] = useState<{
    id: string;
    displayName?: string;
  } | null>(null);

  const providerId = requiredCredential.oauth_provider ?? "";
  const providerConnections = connections.filter(
    (c) => c.provider === providerId,
  );
  const activeConnections = providerConnections.filter(
    (c) => c.status === "active",
  );
  const needsReauthConnection = providerConnections.find(
    (c) => c.status === "needs_reauth",
  );
  const provider = providers.find((p) => p.id === providerId);
  const hasAnyConnection = providerConnections.length > 0;
  const isConnected = activeConnections.length > 0;

  function handleConnect() {
    if (!session?.access_token || !providerId) return;
    if (SHOP_REQUIRED_PROVIDERS.has(providerId)) {
      setShopDialogState({});
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
      { scopes: requiredCredential.oauth_scopes },
    );
  }

  function handleReconnect(connectionId: string) {
    if (!session?.access_token || !providerId) return;
    if (SHOP_REQUIRED_PROVIDERS.has(providerId)) {
      setShopDialogState({ replaceId: connectionId });
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      providerId,
      session.access_token,
      { scopes: requiredCredential.oauth_scopes, replaceId: connectionId },
    );
  }

  return (
    <>
      <div className="rounded-lg border p-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : needsReauthConnection ? (
              <AlertTriangle className="size-5 shrink-0 text-amber-500" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
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
                  ? `${activeConnections.length} account${activeConnections.length !== 1 ? "s" : ""} connected`
                  : needsReauthConnection
                    ? "Re-authorization required"
                    : "Not connected"}
              </p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {!hasAnyConnection && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleConnect}
                disabled={!provider?.has_credentials}
              >
                <LogIn className="size-3" />
                Connect {providerLabel(providerId)}
              </Button>
            )}
            {hasAnyConnection && provider?.has_credentials && (
              <Button variant="outline" size="sm" onClick={handleConnect}>
                <Plus className="size-3" />
                Add account
              </Button>
            )}
          </div>
        </div>

        {/* List each connection */}
        {providerConnections.length > 0 && (
          <div className="mt-3 space-y-2 border-t pt-3">
            {providerConnections.map((conn) => (
              <div
                key={conn.id}
                className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm">
                    {conn.display_name || providerLabel(conn.provider)}
                    {conn.instance && !conn.display_name && (
                      <span className="text-muted-foreground ml-1">
                        ({conn.instance})
                      </span>
                    )}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {conn.status === "active"
                      ? `Connected ${new Date(conn.connected_at).toLocaleDateString()}`
                      : conn.status === "needs_reauth"
                        ? "Needs re-authorization"
                        : conn.status}
                  </p>
                </div>
                <div className="flex items-center gap-1">
                  {conn.status === "needs_reauth" ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleReconnect(conn.id)}
                    >
                      <LogIn className="size-3" />
                      Re-authorize
                    </Button>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleReconnect(conn.id)}
                      aria-label="Reconnect"
                    >
                      <LogIn className="size-3" />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive hover:text-destructive"
                    onClick={() =>
                      setDisconnectTarget({
                        id: conn.id,
                        displayName: conn.display_name,
                      })
                    }
                    aria-label={`Disconnect ${conn.display_name ?? providerLabel(conn.provider)}`}
                  >
                    <Unplug className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        {!hasAnyConnection && !provider?.has_credentials && (
          <p className="text-muted-foreground mt-2 pl-8 text-xs">
            OAuth is not available yet — ask your admin to configure{" "}
            {providerLabel(providerId)} OAuth credentials.
          </p>
        )}
      </div>

      {SHOP_REQUIRED_PROVIDERS.has(providerId) && shopDialogState && (
        <ShopDomainDialog
          open
          onOpenChange={(open) => {
            if (!open) setShopDialogState(null);
          }}
          providerId={providerId}
          oauthScopes={requiredCredential.oauth_scopes}
          replaceId={shopDialogState.replaceId}
        />
      )}

      {disconnectTarget && (
        <DisconnectOAuthDialog
          open
          onOpenChange={(open) => {
            if (!open) setDisconnectTarget(null);
          }}
          connectionId={disconnectTarget.id}
          providerName={providerLabel(providerId)}
          displayName={disconnectTarget.displayName}
        />
      )}
    </>
  );
}

function ShopDomainDialog({
  open,
  onOpenChange,
  providerId,
  oauthScopes,
  replaceId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  providerId: string;
  oauthScopes?: string[];
  replaceId?: string;
}) {
  const { session } = useAuth();
  const [shop, setShop] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!session?.access_token) return;
    const trimmed = shop.trim().toLowerCase();
    if (!trimmed) return;
    const subdomain = trimmed.replace(/\.myshopify\.com$/, "");
    const url = getOAuthAuthorizeUrl(providerId, session.access_token, {
      scopes: oauthScopes,
      replaceId,
    });
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
  const [removeTarget, setRemoveTarget] = useState<CredentialSummary | null>(null);

  const isConnected = storedCredentials.length > 0;

  return (
    <>
      <div className="rounded-lg border p-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
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
          <div className="flex shrink-0 items-center gap-2">
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
          open
          onOpenChange={() => setRemoveTarget(null)}
          credential={removeTarget}
        />
      )}
    </>
  );
}

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

function AgentCredentialBinding({
  agentId,
  connectorId,
  credentials,
  connections,
  anyLoading,
}: {
  agentId: number;
  connectorId: string;
  credentials: CredentialSummary[];
  connections: OAuthConnection[];
  anyLoading: boolean;
}) {
  const { binding, isLoading: bindingLoading } =
    useAgentConnectorCredential(agentId, connectorId);
  const { assign, isPending: assigning } =
    useAssignAgentConnectorCredential();

  const isLoading = anyLoading || bindingLoading;
  const isPending = assigning;

  // Build options scoped to this connector — only show credentials whose
  // service matches the connector ID and OAuth connections whose provider
  // matches, so e.g. Slack creds don't appear in a Google connector dropdown.
  const scopedCredentials = credentials.filter(
    (c) => c.service === connectorId,
  );
  const activeConnections = connections.filter(
    (c) => c.status === "active" && c.id && c.provider === connectorId,
  );

  const currentValue = binding?.credential_id
    ? `cred:${binding.credential_id}`
    : binding?.oauth_connection_id
      ? `oauth:${binding.oauth_connection_id}`
      : "";

  async function handleChange(value: string) {
    if (isPending || !value) return;

    try {
      if (value.startsWith("oauth:")) {
        await assign({
          agentId,
          connectorId,
          oauthConnectionId: value.slice("oauth:".length),
        });
        toast.success("OAuth connection assigned to this agent.");
      } else if (value.startsWith("cred:")) {
        await assign({
          agentId,
          connectorId,
          credentialId: value.slice("cred:".length),
        });
        toast.success("Credential assigned to this agent.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update credential.",
      );
    }
  }

  if (isLoading) return null;

  // Don't show if there are no scoped credentials, connections, or existing binding.
  // A stale binding (pointing to a deleted credential) must still be visible
  // so the user can reassign it.
  if (scopedCredentials.length === 0 && activeConnections.length === 0 && !binding) return null;

  return (
    <div className="mb-4 rounded-lg border p-3">
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Label htmlFor="agent-credential-select" className="text-sm font-medium">
            Agent Credential
          </Label>
          {currentValue ? (
            <Badge
              variant="secondary"
              className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
            >
              Assigned
            </Badge>
          ) : (
            <Badge variant="destructive">
              Not set
            </Badge>
          )}
        </div>
        <p className="text-muted-foreground text-xs">
          {currentValue
            ? "This agent uses a specific credential for this connector."
            : "Select a credential for this agent. The connector won\u2019t work until one is assigned."}
        </p>
        <select
          id="agent-credential-select"
          className={selectClassName}
          value={currentValue}
          onChange={(e) => handleChange(e.target.value)}
          disabled={isPending}
        >
          {!currentValue && <option value="">Select a credential…</option>}
          {activeConnections.map((conn) => (
            <option key={`oauth:${conn.id}`} value={`oauth:${conn.id}`}>
              {providerLabel(conn.provider)} OAuth
              {conn.display_name ? ` — ${conn.display_name}` : ""} (connected{" "}
              {new Date(conn.connected_at).toLocaleDateString()})
            </option>
          ))}
          {scopedCredentials.map((cred) => (
            <option key={`cred:${cred.id}`} value={`cred:${cred.id}`}>
              {cred.label ?? cred.service} (added{" "}
              {new Date(cred.created_at).toLocaleDateString()})
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
