import { useState } from "react";
import {
  CheckCircle2,
  Circle,
  ExternalLink,
  KeyRound,
  Loader2,
  LogIn,
  Plus,
  Trash2,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

/** Providers that require a shop subdomain for per-shop OAuth URLs. */
const SHOP_REQUIRED_PROVIDERS = new Set(["shopify"]);

const PROVIDER_LABELS: Record<string, string> = {
  google: "Google",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  shopify: "Shopify",
  zoom: "Zoom",
};

function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

interface ConnectorCredentialsSectionProps {
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  connectorId,
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasRequiredCredentials,
  });
  const { connections } = useOAuthConnections();
  const { providers } = useOAuthProviders();

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  // Check if there's a matching OAuth provider for this connector.
  const matchingProvider = providers.find((p) => p.id === connectorId);
  const oauthConnection = connections.find((c) => c.provider === connectorId);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials</CardTitle>
      </CardHeader>
      <CardContent>
        {!hasRequiredCredentials && !matchingProvider ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            This connector does not require any credentials.
          </p>
        ) : isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2
              className="text-muted-foreground size-5 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : (
          <div className="space-y-3">
            {/* OAuth connection row (shown when a matching provider exists) */}
            {matchingProvider && (
              <OAuthConnectionRow
                connectorId={connectorId}
                connection={oauthConnection ?? null}
                hasCredentials={matchingProvider.has_credentials}
              />
            )}

            {/* Static credential rows */}
            {requiredCredentials.map((cred) => (
              <CredentialRow
                key={cred.service}
                requiredCredential={cred}
                storedCredentials={storedByService.get(cred.service) ?? []}
                isAlternative={!!matchingProvider}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function OAuthConnectionRow({
  connectorId,
  connection,
  hasCredentials,
}: {
  connectorId: string;
  connection: { provider: string; status: string; scopes: string[] } | null;
  hasCredentials: boolean;
}) {
  const { session } = useAuth();
  const [shopDialogOpen, setShopDialogOpen] = useState(false);

  const isConnected = connection?.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (SHOP_REQUIRED_PROVIDERS.has(connectorId)) {
      setShopDialogOpen(true);
      return;
    }
    navigateToOAuth(connectorId, session?.access_token);
  }

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
                  {providerLabel(connectorId)} OAuth
                </p>
                <Badge variant="secondary" className="text-[10px]">
                  Recommended
                </Badge>
              </div>
              <p className="text-muted-foreground text-xs">
                Connect your {providerLabel(connectorId)} account for automatic
                token management
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isConnected && (
              <span className="text-xs font-medium text-green-600 dark:text-green-400">
                Connected
              </span>
            )}
            {needsReauth && (
              <Badge variant="destructive" className="gap-1 text-xs">
                Needs Re-auth
              </Badge>
            )}
            {!isConnected || needsReauth ? (
              <Button
                variant="outline"
                size="sm"
                onClick={handleConnect}
                disabled={!hasCredentials}
                title={
                  !hasCredentials
                    ? "OAuth provider not configured. Set up client credentials in Settings."
                    : undefined
                }
              >
                <LogIn className="size-3" />
                {needsReauth ? "Re-authorize" : "Connect with OAuth"}
              </Button>
            ) : null}
          </div>
        </div>
      </div>

      {SHOP_REQUIRED_PROVIDERS.has(connectorId) && (
        <ShopDomainDialog
          open={shopDialogOpen}
          onOpenChange={setShopDialogOpen}
          connectorId={connectorId}
        />
      )}
    </>
  );
}

function ShopDomainDialog({
  open,
  onOpenChange,
  connectorId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  connectorId: string;
}) {
  const { session } = useAuth();
  const [shop, setShop] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = shop.trim().toLowerCase();
    if (!trimmed) return;
    // Strip .myshopify.com if provided
    const subdomain = trimmed.replace(/\.myshopify\.com$/, "");
    navigateToOAuth(connectorId, session?.access_token, subdomain);
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Connect Shopify Store</DialogTitle>
          <DialogDescription>
            Enter your Shopify store subdomain to begin the OAuth connection.
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
              Continue to Shopify
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function navigateToOAuth(
  providerId: string,
  accessToken: string | undefined,
  shop?: string,
) {
  if (!accessToken) return;
  const baseUrl =
    import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
  let url = `${baseUrl}/v1/oauth/${providerId}/authorize?access_token=${encodeURIComponent(accessToken)}`;
  if (shop) {
    url += `&shop=${encodeURIComponent(shop)}`;
  }
  window.location.href = url;
}

function CredentialRow({
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
                  {isAlternative ? (
                    <>
                      <KeyRound className="mr-1 inline size-3.5" />
                      API Key
                    </>
                  ) : (
                    requiredCredential.service
                  )}
                </p>
                {isAlternative && (
                  <span className="text-muted-foreground text-[10px]">
                    Alternative
                  </span>
                )}
              </div>
              <p className="text-muted-foreground text-xs">
                {isAlternative
                  ? `Use a ${requiredCredential.auth_type === "basic" ? "username and password" : "manually-created API key"} instead`
                  : `Auth type: ${requiredCredential.auth_type}`}
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
                    Added{" "}
                    {new Date(cred.created_at).toLocaleDateString()}
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
