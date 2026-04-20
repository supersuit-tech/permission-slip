import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAgent } from "@/hooks/useAgent";
import { useActionConfigs } from "@/hooks/useActionConfigs";
import { useConnectorDetail } from "@/hooks/useConnectorDetail";
import { useAgentConnectors } from "@/hooks/useAgentConnectors";
import { ConnectorOverviewSection } from "./ConnectorOverviewSection";
import { ConnectorActionsDialog } from "./ConnectorActionsDialog";
import { ActionConfigurationsSection } from "./ActionConfigurationsSection";
import { ConnectorInstancesSection } from "./ConnectorInstancesSection";
import { DisableConnectorSection } from "./DisableConnectorSection";

export function ConnectorConfigPage() {
  const { agentId: rawAgentId, connectorId } = useParams<{
    agentId: string;
    connectorId: string;
  }>();
  const parsedAgentId = Number(rawAgentId);
  const paramsValid =
    !isNaN(parsedAgentId) && parsedAgentId > 0 && !!connectorId;
  // Pass 0 when invalid so the hook's own `enabled: agentId > 0` guard
  // prevents the fetch without creating a query keyed on NaN.
  const agentId = paramsValid ? parsedAgentId : 0;

  // Verify the current user owns this agent. The backend scopes by user,
  // so a non-owned agent returns 404. Without this check the page would
  // still render public connector catalog data for arbitrary agent IDs.
  const { agent, isLoading: agentLoading, error: agentError } = useAgent(agentId);

  const {
    connector,
    isLoading: connectorLoading,
    error: connectorError,
    refetch,
  } = useConnectorDetail(connectorId ?? "");

  const { connectors: agentConnectors, isLoading: agentConnectorsLoading } =
    useAgentConnectors(agentId);

  const {
    configs,
    isLoading: configsLoading,
    error: configsError,
    refetch: refetchConfigs,
  } = useActionConfigs(agentId);

  const connectorConfigs = configs.filter((c) => c.connector_id === connectorId);

  const [actionsDialogOpen, setActionsDialogOpen] = useState(false);

  const backTo = `/agents/${rawAgentId}`;

  if (!paramsValid) {
    return (
      <div className="space-y-4">
        <BackLink to={backTo} />
        <p className="text-destructive text-sm">
          Invalid agent ID or connector ID.
        </p>
      </div>
    );
  }

  // Block access until we confirm the current user owns this agent.
  if (agentLoading) {
    return (
      <div className="space-y-4">
        <BackLink to={backTo} />
        <div
          className="flex items-center justify-center py-12"
          role="status"
          aria-label="Loading agent"
        >
          <Loader2
            className="text-muted-foreground size-6 animate-spin"
            aria-hidden="true"
          />
        </div>
      </div>
    );
  }

  if (agentError || !agent) {
    return (
      <div className="space-y-4">
        <BackLink to={backTo} />
        <p className="text-destructive text-sm">Agent not found.</p>
      </div>
    );
  }

  if (connectorLoading || agentConnectorsLoading) {
    return (
      <div className="space-y-4">
        <BackLink to={backTo} />
        <div
          className="flex items-center justify-center py-12"
          role="status"
          aria-label="Loading connector"
        >
          <Loader2
            className="text-muted-foreground size-6 animate-spin"
            aria-hidden="true"
          />
        </div>
      </div>
    );
  }

  if (connectorError || !connector) {
    return (
      <div className="space-y-4">
        <BackLink to={backTo} />
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <p className="text-destructive mb-2 text-sm">
            {connectorError ?? "Connector not found."}
          </p>
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const agentConnector = agentConnectors.find((c) => c.id === connectorId);

  return (
    <div className="space-y-6">
      <BackLink to={backTo} />
      <ConnectorOverviewSection
        connector={connector}
        enabledAt={agentConnector?.enabled_at}
      />
      <button
        type="button"
        onClick={() => setActionsDialogOpen(true)}
        className="text-muted-foreground hover:text-foreground -mt-4 inline-flex items-center gap-1 text-sm transition-colors"
      >
        <ExternalLink className="size-3.5" />
        View all {connector.actions.length} available actions
      </button>
      <ConnectorActionsDialog
        open={actionsDialogOpen}
        onOpenChange={setActionsDialogOpen}
        actions={connector.actions}
      />
      <ActionConfigurationsSection
        agentId={agentId}
        connectorId={connectorId}
        connectorName={connector.name}
        actions={connector.actions}
        configs={connectorConfigs}
        isLoading={configsLoading}
        error={configsError}
        onConfigsChanged={() => void refetchConfigs()}
      />
      <ConnectorInstancesSection
        agentId={agentId}
        connectorId={connectorId}
        requiredCredentials={connector.required_credentials}
      />
      <DisableConnectorSection
        agentId={agentId}
        connectorId={connectorId}
        connectorName={connector.name}
        oauthProvider={
          connector.required_credentials?.find(
            (c) => c.auth_type === "oauth2" && c.oauth_provider,
          )?.oauth_provider
        }
      />
    </div>
  );
}

function BackLink({ to }: { to: string }) {
  return (
    <Link
      to={to}
      className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm transition-colors"
    >
      <ArrowLeft className="size-4" />
      Back to Agent
    </Link>
  );
}
