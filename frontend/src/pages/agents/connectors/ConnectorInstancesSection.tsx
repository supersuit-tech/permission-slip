import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, Loader2, LogIn, Settings, Star } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { useAuth } from "@/auth/AuthContext";
import { useCredentials } from "@/hooks/useCredentials";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { useOAuthConnections } from "@/hooks/useOAuthConnections";
import { useOAuthProviders } from "@/hooks/useOAuthProviders";
import {
  providerLabel,
  getOAuthAuthorizeUrl,
  SHOP_REQUIRED_PROVIDERS,
} from "@/lib/oauth";
import { serviceLabel } from "@/lib/labels";
import {
  useAgentConnectorInstances,
  useCreateAgentConnectorInstance,
  useDeleteAgentConnectorInstance,
  useSetDefaultAgentConnectorInstance,
  type AgentConnectorInstance,
} from "@/hooks/useAgentConnectorInstances";
import {
  useAssignAgentConnectorInstanceCredential,
  useRemoveAgentConnectorInstanceCredential,
} from "@/hooks/useAgentConnectorInstanceCredential";
import { useConnectorInstanceCredentialBindings } from "@/hooks/useConnectorInstanceCredentialBindings";
import { useAutoAssignOAuthCredential } from "@/hooks/useAutoAssignOAuthCredential";
import type { RequiredCredential } from "@/hooks/useConnectorDetail";
import type { OAuthConnection } from "@/hooks/useOAuthConnections";
import { ManageCredentialsDialog } from "./ManageCredentialsDialog";
import type { InstanceCredentialBinding } from "@/hooks/useConnectorInstanceCredentialBindings";

export interface ConnectorInstancesSectionProps {
  agentId: number;
  connectorId: string;
  requiredCredentials: RequiredCredential[];
}

type CredentialRow =
  | { kind: "oauth"; rowKey: string; connection: OAuthConnection }
  | { kind: "cred"; rowKey: string; credential: CredentialSummary };

