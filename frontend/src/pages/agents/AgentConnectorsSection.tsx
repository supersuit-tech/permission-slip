import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2, Plus, Plug } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { useDisableAgentConnector } from "@/hooks/useDisableAgentConnector";
import type { AgentConnector } from "@/hooks/useAgentConnectors";
import { AddConnectorDialog } from "./AddConnectorDialog";
import { ConnectorLogo } from "@/components/ConnectorLogo";

interface AgentConnectorsSectionProps {
  agentId: number;
  connectors: AgentConnector[];
  isLoading: boolean;
  error: string | null;
}

export function AgentConnectorsSection({
  agentId,
  connectors,
  isLoading,
  error,
}: AgentConnectorsSectionProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Enabled Connectors</CardTitle>
          <Button
            size="sm"
            onClick={() => setAddDialogOpen(true)}
            disabled={isLoading || !!error}
          >
            <Plus className="size-4" />
            Add Connector
          </Button>
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
          ) : connectors.length === 0 ? (
            <EmptyConnectors onAdd={() => setAddDialogOpen(true)} />
          ) : (
            <ConnectorsTable connectors={connectors} agentId={agentId} />
          )}
        </CardContent>
      </Card>

      <AddConnectorDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        agentId={agentId}
        enabledConnectors={connectors}
      />
    </>
  );
}

function EmptyConnectors({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <Plug className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No connectors enabled
      </p>
      <p className="text-muted-foreground mb-4 max-w-xs text-xs">
        Enable connectors for this agent to allow it to submit actions from
        external services.
      </p>
      <Button variant="outline" size="sm" onClick={onAdd}>
        <Plus className="size-4" />
        Add Connector
      </Button>
    </div>
  );
}

function ConnectorsTable({
  connectors,
  agentId,
}: {
  connectors: AgentConnector[];
  agentId: number;
}) {
  return (
    <div className="overflow-hidden rounded-lg">
      <Table>
        <TableHeader>
          <TableRow className="border-none bg-primary hover:bg-primary">
            <TableHead className="font-semibold text-primary-foreground">
              Connector
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              Actions
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              Enabled
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className="[&>tr:nth-child(even)]:bg-muted">
          {connectors.map((connector) => (
            <ConnectorRow
              key={connector.id}
              connector={connector}
              agentId={agentId}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function ConnectorRow({
  connector,
  agentId,
}: {
  connector: AgentConnector;
  agentId: number;
}) {
  const navigate = useNavigate();
  const [removeDialogOpen, setRemoveDialogOpen] = useState(false);
  const { disableConnector, isLoading: disabling } = useDisableAgentConnector();

  async function handleRemove() {
    try {
      const result = await disableConnector({
        agentId,
        connectorId: connector.id,
      });
      const revoked = result.revoked_standing_approvals;
      if (revoked > 0) {
        toast.success(
          `${connector.name} disabled. ${revoked} standing approval${revoked === 1 ? "" : "s"} revoked.`,
        );
      } else {
        toast.success(`${connector.name} disabled`);
      }
      setRemoveDialogOpen(false);
    } catch {
      toast.error(`Failed to disable ${connector.name}`);
    }
  }

  return (
    <>
      <TableRow>
        <TableCell>
          <div className="flex items-center gap-2.5">
            <ConnectorLogo
              name={connector.name}
              logoSvg={connector.logo_svg}
              size="sm"
            />
            <div>
              <p className="font-medium">{connector.name}</p>
              {connector.description && (
                <p className="text-muted-foreground text-xs">
                  {connector.description}
                </p>
              )}
            </div>
          </div>
        </TableCell>
        <TableCell className="text-muted-foreground text-sm">
          {connector.actions.join(", ")}
        </TableCell>
        <TableCell className="text-muted-foreground text-sm">
          {connector.enabled_at
            ? new Date(connector.enabled_at).toLocaleDateString()
            : "—"}
        </TableCell>
        <TableCell className="text-right">
          <div className="flex items-center justify-end gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() =>
                navigate(`/agents/${agentId}/connectors/${connector.id}`)
              }
            >
              Configure
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={() => setRemoveDialogOpen(true)}
            >
              Remove
            </Button>
          </div>
        </TableCell>
      </TableRow>

      <Dialog open={removeDialogOpen} onOpenChange={setRemoveDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove {connector.name}</DialogTitle>
            <DialogDescription>
              This will disable the <strong>{connector.name}</strong> connector
              for this agent. Any active standing approvals for actions from
              this connector will be automatically revoked.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setRemoveDialogOpen(false)}
              disabled={disabling}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleRemove}
              disabled={disabling}
            >
              {disabling && <Loader2 className="animate-spin" />}
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
