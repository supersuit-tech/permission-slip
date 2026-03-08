import {
  CheckCircle2,
  Circle,
  LogIn,
  Unplug,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/auth/AuthContext";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { providerLabel, authTypeLabel } from "@/lib/providerLabels";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";

export function OAuthCredentialRow({
  requiredCredential,
  connection,
  recommended,
}: {
  requiredCredential: RequiredCredential;
  connection?: OAuthConnection;
  recommended?: boolean;
}) {
  const { session } = useAuth();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const isConnected = !!connection && connection.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token || !requiredCredential.oauth_provider) return;
    const provider = requiredCredential.oauth_provider;
    // Validate provider ID contains only safe characters (alphanumeric, hyphens, underscores)
    if (!/^[a-zA-Z0-9_-]+$/.test(provider)) return;
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${provider}/authorize`;
    window.location.href = `${url}?access_token=${encodeURIComponent(session.access_token)}`;
  }

  async function handleDisconnect() {
    if (!requiredCredential.oauth_provider) return;
    await disconnect(requiredCredential.oauth_provider);
  }

  const label = providerLabel(requiredCredential.oauth_provider ?? requiredCredential.service);

  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {isConnected ? (
            <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
          ) : needsReauth ? (
            <Circle className="size-5 shrink-0 text-yellow-500" />
          ) : (
            <Circle className="text-muted-foreground size-5 shrink-0" />
          )}
          <div>
            <div className="flex items-center gap-2">
              <p className="text-sm font-medium">{label}</p>
              <Badge variant="secondary" className="text-xs">
                {authTypeLabel(requiredCredential.auth_type)}
              </Badge>
              {recommended && (
                <Badge className="bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 text-xs">
                  Recommended
                </Badge>
              )}
            </div>
            {isConnected && connection && (
              <p className="text-muted-foreground text-xs">
                {connection.scopes.length} scope
                {connection.scopes.length !== 1 ? "s" : ""} granted &middot;
                Connected{" "}
                {new Date(connection.connected_at).toLocaleDateString()}
              </p>
            )}
            {needsReauth && (
              <p className="text-xs text-yellow-600 dark:text-yellow-400">
                Connection expired — please re-authorize
              </p>
            )}
            {!isConnected && !needsReauth && (
              <p className="text-muted-foreground text-xs">
                Connect your {label} account to enable this connector
              </p>
            )}
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
                  aria-label={`Disconnect ${label}`}
                >
                  <Unplug className="text-muted-foreground size-4" />
                </Button>
              </InlineConfirmButton>
            </>
          ) : (
            <>
              <span className="text-muted-foreground text-xs font-medium">
                {needsReauth ? "Needs re-auth" : "Not configured"}
              </span>
              <Button variant="outline" size="sm" onClick={handleConnect}>
                <LogIn className="size-3" />
                {needsReauth ? "Re-authorize" : `Connect ${label}`}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
