import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2, Plug, Search } from "lucide-react";
import { toast } from "sonner";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from "@/components/ui/card";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { useConnectors } from "@/hooks/useConnectors";
import type { ConnectorSummary } from "@/hooks/useConnectors";
import { useEnableAgentConnector } from "@/hooks/useEnableAgentConnector";
import type { AgentConnector } from "@/hooks/useAgentConnectors";

interface AgentConnectorsSectionProps {
  agentId: number;
  connectors: AgentConnector[];
  isLoading: boolean;
  error: string | null;
}

/**
 * Searchable grid of all available connectors for an agent.
 * Enabled connectors show an "Enabled" badge and navigate directly to config.
 * Non-enabled connectors are auto-enabled on click before navigating.
 */
export function AgentConnectorsSection({
  agentId,
  connectors: enabledConnectors,
  isLoading: enabledLoading,
  error: enabledError,
}: AgentConnectorsSectionProps) {
  const {
    connectors: allConnectors,
    isLoading: allLoading,
    error: allError,
  } = useConnectors();
  const { enableConnector } = useEnableAgentConnector();
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [enablingId, setEnablingId] = useState<string | null>(null);

  const isLoading = enabledLoading || allLoading;
  const error = allError ?? enabledError;

  const enabledIds = new Set(enabledConnectors.map((c) => c.id));

  const query = search.trim().toLowerCase();
  const filtered = query
    ? allConnectors.filter(
        (c) =>
          c.name.toLowerCase().includes(query) ||
          c.description?.toLowerCase().includes(query),
      )
    : allConnectors;

  async function handleClick(connector: ConnectorSummary) {
    if (enabledIds.has(connector.id)) {
      navigate(`/agents/${agentId}/connectors/${connector.id}`);
      return;
    }

    setEnablingId(connector.id);
    try {
      await enableConnector({ agentId, connectorId: connector.id });
      toast.success(`${connector.name} enabled`);
      navigate(`/agents/${agentId}/connectors/${connector.id}`);
    } catch (err) {
      console.error(`Failed to enable connector ${connector.id}:`, err);
      toast.error(`Failed to enable ${connector.name}`);
    } finally {
      setEnablingId(null);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Connectors</CardTitle>
        <CardDescription>
          Services this agent can interact with. Click a connector to configure
          credentials and actions.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading connectors"
          >
            <Loader2
              className="text-muted-foreground size-6 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : allConnectors.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Plug className="text-muted-foreground mb-3 size-10" />
            <p className="text-muted-foreground text-sm">
              No connectors are available yet.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {allConnectors.length >= 4 && (
              <div className="relative">
                <Search className="text-muted-foreground absolute left-3 top-1/2 size-4 -translate-y-1/2" />
                <Input
                  placeholder="Search connectors..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9"
                />
              </div>
            )}

            {filtered.length === 0 ? (
              <p className="text-muted-foreground py-6 text-center text-sm">
                No connectors match &ldquo;{search.trim()}&rdquo;
              </p>
            ) : (
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                {filtered.map((connector) => (
                  <ConnectorCard
                    key={connector.id}
                    connector={connector}
                    isEnabled={enabledIds.has(connector.id)}
                    isEnabling={enablingId === connector.id}
                    disabled={enablingId !== null}
                    onClick={handleClick}
                  />
                ))}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ConnectorCard({
  connector,
  isEnabled,
  isEnabling,
  disabled,
  onClick,
}: {
  connector: ConnectorSummary;
  isEnabled: boolean;
  isEnabling: boolean;
  disabled: boolean;
  onClick: (connector: ConnectorSummary) => void;
}) {
  const actionCount = connector.actions.length;

  return (
    <button
      type="button"
      className="border-border bg-card hover:bg-accent flex flex-col items-center gap-2 rounded-lg border p-3 text-center transition-colors disabled:opacity-50"
      onClick={() => onClick(connector)}
      disabled={disabled}
    >
      {isEnabling ? (
        <Loader2 className="text-muted-foreground size-10 animate-spin" />
      ) : (
        <ConnectorLogo
          name={connector.name}
          logoSvg={connector.logo_svg}
          size="lg"
        />
      )}
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-center gap-1.5">
          <p className="text-sm font-medium leading-tight">{connector.name}</p>
          {isEnabled && (
            <Badge
              variant="secondary"
              className="px-1.5 py-0 text-[10px] leading-4"
            >
              Enabled
            </Badge>
          )}
        </div>
        {connector.description && (
          <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs leading-tight">
            {connector.description}
          </p>
        )}
        <p className="text-muted-foreground mt-0.5 text-xs leading-tight">
          {actionCount} action{actionCount !== 1 ? "s" : ""}
        </p>
      </div>
    </button>
  );
}
