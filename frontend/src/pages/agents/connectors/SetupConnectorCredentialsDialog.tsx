import { useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  KeyRound,
  Loader2,
  LogIn,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { useAuth } from "@/auth/AuthContext";
import { useConnectorDetail } from "@/hooks/useConnectorDetail";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import {
  providerLabel,
  getOAuthAuthorizeUrl,
  SHOP_REQUIRED_PROVIDERS,
} from "@/lib/oauth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { useTryAutoAssign } from "@/hooks/useTryAutoAssign";
import { AddCredentialDialog } from "./AddCredentialDialog";

interface SetupConnectorCredentialsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId?: number;
  connectorId: string;
  connectorName: string;
  connectorLogoSvg?: string;
}

/**
 * Modal shown immediately after enabling a connector, guiding the user through
 * credential setup (OAuth or API key) in a single flow.
 *
 * Handles six states:
 * - Loading: fetching connector detail, OAuth providers, and connections
 * - No credentials required: connector is ready to use
 * - Already connected: OAuth connection is active
 * - Needs reauth: OAuth connection expired or revoked
 * - OAuth available: prominent "Connect with {Provider}" button
 * - OAuth unavailable / static only: API key or basic auth input
 *
 * For connectors that support both OAuth and static credentials, OAuth is
 * shown prominently with a "Use API key instead" link in the footer.
 */
