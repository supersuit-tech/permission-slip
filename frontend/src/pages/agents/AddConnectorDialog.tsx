import { useEffect, useMemo, useState } from "react";
import { Loader2, Plus, Plug, Search } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ConnectorLogo } from "@/components/ConnectorLogo";
import { useConnectors } from "@/hooks/useConnectors";
import { useEnableAgentConnector } from "@/hooks/useEnableAgentConnector";
import type { AgentConnector } from "@/hooks/useAgentConnectors";
import type { ConnectorSummary } from "@/hooks/useConnectors";
import { SetupConnectorCredentialsDialog } from "./connectors/SetupConnectorCredentialsDialog";

const STATUS_SECTIONS = [
  { key: "tested", label: "Tested", description: "Production-ready connectors" },
  { key: "early_preview", label: "Early Preview", description: "Functional but still being validated" },
  { key: "untested", label: "Untested", description: "Not yet verified against live services" },
] as const;

interface AddConnectorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agentId: number;
  enabledConnectors: AgentConnector[];
}

export function AddConnectorDialog({
  open,
  onOpenChange,
  agentId,
  enabledConnectors,
}: AddConnectorDialogProps) {
  const {
    connectors: allConnectors,
    isLoading,
    error,
  } = useConnectors();
  const { enableConnector } = useEnableAgentConnector();
  const [enablingId, setEnablingId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [setupConnector, setSetupConnector] = useState<ConnectorSummary | null>(null);

  // Reset setup state when the parent closes the dialog externally (e.g. by
  // navigating away). Without this, reopening the add-connector dialog would
  // skip straight to the stale setup modal for the previously-enabled connector.
  useEffect(() => {
    if (!open) {
      setSetupConnector(null);
      setSearch("");
    }
  }, [open]);

  const enabledIds = new Set(enabledConnectors.map((c) => c.id));
  const available = allConnectors.filter((c) => !enabledIds.has(c.id));

  const filtered = search.trim()
    ? available.filter(
        (c) =>
          c.name.toLowerCase().includes(search.toLowerCase()) ||
          c.description?.toLowerCase().includes(search.toLowerCase()),
      )
    : available;

  const groupedByStatus = useMemo(() => {
    const groups: Record<string, ConnectorSummary[]> = {};
    for (const section of STATUS_SECTIONS) {
      groups[section.key] = [];
    }
    for (const connector of filtered) {
      const status = connector.status ?? "untested";
      const bucket = groups[status] ?? groups["untested"];
      bucket?.push(connector);
    }
    return groups;
  }, [filtered]);

  async function handleEnable(connector: ConnectorSummary) {
    setEnablingId(connector.id);
    try {
      await enableConnector({ agentId, connectorId: connector.id });
      toast.success(`${connector.name} enabled`);
      setSetupConnector(connector);
    } catch {
      toast.error(`Failed to enable ${connector.name}`);
    } finally {
      setEnablingId(null);
    }
  }

  return (
    <>
    <Dialog open={open && !setupConnector} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Add Connector</DialogTitle>
          <DialogDescription>
            Enable a connector for this agent to allow it to submit actions
            from external services.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2
              className="text-muted-foreground size-6 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error ? (
          <p className="text-destructive py-4 text-sm">{error}</p>
        ) : available.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Plug className="text-muted-foreground mb-3 size-10" />
            <p className="text-muted-foreground text-sm">
              All available connectors are already enabled.
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            <div className="relative">
              <Search className="text-muted-foreground absolute left-3 top-1/2 size-4 -translate-y-1/2" />
              <Input
                placeholder="Search connectors..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>

            {filtered.length === 0 ? (
              <p className="text-muted-foreground py-6 text-center text-sm">
                No connectors match &ldquo;{search}&rdquo;
              </p>
            ) : (
              <div className="max-h-96 space-y-4 overflow-y-auto">
                {STATUS_SECTIONS.map((section) => {
                  const connectors = groupedByStatus[section.key];
                  if (!connectors || connectors.length === 0) return null;
                  return (
                    <ConnectorStatusSection
                      key={section.key}
                      label={section.label}
                      description={section.description}
                      status={section.key}
                      connectors={connectors}
                      enablingId={enablingId}
                      onEnable={handleEnable}
                    />
                  );
                })}
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>

    {setupConnector && (
      <SetupConnectorCredentialsDialog
        open
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setSetupConnector(null);
            onOpenChange(false);
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

function ConnectorStatusSection({
  label,
  description,
  status,
  connectors,
  enablingId,
  onEnable,
}: {
  label: string;
  description: string;
  status: string;
  connectors: ConnectorSummary[];
  enablingId: string | null;
  onEnable: (connector: ConnectorSummary) => void;
}) {
  return (
    <div>
      <div className="mb-2 flex items-center gap-2">
        <StatusBadge status={status} label={label} />
        <span className="text-muted-foreground text-xs">{description}</span>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
        {connectors.map((connector) => (
          <ConnectorCard
            key={connector.id}
            connector={connector}
            enabling={enablingId === connector.id}
            disabled={enablingId !== null}
            onEnable={onEnable}
          />
        ))}
      </div>
    </div>
  );
}

function StatusBadge({ status, label }: { status: string; label: string }) {
  const variant =
    status === "tested"
      ? "default"
      : status === "early_preview"
        ? "secondary"
        : "outline";
  return <Badge variant={variant}>{label}</Badge>;
}

function ConnectorCard({
  connector,
  enabling,
  disabled,
  onEnable,
}: {
  connector: ConnectorSummary;
  enabling: boolean;
  disabled: boolean;
  onEnable: (connector: ConnectorSummary) => void;
}) {
  return (
    <div className="border-border bg-card flex flex-col items-center gap-2 rounded-lg border p-3 text-center">
      <ConnectorLogo
        name={connector.name}
        logoSvg={connector.logo_svg}
        size="lg"
      />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium leading-tight">{connector.name}</p>
        {connector.description && (
          <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs leading-tight">
            {connector.description}
          </p>
        )}
      </div>
      <Button
        variant="default"
        size="sm"
        className="w-full"
        onClick={() => onEnable(connector)}
        disabled={disabled}
      >
        {enabling ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : (
          <Plus className="size-3.5" />
        )}
        Enable
      </Button>
    </div>
  );
}
