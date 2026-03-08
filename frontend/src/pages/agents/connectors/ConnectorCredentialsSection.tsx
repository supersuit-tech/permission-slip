import { useState } from "react";
import { Link } from "react-router-dom";
import {
  AlertTriangle,
  CheckCircle2,
  Circle,
  ExternalLink,
  Loader2,
  LogIn,
  Plus,
  Trash2,
} from "lucide-react";
import { useAuth } from "@/auth/AuthContext";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
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
  const hasStaticCredentials = requiredCredentials.some(
    (c) => c.auth_type !== "oauth2",
  );
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasStaticCredentials,
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

  const oauthByProvider = new Map(connections.map((c) => [c.provider, c]));

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
        ) : isLoading || oauthLoading ? (
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
            {requiredCredentials.map((cred) =>
              cred.auth_type === "oauth2" ? (
                <OAuthCredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  connection={oauthByProvider.get(
                    cred.oauth_provider ?? cred.service,
                  )}
                />
              ) : (
                <CredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  storedCredentials={
                    storedByService.get(cred.service) ?? []
                  }
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
  connection,
}: {
  requiredCredential: RequiredCredential;
  connection?: { provider: string; status: string; connected_at: string };
}) {
  const { session } = useAuth();
  const isConnected = connection?.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token) return;
    const providerId = requiredCredential.oauth_provider ?? requiredCredential.service;
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${providerId}/authorize`;
    window.location.href = `${url}?access_token=${encodeURIComponent(session.access_token)}`;
  }

  const scopes = requiredCredential.oauth_scopes ?? [];

  return (
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
                {requiredCredential.service}
              </p>
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                OAuth
              </Badge>
              <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                Recommended
              </Badge>
            </div>
            <p className="text-muted-foreground text-xs">
              {isConnected
                ? `Connected ${new Date(connection.connected_at).toLocaleDateString()}`
                : needsReauth
                  ? "Connection expired or was revoked \u2014 re-authorize to restore access"
                  : "Connect via OAuth for automatic token management"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isConnected ? (
            <>
              <span className="text-xs font-medium text-green-600 dark:text-green-400">
                Connected
              </span>
              <Link to="/settings">
                <Button variant="ghost" size="sm">
                  Manage
                </Button>
              </Link>
            </>
          ) : needsReauth ? (
            <Button variant="outline" size="sm" onClick={handleConnect}>
              <LogIn className="size-3" />
              Re-authorize
            </Button>
          ) : (
            <Button variant="outline" size="sm" onClick={handleConnect}>
              <LogIn className="size-3" />
              Connect
            </Button>
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
              <p className="text-sm font-medium">{requiredCredential.service}</p>
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
