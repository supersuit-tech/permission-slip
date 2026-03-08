import { useState } from "react";
import {
  CheckCircle2,
  Circle,
  ExternalLink,
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
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { providerLabel } from "@/lib/oauthProviders";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

interface ConnectorCredentialsSectionProps {
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasOAuthCredential = requiredCredentials.some(
    (c) => c.auth_type === "oauth2",
  );
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasRequiredCredentials,
  });
  const { connections, isLoading: oauthLoading } = useOAuthConnections();

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  // Build a map of OAuth connections by provider for quick lookup.
  const oauthByProvider = new Map(connections.map((c) => [c.provider, c]));

  // Sort credentials so OAuth appears first (recommended default).
  const sortedCredentials = [...requiredCredentials].sort((a, b) => {
    if (a.auth_type === "oauth2" && b.auth_type !== "oauth2") return -1;
    if (a.auth_type !== "oauth2" && b.auth_type === "oauth2") return 1;
    return 0;
  });

  const loading = isLoading || (hasOAuthCredential && oauthLoading);

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
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : (
          <div className="space-y-3">
            {sortedCredentials.map((cred) =>
              cred.auth_type === "oauth2" ? (
                <OAuthCredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  oauthConnection={oauthByProvider.get(
                    cred.oauth_provider ?? "",
                  )}
                  hasAlternativeAuth={sortedCredentials.length > 1}
                />
              ) : (
                <CredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  storedCredentials={
                    storedByService.get(cred.service) ?? []
                  }
                  isAlternative={hasOAuthCredential}
                />
              ),
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function OAuthCredentialRow({
  requiredCredential,
  oauthConnection,
  hasAlternativeAuth,
}: {
  requiredCredential: RequiredCredential;
  oauthConnection?: { provider: string; status: string; connected_at: string; scopes: string[] };
  hasAlternativeAuth: boolean;
}) {
  const { session } = useAuth();
  const provider = requiredCredential.oauth_provider ?? "";
  const isConnected = oauthConnection?.status === "active";
  const needsReauth = oauthConnection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token) return;
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${provider}/authorize`;
    window.location.href = `${url}?access_token=${encodeURIComponent(session.access_token)}`;
  }

  const label = providerLabel(provider);

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
              <p className="text-sm font-medium">{label} OAuth</p>
              {hasAlternativeAuth && (
                <Badge variant="secondary" className="text-xs">
                  Recommended
                </Badge>
              )}
            </div>
            <p className="text-muted-foreground text-xs">
              Connect your {label} account for automatic token management
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isConnected ? (
            <>
              <span className="text-xs font-medium text-green-600 dark:text-green-400">
                Connected
              </span>
              <Badge variant="outline" className="text-xs">
                {oauthConnection?.scopes.length} scope
                {oauthConnection?.scopes.length !== 1 ? "s" : ""}
              </Badge>
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
              <Button variant="outline" size="sm" onClick={handleConnect}>
                <LogIn className="size-3" />
                Connect {label}
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
                  {requiredCredential.service}
                </p>
                {isAlternative && (
                  <Badge variant="outline" className="text-xs">
                    Alternative
                  </Badge>
                )}
              </div>
              <p className="text-muted-foreground text-xs">
                Auth type: {requiredCredential.auth_type}
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
