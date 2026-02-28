import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useActionConfigs } from "@/hooks/useActionConfigs";
import { useConnectorDetail } from "@/hooks/useConnectorDetail";
import { useAgentConnectors } from "@/hooks/useAgentConnectors";
import { useCredentials } from "@/hooks/useCredentials";
import { ConnectorOverviewSection } from "./ConnectorOverviewSection";
import { ConnectorActionsSection } from "./ConnectorActionsSection";
import { ActionConfigurationsSection } from "./ActionConfigurationsSection";
import { ConnectorCredentialsSection } from "./ConnectorCredentialsSection";
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
  } = useActionConfigs(agentId);

  const connectorConfigs = configs.filter((c) => c.connector_id === connectorId);

  const hasRequiredCredentials =
    !connectorLoading && !!connector && connector.required_credentials.length > 0;
  const hasConfigCredentials = connectorConfigs.some((c) => !!c.credential_id);
  const shouldFetchCredentials = hasRequiredCredentials || hasConfigCredentials;
  const { credentials } = useCredentials({ enabled: shouldFetchCredentials });

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
      <ConnectorActionsSection actions={connector.actions} />
      <ActionConfigurationsSection
        agentId={agentId}
        connectorId={connectorId}
        actions={connector.actions}
        credentials={credentials}
        configs={connectorConfigs}
        isLoading={configsLoading}
        error={configsError}
      />
      <ConnectorCredentialsSection
        requiredCredentials={connector.required_credentials}
      />
      <DisableConnectorSection
        agentId={agentId}
        connectorId={connectorId}
        connectorName={connector.name}
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
