import { useState } from "react";
import {
  CheckCircle2,
  Circle,
  ExternalLink,
  Key,
  LogIn,
  Plus,
  Unplug,
} from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/auth/AuthContext";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useDisconnectOAuth } from "@/hooks/useDisconnectOAuth";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import { providerLabel } from "@/lib/providerLabels";
import { AddCredentialDialog } from "./AddCredentialDialog";
import { StoredCredentialList } from "./StoredCredentialList";

interface OAuthCredentialRowProps {
  requiredCredential: RequiredCredential;
  connections: {
    provider: string;
    status: string;
    scopes: string[];
    connected_at: string;
  }[];
  storedCredentials: CredentialSummary[];
}

/**
 * OAuthCredentialRow renders an OAuth-first credential with PAT fallback.
 * Shows the OAuth connection status as primary and a "Use personal access
 * token instead" alternative below.
 */
export function OAuthCredentialRow({
  requiredCredential,
  connections,
  storedCredentials,
}: OAuthCredentialRowProps) {
  const { session } = useAuth();
  const { disconnect, isLoading: isDisconnecting } = useDisconnectOAuth();
  const [showPATForm, setShowPATForm] = useState(false);

  const providerId = requiredCredential.oauth_provider ?? "";
  const connection = connections.find((c) => c.provider === providerId);
  const isOAuthConnected = connection?.status === "active";
  const hasPAT = storedCredentials.length > 0;

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
            {isOAuthConnected ? (
              <CheckCircle2 className="size-5 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <Circle className="text-muted-foreground size-5 shrink-0" />
            )}
            <div>
              <p className="text-sm font-medium">
                {providerLabel(providerId)}
              </p>
              <p className="text-muted-foreground text-xs">
                OAuth{" "}
                <span className="text-muted-foreground/70">(recommended)</span>
                {isOAuthConnected && (
                  <>
                    {" · "}
                    {connection.scopes.length} scope
                    {connection.scopes.length !== 1 ? "s" : ""} granted
                    {" · Connected "}
                    {new Date(connection.connected_at).toLocaleDateString()}
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
              {hasPAT ? (
                <CheckCircle2 className="size-4 shrink-0 text-green-600 dark:text-green-400" />
              ) : (
                <Key className="text-muted-foreground size-4" />
              )}
              <div>
                <p className="text-muted-foreground text-xs">
                  Or use a personal access token
                </p>
                {requiredCredential.instructions_url && (
                  <a
                    href={requiredCredential.instructions_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground mt-0.5 inline-flex items-center gap-1 text-xs"
                  >
                    <ExternalLink className="size-3" />
                    How to get a token
                  </a>
                )}
              </div>
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

          <div className="mt-2">
            <StoredCredentialList
              credentials={storedCredentials}
              defaultLabel="Personal access token"
            />
          </div>
        </div>
      </div>

      <AddCredentialDialog
        open={showPATForm}
        onOpenChange={setShowPATForm}
        credential={{
          service: requiredCredential.service,
          auth_type: "custom",
        }}
        title="Add Personal Access Token"
        credentialKey="personal_access_token"
        fieldLabel="Personal Access Token"
        fieldPlaceholder="Paste your personal access token"
      />
    </>
  );
}
