import { useState } from "react";
import { Settings } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { providerLabel } from "@/lib/oauth";
import { serviceLabel } from "@/lib/labels";
import {
  useAgentConnectorCredential,
  useAssignAgentConnectorCredential,
} from "@/hooks/useAgentConnectorCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";
import { ManageCredentialsDialog } from "./ManageCredentialsDialog";

export interface ConnectorCredentialsSectionProps {
  agentId: number;
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

export function ConnectorCredentialsSection({
  agentId,
  connectorId,
  requiredCredentials,
}: ConnectorCredentialsSectionProps) {
  const [manageDialogOpen, setManageDialogOpen] = useState(false);

  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasExplicitOAuth = requiredCredentials.some(
    (c) => c.auth_type === "oauth2",
  );
  const hasStatic = requiredCredentials.some((c) => c.auth_type !== "oauth2");
  const { credentials, isLoading, error } = useCredentials({
    enabled: hasStatic,
  });
  const {
    connections,
    isLoading: connectionsLoading,
    error: oauthError,
  } = useOAuthConnections({ enabled: true });
  const { providers, isLoading: providersLoading } = useOAuthProviders({
    enabled: true,
  });

  const storedByService = new Map<string, CredentialSummary[]>();
  for (const cred of credentials) {
    const list = storedByService.get(cred.service) ?? [];
    list.push(cred);
    storedByService.set(cred.service, list);
  }

  const matchingProvider = providers.find((p) => p.id === connectorId);
  const hasImplicitOAuth = !hasExplicitOAuth && !!matchingProvider;
  const hasOAuth = hasExplicitOAuth || hasImplicitOAuth;

  const sorted = [...requiredCredentials].sort((a, b) => {
    if (a.auth_type === "oauth2" && b.auth_type !== "oauth2") return -1;
    if (a.auth_type !== "oauth2" && b.auth_type === "oauth2") return 1;
    return 0;
  });

  const anyLoading =
    isLoading || connectionsLoading || providersLoading;

  const showManageButton = hasRequiredCredentials || !!matchingProvider;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials</CardTitle>
        {hasOAuth && hasStatic && (
          <CardDescription>
            Connect via OAuth (recommended) or use an API key as an
            alternative.
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        <AgentCredentialBinding
          agentId={agentId}
          connectorId={connectorId}
          credentials={credentials}
          connections={connections}
          anyLoading={anyLoading}
        />

        {showManageButton && (
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setManageDialogOpen(true)}
            >
              <Settings className="size-3" />
              Manage credentials
            </Button>
          </div>
        )}
      </CardContent>

      <ManageCredentialsDialog
        open={manageDialogOpen}
        onOpenChange={setManageDialogOpen}
        connectorId={connectorId}
        connectorLabel={serviceLabel(connectorId)}
        hasRequiredCredentials={hasRequiredCredentials}
        hasImplicitOAuth={hasImplicitOAuth}
        hasOAuth={hasOAuth}
        sorted={sorted}
        connections={connections}
        providers={providers}
        storedByService={storedByService}
        matchingProvider={matchingProvider}
        anyLoading={anyLoading}
        error={error}
        oauthError={oauthError}
      />
    </Card>
  );
}

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

function AgentCredentialBinding({
  agentId,
  connectorId,
  credentials,
  connections,
  anyLoading,
}: {
  agentId: number;
  connectorId: string;
  credentials: CredentialSummary[];
  connections: OAuthConnection[];
  anyLoading: boolean;
}) {
  const { binding, isLoading: bindingLoading } =
    useAgentConnectorCredential(agentId, connectorId);
  const { assign, isPending: assigning } =
    useAssignAgentConnectorCredential();

  const isLoading = anyLoading || bindingLoading;
  const isPending = assigning;

  const scopedCredentials = credentials.filter(
    (c) => c.service === connectorId,
  );
  const activeConnections = connections.filter(
    (c) => c.status === "active" && c.id && c.provider === connectorId,
  );

  const currentValue = binding?.credential_id
    ? `cred:${binding.credential_id}`
    : binding?.oauth_connection_id
      ? `oauth:${binding.oauth_connection_id}`
      : "";

  async function handleChange(value: string) {
    if (isPending || !value) return;

    try {
      if (value.startsWith("oauth:")) {
        await assign({
          agentId,
          connectorId,
          oauthConnectionId: value.slice("oauth:".length),
        });
        toast.success("OAuth connection assigned to this agent.");
      } else if (value.startsWith("cred:")) {
        await assign({
          agentId,
          connectorId,
          credentialId: value.slice("cred:".length),
        });
        toast.success("Credential assigned to this agent.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update credential.",
      );
    }
  }

  if (isLoading) return null;

  if (scopedCredentials.length === 0 && activeConnections.length === 0 && !binding) return null;

  return (
    <div className="mb-4 rounded-lg border p-3">
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Label htmlFor="agent-credential-select" className="text-sm font-medium">
            Agent Credential
          </Label>
          {currentValue ? (
            <Badge
              variant="secondary"
              className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
            >
              Assigned
            </Badge>
          ) : (
            <Badge variant="destructive">
              Not set
            </Badge>
          )}
        </div>
        <p className="text-muted-foreground text-xs">
          {currentValue
            ? "This agent uses a specific credential for this connector."
            : "Select a credential for this agent. The connector won\u2019t work until one is assigned."}
        </p>
        <select
          id="agent-credential-select"
          className={selectClassName}
          value={currentValue}
          onChange={(e) => handleChange(e.target.value)}
          disabled={isPending}
        >
          {!currentValue && <option value="">Select a credential…</option>}
          {activeConnections.map((conn) => (
            <option key={`oauth:${conn.id}`} value={`oauth:${conn.id}`}>
              {providerLabel(conn.provider)} OAuth
              {conn.display_name ? ` — ${conn.display_name}` : ""} (connected{" "}
              {new Date(conn.connected_at).toLocaleDateString()})
            </option>
          ))}
          {scopedCredentials.map((cred) => (
            <option key={`cred:${cred.id}`} value={`cred:${cred.id}`}>
              {serviceLabel(cred.service)}
              {cred.label ? ` — ${cred.label}` : ""} (added{" "}
              {new Date(cred.created_at).toLocaleDateString()})
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
