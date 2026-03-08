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
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

interface ConnectorCredentialsSectionProps {
  requiredCredentials: RequiredCredential[];
}

/** Human-readable labels for auth types. */
const AUTH_TYPE_LABELS: Record<string, string> = {
  api_key: "API Key",
  basic: "Username & Password",
  custom: "Custom Credentials",
  oauth2: "OAuth",
};

/** Returns a human-readable label for an auth type. */
function authTypeLabel(authType: string): string {
  return AUTH_TYPE_LABELS[authType] ?? authType;
}

/** Capitalize a service name for display (e.g. "pagerduty" → "Pagerduty"). */
function capitalizeService(service: string): string {
  return service.charAt(0).toUpperCase() + service.slice(1);
}

/**
 * Group required credentials by service so that connectors offering multiple
 * auth methods (e.g. PagerDuty with OAuth + API key) are presented as a single
 * group with an "or" separator instead of two independent rows.
 */
function groupByService(
  creds: RequiredCredential[],
): Map<string, RequiredCredential[]> {
  const groups = new Map<string, RequiredCredential[]>();
  for (const c of creds) {
    const list = groups.get(c.service) ?? [];
    list.push(c);
    groups.set(c.service, list);
  }
  return groups;
}

export function ConnectorCredentialsSection({
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasOAuth = requiredCredentials.some((c) => c.auth_type === "oauth2");
  const hasStatic = requiredCredentials.some((c) => c.auth_type !== "oauth2");

  const { credentials, isLoading, error } = useCredentials({
    enabled: hasStatic,
  });
  const { connections, isLoading: oauthLoading } = useOAuthConnections();

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  const oauthByProvider = new Map(
    connections.map((c) => [c.provider, c]),
  );

  const loading = (hasStatic && isLoading) || (hasOAuth && oauthLoading);
  const serviceGroups = groupByService(requiredCredentials);

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
            {[...serviceGroups.entries()].map(([service, creds]) => {
              const first = creds[0];
              return first != null && creds.length === 1 ? (
                // Single auth type — render standalone row.
                first.auth_type === "oauth2" ? (
                  <OAuthCredentialRow
                    key={`${service}-oauth2`}
                    requiredCredential={first}
                    connection={oauthByProvider.get(
                      first.oauth_provider ?? "",
                    )}
                  />
                ) : (
                  <StaticCredentialRow
                    key={`${service}-${first.auth_type}`}
                    requiredCredential={first}
                    storedCredentials={storedByService.get(service) ?? []}
                  />
                )
              ) : (
                // Multiple auth types for the same service — group them.
                <CredentialGroup
                  key={service}
                  service={service}
                  credentials={creds}
                  oauthByProvider={oauthByProvider}
                  storedCredentials={storedByService.get(service) ?? []}
                />
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/** Groups multiple auth options for the same service with an "or" divider. */
function CredentialGroup({
  service,
  credentials,
  oauthByProvider,
  storedCredentials,
}: {
  service: string;
  credentials: RequiredCredential[];
  oauthByProvider: Map<
    string,
    { provider: string; status: string; connected_at: string }
  >;
  storedCredentials: CredentialSummary[];
}) {
  return (
    <div className="rounded-lg border p-3">
      <p className="text-muted-foreground mb-2 text-xs font-medium">
        {capitalizeService(service)} — connect with one of the following:
      </p>
      <div className="space-y-2">
        {credentials.map((cred, idx) => (
          <div key={`${cred.service}-${cred.auth_type}`}>
            {idx > 0 && (
              <div className="relative my-3">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card text-muted-foreground px-2">or</span>
                </div>
              </div>
            )}
            {cred.auth_type === "oauth2" ? (
              <OAuthCredentialRow
                requiredCredential={cred}
                connection={oauthByProvider.get(cred.oauth_provider ?? "")}
                nested
              />
            ) : (
              <StaticCredentialRow
                requiredCredential={cred}
                storedCredentials={storedCredentials}
                nested
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function OAuthCredentialRow({
  requiredCredential,
  connection,
  nested,
}: {
  requiredCredential: RequiredCredential;
  connection?: { provider: string; status: string; connected_at: string };
  nested?: boolean;
}) {
  const { session } = useAuth();
  const isConnected = connection?.status === "active";
  const needsReauth = connection?.status === "needs_reauth";

  function handleConnect() {
    if (!session?.access_token || !requiredCredential.oauth_provider) return;
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${requiredCredential.oauth_provider}/authorize`;
    window.location.href = `${url}?access_token=${encodeURIComponent(session.access_token)}`;
  }

  const providerLabel =
    (requiredCredential.oauth_provider ?? requiredCredential.service)
      .charAt(0)
      .toUpperCase() +
    (requiredCredential.oauth_provider ?? requiredCredential.service).slice(1);

  const wrapperClass = nested ? "" : "rounded-lg border p-3";

  return (
    <div className={wrapperClass}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {isConnected ? (
            <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
          ) : (
            <Circle className="text-muted-foreground size-5 shrink-0" />
          )}
          <div>
            <p className="text-sm font-medium">
              {providerLabel} (OAuth)
            </p>
            <p className="text-muted-foreground text-xs">
              Recommended — automatic token refresh, no manual key management
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`text-xs font-medium ${
              isConnected
                ? "text-green-600 dark:text-green-400"
                : needsReauth
                  ? "text-amber-600 dark:text-amber-400"
                  : "text-muted-foreground"
            }`}
          >
            {isConnected
              ? "Connected"
              : needsReauth
                ? "Needs re-auth"
                : "Not connected"}
          </span>
          {!isConnected && (
            <Button variant="outline" size="sm" onClick={handleConnect}>
              <LogIn className="size-3" />
              {needsReauth ? "Re-authorize" : "Connect"}
            </Button>
          )}
        </div>
      </div>
      {isConnected && connection && (
        <div className="mt-3 border-t pt-3">
          <div className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2">
            <div className="min-w-0">
              <p className="truncate text-sm">{providerLabel} OAuth</p>
              <p className="text-muted-foreground text-xs">
                Connected{" "}
                {new Date(connection.connected_at).toLocaleDateString()}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function StaticCredentialRow({
  requiredCredential,
  storedCredentials,
  nested,
}: {
  requiredCredential: RequiredCredential;
  storedCredentials: CredentialSummary[];
  nested?: boolean;
}) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<CredentialSummary | null>(
    null,
  );

  const isConnected = storedCredentials.length > 0;

  const wrapperClass = nested ? "" : "rounded-lg border p-3";

  return (
    <>
      <div className={wrapperClass}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {capitalizeService(requiredCredential.service)} ({authTypeLabel(requiredCredential.auth_type)})
              </p>
              <p className="text-muted-foreground text-xs">
                Manual setup — you manage the credential directly
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