export function SetupConnectorCredentialsDialog({
  open,
  onOpenChange,
  agentId,
  connectorId,
  connectorName,
  connectorLogoSvg,
}: SetupConnectorCredentialsDialogProps) {
  const { session } = useAuth();
  const { connector, isLoading: detailLoading } =
    useConnectorDetail(connectorId);
  const { providers, isLoading: providersLoading } = useOAuthProviders();
  const { connections, isLoading: connectionsLoading } = useOAuthConnections();

  const [shopSubdomain, setShopSubdomain] = useState("");
  const [addCredentialOpen, setAddCredentialOpen] =
    useState(false);
  const [addCredentialTarget, setAddCredentialTarget] =
    useState<RequiredCredential | null>(null);
  const { tryAssign } = useTryAutoAssign(agentId, connectorId);

  const isLoading = detailLoading || providersLoading || connectionsLoading;
  const requiredCredentials = connector?.required_credentials ?? [];

  // Find OAuth credential requirement
  const oauthCredential = requiredCredentials.find(
    (c) => c.auth_type === "oauth2",
  );

  // Check for implicit OAuth (connector has a matching built-in provider but
  // no explicit oauth2 credential in manifest — e.g. Shopify).
  const matchingProvider = providers.find((p) => p.id === connectorId);
  const hasImplicitOAuth = !oauthCredential && !!matchingProvider;

  const effectiveOAuthProvider =
    oauthCredential?.oauth_provider ?? (hasImplicitOAuth ? connectorId : null);
  const provider = effectiveOAuthProvider
    ? providers.find((p) => p.id === effectiveOAuthProvider)
    : null;
  const hasOAuthCredentials = !!provider?.has_credentials;

  // Check if already connected (any active connection for this provider)
  const providerConnections = effectiveOAuthProvider
    ? connections.filter((c) => c.provider === effectiveOAuthProvider)
    : [];
  const isAlreadyConnected = providerConnections.some(
    (c) => c.status === "active",
  );
  const needsReauth =
    !isAlreadyConnected &&
    providerConnections.some((c) => c.status === "needs_reauth");

  // Find static (API key / basic) credential requirements
  const staticCredentials = requiredCredentials.filter(
    (c) => c.auth_type !== "oauth2",
  );

  const hasOAuth = !!effectiveOAuthProvider;
  const hasNoCredentials =
    !isLoading && requiredCredentials.length === 0 && !matchingProvider;
  const needsShopDomain =
    effectiveOAuthProvider != null &&
    SHOP_REQUIRED_PROVIDERS.has(effectiveOAuthProvider);

  function handleOAuthConnect() {
    if (!session?.access_token || !effectiveOAuthProvider) return;

    if (needsShopDomain) {
      const trimmed = shopSubdomain.trim().toLowerCase();
      if (!trimmed) return;
      const subdomain = trimmed.replace(/\.myshopify\.com$/, "");
      const url = getOAuthAuthorizeUrl(
        effectiveOAuthProvider,
        session.access_token,
        oauthCredential?.oauth_scopes,
      );
      window.location.href = `${url}&shop=${encodeURIComponent(subdomain)}`;
      return;
    }

    window.location.href = getOAuthAuthorizeUrl(
      effectiveOAuthProvider,
      session.access_token,
      oauthCredential?.oauth_scopes,
    );
  }

  function handleUseApiKey(credential: RequiredCredential) {
    setAddCredentialTarget(credential);
    setAddCredentialOpen(true);
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <div className="flex items-center gap-3">
              <ConnectorLogo
                name={connectorName}
                logoSvg={connectorLogoSvg}
                size="lg"
              />
              <div>
                <DialogTitle>Set up {connectorName}</DialogTitle>
                <DialogDescription>
                  {hasNoCredentials && !isLoading
                    ? `${connectorName} has been enabled and is ready to use.`
                    : `Connect your ${connectorName} account to start using this connector.`}
                </DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {isLoading ? (
            <div
              className="flex items-center justify-center py-12"
              role="status"
              aria-label="Loading connector details"
            >
              <Loader2
                className="text-muted-foreground size-6 animate-spin"
                aria-hidden="true"
              />
            </div>
          ) : hasNoCredentials ? (
            <NoCredentialsContent onClose={() => onOpenChange(false)} />
          ) : isAlreadyConnected ? (
            <AlreadyConnectedContent
              providerName={providerLabel(effectiveOAuthProvider ?? connectorId)}
              onClose={() => onOpenChange(false)}
            />
          ) : needsReauth && hasOAuth ? (
            <NeedsReauthContent
              providerName={providerLabel(effectiveOAuthProvider ?? connectorId)}
              onReauthorize={handleOAuthConnect}
            />
          ) : hasOAuth && hasOAuthCredentials ? (
            <OAuthSetupContent
              providerName={providerLabel(
                effectiveOAuthProvider ?? connectorId,
              )}
              needsShopDomain={needsShopDomain}
              shopSubdomain={shopSubdomain}
              onShopSubdomainChange={setShopSubdomain}
              onConnect={handleOAuthConnect}
            />
          ) : hasOAuth && !hasOAuthCredentials ? (
            <OAuthUnavailableContent
              providerName={providerLabel(
                effectiveOAuthProvider ?? connectorId,
              )}
            />
          ) : null}

          {/* Static credentials shown when there's no OAuth, or OAuth is unavailable */}
          {!isLoading &&
            !hasNoCredentials &&
            !isAlreadyConnected &&
            staticCredentials.length > 0 &&
            (!hasOAuth || !hasOAuthCredentials) && (
              <StaticOnlyContent
                credentials={staticCredentials}
                onConnect={handleUseApiKey}
              />
            )}

          <DialogFooter className="flex flex-row items-center sm:justify-between">
            <div>
              {!isLoading &&
                hasOAuth &&
                hasOAuthCredentials &&
                !isAlreadyConnected &&
                staticCredentials.length > 0 && (
                  <Button
                    variant="link"
                    size="sm"
                    className="text-muted-foreground px-0"
                    onClick={() => handleUseApiKey(staticCredentials[0]!)}
                  >
                    <KeyRound className="size-3" />
                    Use API key instead
                  </Button>
                )}
            </div>
            <Button variant="ghost" onClick={() => onOpenChange(false)}>
              {hasNoCredentials || isAlreadyConnected
                ? "Done"
                : needsReauth
                  ? "Skip for now"
                  : "Set up later"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {addCredentialTarget && (
        <AddCredentialDialog
          open={addCredentialOpen}
          onOpenChange={(nextOpen) => {
            setAddCredentialOpen(nextOpen);
            if (!nextOpen) {
              setAddCredentialTarget(null);
            }
          }}
          credential={addCredentialTarget}
          onSuccess={(credentialId) => {
            tryAssign({ credentialId });
            onOpenChange(false);
          }}
        />
      )}
    </>
  );
}

function NoCredentialsContent({ onClose }: { onClose: () => void }) {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <CheckCircle2 className="size-10 text-green-600 dark:text-green-400" />
      <p className="text-sm font-medium">
        No additional setup needed
      </p>
      <p className="text-muted-foreground max-w-xs text-sm">
        This connector is ready to use — no credentials required.
      </p>
      <Button className="mt-2" onClick={onClose}>
        Done
      </Button>
    </div>
  );
}