function bindingRowKey(b: InstanceCredentialBinding | null | undefined): string | null {
  if (!b) return null;
  if (b.oauth_connection_id) return `oauth:${b.oauth_connection_id}`;
  if (b.credential_id) return `cred:${b.credential_id}`;
  return null;
}

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
  const { session } = useAuth();

  useAutoAssignOAuthCredential(agentId, connectorId);

  const { instances, isLoading, error } = useAgentConnectorInstances(
    agentId,
    connectorId,
  );
  const { create, isPending: creating } = useCreateAgentConnectorInstance();
  const { setDefault, isPending: settingDefault } =
    useSetDefaultAgentConnectorInstance();
  const { deleteInstance, isPending: deleting } =
    useDeleteAgentConnectorInstance();
  const { assign, isPending: assigning } =
    useAssignAgentConnectorInstanceCredential();
  const { remove, isPending: removing } =
    useRemoveAgentConnectorInstanceCredential();

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

  const storedByService = useMemo(() => {
    const m = new Map<string, CredentialSummary[]>();
    for (const cred of credentials) {
      const list = m.get(cred.service) ?? [];
      list.push(cred);
      m.set(cred.service, list);
    }
    return m;
  }, [credentials]);

  const matchingProvider = providers.find((p) => p.id === connectorId);
  const hasImplicitOAuth = !hasExplicitOAuth && !!matchingProvider;
  const hasOAuth = hasExplicitOAuth || hasImplicitOAuth;

  const sorted = [...requiredCredentials].sort((a, b) => {
    if (a.auth_type === "oauth2" && b.auth_type !== "oauth2") return -1;
    if (a.auth_type !== "oauth2" && b.auth_type === "oauth2") return 1;
    return 0;
  });

  const instanceIds = useMemo(
    () => instances.map((i) => i.connector_instance_id),
    [instances],
  );

  const {
    data: bindingByInstance,
    isLoading: bindingsLoading,
    isError: bindingsError,
  } = useConnectorInstanceCredentialBindings(agentId, connectorId, instanceIds);

  const rows: CredentialRow[] = useMemo(() => {
    const usableConnections = connections.filter(
      (c) =>
        (c.status === "active" || c.status === "needs_reauth") &&
        c.id &&
        c.provider === connectorId,
    );
    const scopedCredentials = credentials
      .filter((c) => credentialServiceIds.has(c.service))
      .slice()
      .sort((a, b) => {
        const la = (a.label ?? a.service).toLowerCase();
        const lb = (b.label ?? b.service).toLowerCase();
        return la.localeCompare(lb);
      });

    const oauthRows: CredentialRow[] = usableConnections
      .slice()
      .sort((a, b) =>
        (a.display_name ?? a.id).localeCompare(b.display_name ?? b.id),
      )
      .map((connection) => ({
        kind: "oauth" as const,
        rowKey: `oauth:${connection.id}`,
        connection,
      }));

    const credRows: CredentialRow[] = scopedCredentials.map((credential) => ({
      kind: "cred" as const,
      rowKey: `cred:${credential.id}`,
      credential,
    }));

    return [...oauthRows, ...credRows];
  }, [connections, connectorId, credentials, credentialServiceIds]);

  const oauthScopesByProvider = useMemo(() => {
    const m = new Map<string, string[] | undefined>();
    for (const rc of requiredCredentials) {
      if (rc.auth_type === "oauth2" && rc.oauth_provider) {
        m.set(rc.oauth_provider, rc.oauth_scopes);
      }
    }
    return m;
  }, [requiredCredentials]);

  const reauthCount = useMemo(
    () =>
      rows.filter(
        (r) => r.kind === "oauth" && r.connection.status === "needs_reauth",
      ).length,
    [rows],
  );

  const instanceByRowKey = useMemo(() => {
    const map = new Map<string, AgentConnectorInstance>();
    if (!bindingByInstance) return map;
    for (const inst of instances) {
      const b = bindingByInstance.get(inst.connector_instance_id);
      const key = bindingRowKey(b ?? undefined);
      if (key) map.set(key, inst);
    }
    return map;
  }, [instances, bindingByInstance]);

  const enabledInstanceCount = useMemo(() => {
    if (!bindingByInstance) return 0;
    let n = 0;
    for (const inst of instances) {
      const b = bindingByInstance.get(inst.connector_instance_id);
      if (bindingRowKey(b ?? undefined)) n += 1;
    }
    return n;
  }, [instances, bindingByInstance]);

  const orphanInstanceIds = useMemo(() => {
    if (!bindingByInstance) return [];
    const out: string[] = [];
    for (const inst of instances) {
      const b = bindingByInstance.get(inst.connector_instance_id);
      if (!bindingRowKey(b ?? undefined)) out.push(inst.connector_instance_id);
    }
    return out;
  }, [instances, bindingByInstance]);

  const anyLoading = credsLoading || connectionsLoading || providersLoading;
  const showManageButton = hasRequiredCredentials || !!matchingProvider;
  const busyRow =
    creating || assigning || removing || deleting || settingDefault;

  async function handleSetDefault(instanceId: string) {
    try {
      await setDefault({ agentId, connectorId, instanceId });
      toast.success("Default credential updated.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update default.",
      );
    }
  }

  const cleaningUpRef = useRef(new Set<string>());
  useEffect(() => {
    for (const id of orphanInstanceIds) {
      if (cleaningUpRef.current.has(id)) continue;
      cleaningUpRef.current.add(id);
      deleteInstance({ agentId, connectorId, instanceId: id }).finally(() => {
        cleaningUpRef.current.delete(id);
      });
    }
  }, [orphanInstanceIds, agentId, connectorId, deleteInstance]);

  async function enableRow(row: CredentialRow) {
    const created = await create({ agentId, connectorId });
    const newId = created?.connector_instance_id;
    if (!newId) {
      throw new Error("Server did not return a connector instance id");
    }
    if (row.kind === "oauth") {
      await assign({
        agentId,
        connectorId,
        instanceId: newId,
        oauthConnectionId: row.connection.id,
      });
    } else {
      await assign({
        agentId,
        connectorId,
        instanceId: newId,
        credentialId: row.credential.id,
      });
    }
  }

  /** Returns true if the instance was fully removed; false if user messaging already ran. */
  async function disableRow(inst: AgentConnectorInstance): Promise<boolean> {
    const otherWithCredential = instances.filter((i) => {
      if (i.connector_instance_id === inst.connector_instance_id) return false;
      const b = bindingByInstance?.get(i.connector_instance_id);
      return bindingRowKey(b ?? undefined) !== null;
    });

    let transferredDefaultAway = false;
    if (inst.is_default) {
      if (otherWithCredential.length === 0) {
        toast.error(
          "Enable another credential before disabling the default, or pick a different default first.",
        );
        return false;
      }
      const nextDefault = [...otherWithCredential].sort((a, b) =>
        a.connector_instance_id.localeCompare(b.connector_instance_id),
      )[0];
      if (nextDefault) {
        await setDefault({
          agentId,
          connectorId,
          instanceId: nextDefault.connector_instance_id,
        });
        transferredDefaultAway = true;
      }
    }

    try {
      await remove({ agentId, connectorId, instanceId: inst.connector_instance_id });
      await deleteInstance({ agentId, connectorId, instanceId: inst.connector_instance_id });
      return true;
    } catch (err) {
      if (transferredDefaultAway) {
        try {
          await setDefault({
            agentId,
            connectorId,
            instanceId: inst.connector_instance_id,
          });
        } catch {
          /* best-effort rollback */
        }
        toast.error(
          err instanceof Error
            ? `${err.message} The default was reverted to this credential — try again or change default manually.`
            : "Failed to disable credential. The default was reverted — try again or change default manually.",
        );
      } else {
        toast.error(
          err instanceof Error ? err.message : "Failed to disable credential.",
        );
      }
      return false;
    }
  }

  async function toggleRow(row: CredentialRow, checked: boolean) {
    const inst = instanceByRowKey.get(row.rowKey);
    if (checked && !inst) {
      try {
        await enableRow(row);
        toast.success("Credential enabled for this agent.");
      } catch (err) {
        toast.error(
          err instanceof Error ? err.message : "Failed to enable credential.",
        );
      }
      return;
    }
    if (!checked && inst) {
      const disabledOk = await disableRow(inst);
      if (disabledOk) {
        toast.success("Credential disabled for this agent.");
      }
    }
  }

  function rowLabel(row: CredentialRow): string {
    if (row.kind === "oauth") {
      const conn = row.connection;
      const base = `${providerLabel(conn.provider)} OAuth`;
      return conn.display_name ? `${base} — ${conn.display_name}` : base;
    }
    const cred = row.credential;
    return cred.label
      ? `${cred.label} — ${serviceLabel(cred.service)}`
      : serviceLabel(cred.service);
  }

  function rowMeta(row: CredentialRow): string {
    if (row.kind === "oauth") {
      if (row.connection.status === "needs_reauth") {
        return "Needs re-authorization";
      }
      return `Connected ${new Date(row.connection.connected_at).toLocaleDateString()}`;
    }
    return `Added ${new Date(row.credential.created_at).toLocaleDateString()}`;
  }

  function handleReauthorize(connection: OAuthConnection) {
    if (!session?.access_token) return;
    // Shop-required providers (e.g. shopify) prompt for a shop subdomain —
    // defer to the Manage credentials dialog, which owns that flow.
    if (SHOP_REQUIRED_PROVIDERS.has(connection.provider)) {
      setManageDialogOpen(true);
      return;
    }
    window.location.href = getOAuthAuthorizeUrl(
      connection.provider,
      session.access_token,
      {
        scopes: oauthScopesByProvider.get(connection.provider),
        replaceId: connection.id,
      },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Credentials for this agent</CardTitle>
        {hasOAuth && hasStatic && (
          <CardDescription>
            Choose which stored credentials this agent may use for{" "}
            {serviceLabel(connectorId)}. One credential is the default for
            approvals when no specific instance is selected.
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
            {bindingsLoading && (
              <p className="text-muted-foreground text-sm">Loading credential links…</p>
            )}
            {bindingsError && (
              <p className="text-destructive text-sm">
                Failed to load credential links. Refresh the page to retry.
              </p>
            )}
            {reauthCount > 0 && (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm dark:border-amber-900/50 dark:bg-amber-950/40"
              >
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-500" />
                <p className="text-amber-900 dark:text-amber-200">
                  {reauthCount === 1
                    ? "1 credential needs re-authorization before this agent can use it."
                    : `${reauthCount} credentials need re-authorization before this agent can use them.`}
                </p>
              </div>
            )}
            {rows.length === 0 && (
              <p className="text-muted-foreground text-sm">
                No credentials found for this connector. Add one using Manage
                credentials.
              </p>
            )}
            {rows.map((row) => {
              const inst = instanceByRowKey.get(row.rowKey);
              const enabled = !!inst;
              const isDefault = !!inst?.is_default;
              const showDefaultControls = enabled && enabledInstanceCount > 1;
              const needsReauth =
                row.kind === "oauth" && row.connection.status === "needs_reauth";

              return (
                <div
                  key={row.rowKey}
                  className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="flex min-w-0 flex-1 items-start gap-3">
                    <Checkbox
                      id={`cred-row-${row.rowKey}`}
                      checked={enabled}
                      disabled={busyRow}
                      onCheckedChange={(v) => void toggleRow(row, v === true)}
                      className="mt-1"
                    />
                    <div className="min-w-0 flex-1">
                      <label
                        htmlFor={`cred-row-${row.rowKey}`}
                        className="flex cursor-pointer flex-wrap items-center gap-2 font-medium"
                      >
                        {rowLabel(row)}
                        {needsReauth && (
                          <Badge
                            variant="outline"
                            className="gap-1 border-amber-300 text-amber-700 dark:border-amber-700 dark:text-amber-300"
                          >
                            <AlertTriangle className="size-3" />
                            Needs re-authorization
                          </Badge>
                        )}
                      </label>
                      <p
                        className={`text-xs ${needsReauth ? "text-amber-700 dark:text-amber-300" : "text-muted-foreground"}`}
                      >
                        {rowMeta(row)}
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 sm:justify-end">
                    {needsReauth && row.kind === "oauth" && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={busyRow || !session?.access_token}
                        onClick={() => handleReauthorize(row.connection)}
                      >
                        <LogIn className="size-3.5" />
                        Re-authorize
                      </Button>
                    )}
                    {enabled && isDefault && (
                      <Badge variant="secondary">Default</Badge>
                    )}
                    {enabled && showDefaultControls && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={busyRow || isDefault}
                        onClick={() =>
                          inst &&
                          void handleSetDefault(inst.connector_instance_id)
                        }
                      >
                        <Star className="size-3.5" />
                        Make default
                      </Button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {showManageButton && (
          <div className="mt-4 flex justify-end">
            <Button
              type="button"
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
