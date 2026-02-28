import { useState } from "react";
import { Loader2, Plus, Plug } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { useConnectors } from "@/hooks/useConnectors";
import { useEnableAgentConnector } from "@/hooks/useEnableAgentConnector";
import type { AgentConnector } from "@/hooks/useAgentConnectors";
import type { ConnectorSummary } from "@/hooks/useConnectors";

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

  const enabledIds = new Set(enabledConnectors.map((c) => c.id));
  const available = allConnectors.filter((c) => !enabledIds.has(c.id));

  async function handleEnable(connector: ConnectorSummary) {
    setEnablingId(connector.id);
    try {
      await enableConnector({ agentId, connectorId: connector.id });
      toast.success(`${connector.name} enabled`);
      onOpenChange(false);
    } catch {
      toast.error(`Failed to enable ${connector.name}`);
    } finally {
      setEnablingId(null);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
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
          <div className="max-h-80 overflow-y-auto rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Connector
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    <span className="sr-only">Enable</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {available.map((connector) => (
                  <TableRow key={connector.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{connector.name}</p>
                        {connector.description && (
                          <p className="text-muted-foreground text-xs">
                            {connector.description}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="default"
                        size="sm"
                        onClick={() => handleEnable(connector)}
                        disabled={enablingId !== null}
                      >
                        {enablingId === connector.id ? (
                          <Loader2 className="size-4 animate-spin" />
                        ) : (
                          <Plus className="size-4" />
                        )}
                        Enable
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
