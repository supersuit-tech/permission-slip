import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
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

/** Providers that require a shop subdomain for per-shop OAuth URLs. */
const SHOP_REQUIRED_PROVIDERS = new Set(["shopify"]);

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
  const { connections, isLoading, error, refetch } = useOAuthConnections();
  const { providers } = useOAuthProviders();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const [searchParams, setSearchParams] = useSearchParams();

  // Handle OAuth callback status from redirect
  useEffect(() => {
    const oauthStatus = searchParams.get("oauth_status");
    const oauthProvider = searchParams.get("oauth_provider");
    if (oauthStatus) {
      if (oauthStatus === "success") {
        toast.success(
          `Successfully connected ${oauthProvider ? providerLabel(oauthProvider) : "account"}.`,
        );
        refetch();
      } else {
        const oauthError = searchParams.get("oauth_error");
        const label = oauthProvider
          ? providerLabel(oauthProvider)
          : "account";
        const detail = oauthError
          ? `Failed to connect ${label}: ${oauthError}`
          : `Failed to connect ${label}. Please try again.`;
        toast.error(detail);
      }
      // Remove query params without a full navigation
      searchParams.delete("oauth_status");
      searchParams.delete("oauth_provider");
      searchParams.delete("oauth_error");
      searchParams.delete("oauth_tab");
      setSearchParams(searchParams, { replace: true });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- run once on mount

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

  const [shopDialogProvider, setShopDialogProvider] = useState<string | null>(
    null,
  );

  function handleConnect(providerId: string) {
    if (!session?.access_token) return;
    if (SHOP_REQUIRED_PROVIDERS.has(providerId)) {
      setShopDialogProvider(providerId);
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

      {shopDialogProvider && session?.access_token && (
        <ShopDomainDialog
          open
          onOpenChange={(open) => {
            if (!open) setShopDialogProvider(null);
          }}
          onSubmit={(shop) => {
            if (!session?.access_token || !shopDialogProvider) return;
            const url = getOAuthAuthorizeUrl(
              shopDialogProvider,
              session.access_token,
            );
            window.location.href = `${url}&shop=${encodeURIComponent(shop)}`;
            setShopDialogProvider(null);
          }}
        />
      )}
    </Card>
  );
}

function ShopDomainDialog({
  open,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (shop: string) => void;
}) {
  const [shop, setShop] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = shop.trim().toLowerCase();
    if (!trimmed) return;
    const subdomain = trimmed.replace(/\.myshopify\.com$/, "");
    onSubmit(subdomain);
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
            <Label htmlFor="settings-shop-domain">Store subdomain</Label>
            <div className="flex items-center gap-2">
              <Input
                id="settings-shop-domain"
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
