import { useMemo, useState } from "react";
import { Loader2, Plus, Settings, Star, Trash2, Unplug } from "lucide-react";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import { providerLabel } from "@/lib/oauth";
import { serviceLabel } from "@/lib/labels";
import {
  useAgentConnectorInstances,
  useCreateAgentConnectorInstance,
  useDeleteAgentConnectorInstance,
  useSetDefaultAgentConnectorInstance,
  type AgentConnectorInstance,
} from "@/hooks/useAgentConnectorInstances";
import {
  useAgentConnectorInstanceCredential,
  useAssignAgentConnectorInstanceCredential,
  useRemoveAgentConnectorInstanceCredential,
} from "@/hooks/useAgentConnectorInstanceCredential";
import { useAutoAssignOAuthCredential } from "@/hooks/useAutoAssignOAuthCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";
import { ManageCredentialsDialog } from "./ManageCredentialsDialog";

export interface ConnectorInstancesSectionProps {
  agentId: number;
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

export function ConnectorInstancesSection({
  agentId,
  connectorId,
  requiredCredentials,
}: ConnectorInstancesSectionProps) {
  const credentialServiceIds = useMemo(
    () => new Set([connectorId, ...requiredCredentials.map((rc) => rc.service)]),
    [connectorId, requiredCredentials],
  );
  const [manageDialogOpen, setManageDialogOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);

  useAutoAssignOAuthCredential(agentId, connectorId);

  const { instances, isLoading, error } = useAgentConnectorInstances(
    agentId,
    connectorId,
  );
  const { create, isPending: creating } = useCreateAgentConnectorInstance();

  const hasRequiredCredentials = requiredCredentials.length > 0;
  const hasExplicitOAuth = requiredCredentials.some(
    (c) => c.auth_type === "oauth2",
  );
  const hasStatic = requiredCredentials.some((c) => c.auth_type !== "oauth2");
  const { credentials, isLoading: credsLoading, error: credsError } =
    useCredentials({ enabled: hasStatic });
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

  const anyLoading = credsLoading || connectionsLoading || providersLoading;
  const showManageButton = hasRequiredCredentials || !!matchingProvider;

  async function handleAddInstance() {
    try {
      await create({ agentId, connectorId });
      toast.success("Connector instance added. Assign a credential to name it.");
      setAddOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add instance.");
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Connector instances</CardTitle>
        {hasOAuth && hasStatic && (
          <CardDescription>
            Connect via OAuth (recommended) or use an API key as an
            alternative. Each instance can use its own credential.
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="flex items-center justify-center py-6" role="status">
            <Loader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        )}
        {!isLoading && error && (
          <p className="text-destructive text-sm">{error}</p>
        )}
        {!isLoading && !error && (
          <div className="space-y-4">
            {instances.map((inst) => (
              <InstanceCard
                key={inst.connector_instance_id}
                agentId={agentId}
                connectorId={connectorId}
                instance={inst}
                credentialServiceIds={credentialServiceIds}
                credentials={credentials}
                connections={connections}
                anyLoading={anyLoading}
              />
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full sm:w-auto"
              onClick={() => setAddOpen(true)}
              disabled={creating}
            >
              <Plus className="size-4" />
              Add another
            </Button>
          </div>
        )}

        {showManageButton && (
          <div className="mt-4 flex justify-end">
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

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add connector instance</DialogTitle>
            <DialogDescription>
              Creates another slot for this connector. After you add it, assign a
              credential on that instance&apos;s card in the list (below this dialog)
              so it has a name in approvals and capabilities.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setAddOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleAddInstance()} disabled={creating}>
              {creating && <Loader2 className="size-4 animate-spin" />}
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ManageCredentialsDialog
        open={manageDialogOpen}
        onOpenChange={setManageDialogOpen}
        agentId={agentId}
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
        error={credsError}
        oauthError={oauthError}
      />
    </Card>
  );
}

function InstanceCard({
  agentId,
  connectorId,
  instance,
  credentialServiceIds,
  credentials,
  connections,
  anyLoading,
}: {
  agentId: number;
  connectorId: string;
  instance: AgentConnectorInstance;
  credentialServiceIds: Set<string>;
  credentials: CredentialSummary[];
  connections: OAuthConnection[];
  anyLoading: boolean;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);

  const { setDefault, isPending: settingDefault } =
    useSetDefaultAgentConnectorInstance();
  const { deleteInstance, isPending: deleting } =
    useDeleteAgentConnectorInstance();

  const isBusy = settingDefault || deleting;
  const displayName =
    instance.display?.trim() ||
    "Unnamed instance — assign a credential";

  async function handleMakeDefault() {
    try {
      await setDefault({
        agentId,
        connectorId,
        instanceId: instance.connector_instance_id,
      });
      toast.success("Default instance updated.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update default.",
      );
    }
  }

  async function handleDelete() {
    try {
      await deleteInstance({
        agentId,
        connectorId,
        instanceId: instance.connector_instance_id,
      });
      toast.success("Instance removed.");
      setDeleteOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove instance.");
    }
  }

  return (
    <>
      <div className="rounded-lg border p-4">
        <div className="mb-3 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className="font-medium">{displayName}</p>
              {instance.is_default && (
                <Badge variant="secondary">Default</Badge>
              )}
            </div>
            <p className="text-muted-foreground mt-1 text-xs">
              Added {new Date(instance.enabled_at).toLocaleString()}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {!instance.is_default && (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => void handleMakeDefault()}
                  disabled={isBusy}
                >
                  <Star className="size-3.5" />
                  Make default
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setDeleteOpen(true)}
                  disabled={isBusy}
                >
                  <Trash2 className="size-3.5" />
                  Remove instance
                </Button>
              </>
            )}
          </div>
        </div>

        <InstanceCredentialBinding
          agentId={agentId}
          connectorId={connectorId}
          instanceId={instance.connector_instance_id}
          credentialServiceIds={credentialServiceIds}
          credentials={credentials}
          connections={connections}
          anyLoading={anyLoading}
        />
      </div>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove this instance?</DialogTitle>
            <DialogDescription>
              This removes the <strong>{displayName}</strong> instance and its
              credential binding. Standing approvals scoped to this instance are
              revoked. The default instance cannot be removed here — disable the
              connector type instead.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setDeleteOpen(false)}
              disabled={deleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleDelete()}
              disabled={deleting}
            >
              {deleting && <Loader2 className="mr-2 size-4 animate-spin" />}
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function InstanceCredentialBinding({
  agentId,
  connectorId,
  instanceId,
  credentialServiceIds,
  credentials,
  connections,
  anyLoading,
}: {
  agentId: number;
  connectorId: string;
  instanceId: string;
  credentialServiceIds: Set<string>;
  credentials: CredentialSummary[];
  connections: OAuthConnection[];
  anyLoading: boolean;
}) {
  const { binding, isLoading: bindingLoading } = useAgentConnectorInstanceCredential(
    agentId,
    connectorId,
    instanceId,
  );
  const { assign, isPending: assigning } =
    useAssignAgentConnectorInstanceCredential();
  const { remove, isPending: disconnecting } =
    useRemoveAgentConnectorInstanceCredential();

  const isLoading = anyLoading || bindingLoading;
  const isPending = assigning || disconnecting;

  const scopedCredentials = credentials.filter(
    (c) => credentialServiceIds.has(c.service),
  );
  const activeConnections = connections.filter(
    (c) => c.status === "active" && c.id && c.provider === connectorId,
  );

  const currentValue = binding?.credential_id
    ? `cred:${binding.credential_id}`
    : binding?.oauth_connection_id
      ? `oauth:${binding.oauth_connection_id}`
      : "";

  async function handleDisconnect() {
    if (isPending || !currentValue) return;
    try {
      await remove({ agentId, connectorId, instanceId });
      toast.success("Credential disconnected from this instance.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to disconnect credential.",
      );
    }
  }

  async function handleChange(value: string) {
    if (isPending || !value) return;

    try {
      if (value.startsWith("oauth:")) {
        await assign({
          agentId,
          connectorId,
          instanceId,
          oauthConnectionId: value.slice("oauth:".length),
        });
        toast.success("OAuth connection assigned to this instance.");
      } else if (value.startsWith("cred:")) {
        await assign({
          agentId,
          connectorId,
          instanceId,
          credentialId: value.slice("cred:".length),
        });
        toast.success("Credential assigned to this instance.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update credential.",
      );
    }
  }

  if (isLoading) return null;

  if (
    scopedCredentials.length === 0 &&
    activeConnections.length === 0 &&
    !binding
  ) {
    return null;
  }

  return (
    <div className="rounded-md border border-dashed p-3">
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Label
            htmlFor={`agent-credential-${instanceId}`}
            className="text-sm font-medium"
          >
            Credential
          </Label>
          {currentValue ? (
            <Badge
              variant="secondary"
              className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
            >
              Assigned
            </Badge>
          ) : (
            <Badge variant="destructive">Not set</Badge>
          )}
        </div>
        <p className="text-muted-foreground text-sm">
          {currentValue
            ? "This instance uses the selected credential or OAuth connection."
            : "Select a credential so this instance can run actions."}
        </p>
        {currentValue && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-8 px-2"
            onClick={() => void handleDisconnect()}
            disabled={isPending}
          >
            <Unplug className="size-3.5" />
            Disconnect
          </Button>
        )}
        <select
          id={`agent-credential-${instanceId}`}
          className={selectClassName}
          value={currentValue}
          onChange={(e) => void handleChange(e.target.value)}
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
              {cred.label
                ? `${cred.label} — ${serviceLabel(cred.service)}`
                : serviceLabel(cred.service)}{" "}
              (added {new Date(cred.created_at).toLocaleDateString()})
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
