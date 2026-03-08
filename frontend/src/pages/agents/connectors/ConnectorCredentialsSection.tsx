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
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { providerLabel, getOAuthAuthorizeUrl } from "@/lib/oauth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

const AUTH_TYPE_LABELS: Record<string, string> = {
  api_key: "API Key",
  basic: "Username & Password",
  custom: "Custom Credential",
};

function authTypeLabel(authType: string): string {
  return AUTH_TYPE_LABELS[authType] ?? authType;
}

interface ConnectorCredentialsSectionProps {
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasRequiredCredentials,
  });

  const hasOAuth = requiredCredentials.some((c) => c.auth_type === "oauth2");
  const { connections, isLoading: connectionsLoading } = useOAuthConnections({ enabled: hasOAuth });
  const { providers, isLoading: providersLoading } = useOAuthProviders({ enabled: hasOAuth });

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  // Sort credentials: OAuth first, then static
  const sorted = [...requiredCredentials].sort((a, b) => {
    if (a.auth_type === "oauth2" && b.auth_type !== "oauth2") return -1;
    if (a.auth_type !== "oauth2" && b.auth_type === "oauth2") return 1;
    return 0;
  });

  const anyLoading =
    isLoading || (hasOAuth && (connectionsLoading || providersLoading));

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
        ) : anyLoading ? (
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
            {sorted.map((cred, idx) => {
              const prevCred =
                idx > 0 ? sorted[idx - 1] : undefined;
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
                      storedCredentials={storedByService.get(cred.service) ?? []}
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
  connections: { provider: string; status: string; scopes: string[]; connected_at: string }[];
  providers: { id: string; has_credentials: boolean }[];
}) {
  const { session } = useAuth();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();

  const providerId = requiredCredential.oauth_provider ?? "";
  const connection = connections.find((c) => c.provider === providerId);
  const provider = providers.find((p) => p.id === providerId);
  const isConnected = connection?.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token) return;
    window.location.href = getOAuthAuthorizeUrl(providerId, session.access_token);
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

  return (
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
                OAuth
              </p>
              <Badge variant="secondary" className="text-xs">
                Recommended
              </Badge>
            </div>
            <p className="text-muted-foreground text-xs">
              Connect your {providerLabel(providerId)} account via OAuth for
              automatic token management
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
                <Button variant="ghost" size="icon" aria-label="Disconnect OAuth">
                  <Unplug className="text-muted-foreground size-4" />
                </Button>
              </InlineConfirmButton>
            </>
          ) : needsReauth ? (
            <>
              <Badge variant="destructive" className="gap-1 text-xs">
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
                Connect
              </Button>
            </>
          )}
        </div>
      </div>
      {!isConnected && !needsReauth && !provider?.has_credentials && (
        <p className="text-muted-foreground mt-2 text-xs">
          OAuth is not available yet — ask your admin to configure{" "}
          {providerLabel(providerId)} OAuth credentials.
        </p>
      )}
    </div>
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
                <p className="text-sm font-medium">{requiredCredential.service}</p>
                {isAlternative && (
                  <Badge variant="outline" className="text-xs">
                    Alternative
                  </Badge>
                )}
              </div>
              <p className="text-muted-foreground text-xs">
                {isAlternative
                  ? "Use a private app access token instead of OAuth"
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