function AlreadyConnectedContent({
  providerName,
  onClose,
}: {
  providerName: string;
  onClose: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <CheckCircle2 className="size-10 text-green-600 dark:text-green-400" />
      <p className="text-sm font-medium">
        Already connected to {providerName}
      </p>
      <p className="text-muted-foreground max-w-xs text-sm">
        Your {providerName} account is already connected. This connector is
        ready to use.
      </p>
      <Button className="mt-2" onClick={onClose}>
        Done
      </Button>
    </div>
  );
}

function NeedsReauthContent({
  providerName,
  onReauthorize,
}: {
  providerName: string;
  onReauthorize: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <AlertTriangle className="size-10 text-amber-500" />
      <p className="text-sm font-medium">
        {providerName} connection expired
      </p>
      <p className="text-muted-foreground max-w-xs text-sm">
        Your {providerName} connection has expired or was revoked.
        Re-authorize to restore access.
      </p>
      <Button className="mt-2" onClick={onReauthorize}>
        <LogIn className="size-4" />
        Re-authorize {providerName}
      </Button>
    </div>
  );
}

function OAuthSetupContent({
  providerName,
  needsShopDomain,
  shopSubdomain,
  onShopSubdomainChange,
  onConnect,
}: {
  providerName: string;
  needsShopDomain: boolean;
  shopSubdomain: string;
  onShopSubdomainChange: (value: string) => void;
  onConnect: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-4 py-6 text-center">
      <p className="text-muted-foreground max-w-sm text-sm">
        Sign in with {providerName} to securely connect your account. Your
        tokens are encrypted and automatically refreshed.
      </p>

      {needsShopDomain && (
        <div className="flex w-full max-w-xs items-center gap-2">
          <Input
            placeholder="mystore"
            value={shopSubdomain}
            onChange={(e) => onShopSubdomainChange(e.target.value)}
          />
          <span className="text-muted-foreground whitespace-nowrap text-sm">
            .myshopify.com
          </span>
        </div>
      )}

      <Button
        size="lg"
        className="w-full max-w-xs"
        onClick={onConnect}
        disabled={needsShopDomain && !shopSubdomain.trim()}
      >
        <LogIn className="size-4" />
        Connect with {providerName}
      </Button>

    </div>
  );
}

function OAuthUnavailableContent({
  providerName,
}: {
  providerName: string;
}) {
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <p className="text-muted-foreground max-w-sm text-sm">
        OAuth is not available yet for {providerName}. Ask your admin to
        configure {providerName} OAuth credentials, or use an API key if
        available.
      </p>
    </div>
  );
}

function StaticOnlyContent({
  credentials,
  onConnect,
}: {
  credentials: RequiredCredential[];
  onConnect: (credential: RequiredCredential) => void;
}) {
  return (
    <div className="flex flex-col items-center gap-4 py-6 text-center">
      <p className="text-muted-foreground max-w-sm text-sm">
        Provide your credentials to connect this service. Credentials are
        encrypted at rest and never exposed to agents.
      </p>
      {credentials.map((cred) => (
        <Button
          key={cred.service}
          size="lg"
          className="w-full max-w-xs"
          onClick={() => onConnect(cred)}
        >
          <KeyRound className="size-4" />
          Add{" "}
          {cred.auth_type === "basic"
            ? "credentials"
            : cred.service === "coinbase_agentkit"
              ? "CDP credentials"
              : "API key"}
        </Button>
      ))}
    </div>
  );
}
