import { useMemo, useState } from "react";
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
import { SetupConnectorCredentialsDialog } from "./connectors/SetupConnectorCredentialsDialog";

const STATUS_SECTIONS = [
  { key: "tested", label: "Tested", description: "Production-ready connectors" },
  { key: "early_preview", label: "Early Preview", description: "Functional but still being validated" },
  { key: "untested", label: "Untested", description: "Not yet verified against live services" },
] as const;

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
  const [setupConnector, setSetupConnector] = useState<ConnectorSummary | null>(null);

  const isLoading = enabledLoading || allLoading;
  const error = allError ?? enabledError;

  const enabledIds = new Set(enabledConnectors.map((c) => c.id));

  const { filtered, groupedByStatus } = useMemo(() => {
    const query = search.trim().toLowerCase();
    const result = (
      query
        ? allConnectors.filter(
            (c) =>
              c.name.toLowerCase().includes(query) ||
              c.description?.toLowerCase().includes(query),
          )
        : allConnectors
    )
      .slice()
      .sort((a, b) => {
        const aEnabled = enabledIds.has(a.id) ? 0 : 1;
        const bEnabled = enabledIds.has(b.id) ? 0 : 1;
        return aEnabled - bEnabled;
      });

    const groups: Record<string, ConnectorSummary[]> = {};
    for (const section of STATUS_SECTIONS) {
      groups[section.key] = [];
    }
    for (const connector of result) {
      const status = connector.status ?? "untested";
      const bucket = groups[status] ?? groups["untested"];
      bucket?.push(connector);
    }
    return { filtered: result, groupedByStatus: groups };
  }, [allConnectors, enabledConnectors, search]);

  async function handleClick(connector: ConnectorSummary) {
    if (enabledIds.has(connector.id)) {
      navigate(`/agents/${agentId}/connectors/${connector.id}`);
      return;
    }

    setEnablingId(connector.id);
    try {
      await enableConnector({ agentId, connectorId: connector.id });
      toast.success(`${connector.name} enabled`);
      setSetupConnector(connector);
    } catch (err) {
      console.error(`Failed to enable connector ${connector.id}:`, err);
      toast.error(`Failed to enable ${connector.name}`);
    } finally {
      setEnablingId(null);
    }
  }

  return (
    <>
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
              <div className="space-y-4">
                {STATUS_SECTIONS.map((section) => {
                  const connectors = groupedByStatus[section.key];
                  if (!connectors || connectors.length === 0) return null;
                  return (
                    <div key={section.key}>
                      <div className="mb-2 flex items-center gap-2">
                        <StatusBadge status={section.key} label={section.label} />
                        <span className="text-muted-foreground text-xs">{section.description}</span>
                      </div>
                      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                        {connectors.map((connector) => (
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
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>

    {setupConnector && (
      <SetupConnectorCredentialsDialog
        open
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            const connectorId = setupConnector.id;
            setSetupConnector(null);
            navigate(`/agents/${agentId}/connectors/${connectorId}`);
          }
        }}
        agentId={agentId}
        connectorId={setupConnector.id}
        connectorName={setupConnector.name}
        connectorLogoSvg={setupConnector.logo_svg}
      />
    )}
    </>
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

type ConnectorStatus = "tested" | "early_preview" | "untested";

function StatusBadge({ status, label }: { status: ConnectorStatus; label: string }) {
  const variant =
    status === "tested"
      ? "default"
      : status === "early_preview"
        ? "secondary"
        : "outline";
  return <Badge variant={variant}>{label}</Badge>;
}
