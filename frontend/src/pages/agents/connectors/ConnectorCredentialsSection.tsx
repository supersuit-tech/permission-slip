import { useState } from "react";
import {
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
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

interface ConnectorCredentialsSectionProps {
  requiredCredentials: RequiredCredential[];
}

function authTypeLabel(authType: string): string {
  switch (authType) {
    case "api_key":
      return "API Key";
    case "basic":
      return "Username & Password";
    case "oauth2":
      return "OAuth";
    case "custom":
      return "Custom";
    default:
      return authType;
  }
}

function providerLabel(id: string): string {
  const labels: Record<string, string> = {
    google: "Google",
    microsoft: "Microsoft",
    square: "Square",
    zoom: "Zoom",
    salesforce: "Salesforce",
    meta: "Meta",
    linkedin: "LinkedIn",
    kroger: "Kroger",
  };
  return labels[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

/** Map service IDs like "square_api_key" to a friendlier display name. */
function serviceDisplayName(service: string): string {
  // Strip common suffixes to get the base provider name, then titlecase it.
  const base = service.replace(/_(api_key|oauth|token|creds?)$/i, "");
  return providerLabel(base);
}

export function ConnectorCredentialsSection({
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasOAuth = requiredCredentials.some((c) => c.auth_type === "oauth2");
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasRequiredCredentials,
  });
  const {
    connections,
    isLoading: oauthLoading,
    error: oauthError,
  } = useOAuthConnections();

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  const oauthByProvider = new Map(
    connections.map((c) => [c.provider, c]),
  );

  const loading = isLoading || (hasOAuth && oauthLoading);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials</CardTitle>
      </CardHeader>
      <CardContent>
        {!hasRequiredCredentials ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            This connector does not require any credentials.
          </p>
        ) : loading ? (
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
            {requiredCredentials.map((cred, index) => {
              const hasMixedAuth =
                requiredCredentials.length > 1 &&
                requiredCredentials.some((c) => c.auth_type === "oauth2") &&
                requiredCredentials.some((c) => c.auth_type !== "oauth2");
              const showDivider = hasMixedAuth && index > 0;
              return (
                <div key={cred.service}>
                  {showDivider && (
                    <div className="flex items-center gap-3 py-1">
                      <div className="bg-border h-px flex-1" />
                      <span className="text-muted-foreground text-xs font-medium uppercase">
                        or
                      </span>
                      <div className="bg-border h-px flex-1" />
                    </div>
                  )}
                  {cred.auth_type === "oauth2" ? (
                    <OAuthCredentialRow
                      requiredCredential={cred}
                      connection={oauthByProvider.get(
                        cred.oauth_provider ?? "",
                      )}
                      recommended={hasMixedAuth}
                    />
                  ) : (
                    <CredentialRow
                      requiredCredential={cred}
                      storedCredentials={
                        storedByService.get(cred.service) ?? []
                      }
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

interface OAuthConnection {
  provider: string;
  status: string;
  scopes: string[];
  connected_at: string;
}

function OAuthCredentialRow({
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
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${requiredCredential.oauth_provider}/authorize`;
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

function CredentialRow({
  requiredCredential,
  storedCredentials,
}: {
  requiredCredential: RequiredCredential;
  storedCredentials: CredentialSummary[];
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
                <p className="text-sm font-medium">{serviceDisplayName(requiredCredential.service)}</p>
                <Badge variant="outline" className="text-xs">
                  {authTypeLabel(requiredCredential.auth_type)}
                </Badge>
              </div>
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
