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
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import type { CredentialSummary } from "@/hooks/useCredentials";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";
import {
  providerLabel,
  getOAuthAuthorizeUrl,
  SHOP_REQUIRED_PROVIDERS,
} from "@/lib/oauth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { serviceLabel, authTypeLabel } from "@/lib/labels";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";
import { DisconnectOAuthDialog } from "./DisconnectOAuthDialog";

export interface ManageCredentialsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  connectorId: string;
  hasRequiredCredentials: boolean;
  hasImplicitOAuth: boolean;
  hasOAuth: boolean;
  sorted: RequiredCredential[];
  connections: OAuthConnection[];
  providers: { id: string; has_credentials: boolean }[];
  storedByService: Map<string, CredentialSummary[]>;
  matchingProvider: { id: string; has_credentials: boolean } | undefined;
  anyLoading: boolean;
  error: string | null;
  oauthError: string | null;
}

export function ManageCredentialsDialog({
  open,
  onOpenChange,
  connectorId,
  hasRequiredCredentials,
  hasImplicitOAuth,
  hasOAuth,
  sorted,
  connections,
  providers,
  storedByService,
  matchingProvider,
  anyLoading,
  error,
  oauthError,
}: ManageCredentialsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Manage Credentials</DialogTitle>
          <DialogDescription>
            Connect accounts and manage credentials for this connector.
          </DialogDescription>
        </DialogHeader>

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
      </DialogContent>
    </Dialog>
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
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
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
          <div className="flex shrink-0 items-center gap-2 pl-8 sm:pl-0">
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
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
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
          <div className="flex shrink-0 items-center gap-2 pl-8 sm:pl-0">
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
