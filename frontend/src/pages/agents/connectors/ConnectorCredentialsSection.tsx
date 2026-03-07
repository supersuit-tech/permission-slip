import { useState } from "react";
import {
  CheckCircle2,
  Circle,
  ExternalLink,
  Key,
  Loader2,
  LogIn,
  Plus,
  Trash2,
  Unplug,
} from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { RemoveCredentialDialog } from "./RemoveCredentialDialog";

const PROVIDER_LABELS: Record<string, string> = {
  figma: "Figma",
  google: "Google",
  microsoft: "Microsoft",
  salesforce: "Salesforce",
  zoom: "Zoom",
};

function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? id.charAt(0).toUpperCase() + id.slice(1);
}

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
            {requiredCredentials.map((cred) =>
              cred.auth_type === "oauth2" && cred.oauth_provider ? (
                <OAuthCredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  connections={connections}
                  storedCredentials={storedByService.get(cred.service) ?? []}
                />
              ) : (
                <CredentialRow
                  key={cred.service}
                  requiredCredential={cred}
                  storedCredentials={storedByService.get(cred.service) ?? []}
                />
              ),
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/**
 * OAuthCredentialRow renders an OAuth-first credential with PAT fallback.
 * Shows the OAuth connection status as primary and a "Use personal access
 * token instead" alternative below.
 */
function OAuthCredentialRow({
  requiredCredential,
  connections,
  storedCredentials,
}: {
  requiredCredential: RequiredCredential;
  connections: { provider: string; status: string; scopes: string[]; connected_at: string }[];
  storedCredentials: CredentialSummary[];
}) {
  const { session } = useAuth();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const [showPATForm, setShowPATForm] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<CredentialSummary | null>(
    null,
  );

  const providerId = requiredCredential.oauth_provider ?? "";
  const connection = connections.find((c) => c.provider === providerId);
  const isOAuthConnected = connection?.status === "active";
  const hasPAT = storedCredentials.length > 0;
  const isConnected = isOAuthConnected || hasPAT;

  function handleConnect() {
    if (!session?.access_token) return;
    const baseUrl =
      import.meta.env.VITE_API_BASE_URL?.replace(/\/v1\/?$/, "") ?? "/api";
    const url = `${baseUrl}/v1/oauth/${providerId}/authorize`;
    window.location.href = `${url}?access_token=${encodeURIComponent(session.access_token)}`;
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
    <>
      <div className="rounded-lg border p-3">
        {/* OAuth connection (primary) */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {isConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {providerLabel(providerId)}
              </p>
              <p className="text-muted-foreground text-xs">
                OAuth
                {isOAuthConnected && (
                  <>
                    {" · "}
                    {connection.scopes.length} scope
                    {connection.scopes.length !== 1 ? "s" : ""} granted
                  </>
                )}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isOAuthConnected ? (
              <>
                <Badge variant="secondary">Connected</Badge>
                <InlineConfirmButton
                  confirmLabel="Disconnect"
                  isProcessing={isDisconnecting}
                  onConfirm={handleDisconnect}
                >
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Disconnect ${providerLabel(providerId)}`}
                  >
                    <Unplug className="text-muted-foreground size-4" />
                  </Button>
                </InlineConfirmButton>
              </>
            ) : connection?.status === "needs_reauth" ? (
              <>
                <Badge variant="destructive" className="gap-1">
                  Needs Re-auth
                </Badge>
                <Button variant="outline" size="sm" onClick={handleConnect}>
                  <LogIn className="size-4" />
                  Re-authorize
                </Button>
              </>
            ) : (
              <Button variant="outline" size="sm" onClick={handleConnect}>
                <LogIn className="size-4" />
                Connect {providerLabel(providerId)}
              </Button>
            )}
          </div>
        </div>

        {/* PAT section (alternative) */}
        <div className="mt-3 border-t pt-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Key className="text-muted-foreground size-4" />
              <p className="text-muted-foreground text-xs">
                Or use a personal access token
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowPATForm(true)}
            >
              <Plus className="size-3" />
              {hasPAT ? "Add Another" : "Add Token"}
            </Button>
          </div>

          {storedCredentials.length > 0 && (
            <div className="mt-2 space-y-2">
              {storedCredentials.map((cred) => (
                <div
                  key={cred.id}
                  className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm">
                      {cred.label ?? "Personal access token"}
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
                    aria-label={`Remove credential ${cred.label ?? "personal access token"}`}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* PAT dialog uses "custom" auth_type so AddCredentialDialog renders the key input */}
      <AddCredentialDialog
        open={showPATForm}
        onOpenChange={setShowPATForm}
        credential={{
          service: requiredCredential.service,
          auth_type: "custom",
        }}
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
