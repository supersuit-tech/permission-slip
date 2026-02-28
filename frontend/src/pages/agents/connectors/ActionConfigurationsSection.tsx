import { useState } from "react";
import { Loader2, Plus, Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
} from "@/components/ui/table";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { ActionConfigRow } from "./ActionConfigRow";
import { AddActionConfigDialog } from "./AddActionConfigDialog";
import { EditActionConfigDialog } from "./EditActionConfigDialog";
import { DeleteActionConfigDialog } from "./DeleteActionConfigDialog";

interface ActionConfigurationsSectionProps {
  agentId: number;
  connectorId: string;
  actions: ConnectorAction[];
  credentials: CredentialSummary[];
  configs: ActionConfiguration[];
  isLoading: boolean;
  error: string | null;
}

export function ActionConfigurationsSection({
  agentId,
  connectorId,
  actions,
  credentials,
  configs,
  isLoading,
  error,
}: ActionConfigurationsSectionProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<ActionConfiguration | null>(
    null,
  );
  const [deleteTarget, setDeleteTarget] = useState<ActionConfiguration | null>(
    null,
  );

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div className="flex items-center gap-2">
          <Settings className="text-muted-foreground size-5" />
          <CardTitle>Action Configurations</CardTitle>
        </div>
        <Button
          size="sm"
          onClick={() => setAddDialogOpen(true)}
          disabled={actions.length === 0}
        >
          <Plus className="size-4" />
          Add Configuration
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2
              className="text-muted-foreground size-5 animate-spin"
              aria-hidden="true"
            />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : configs.length === 0 ? (
          <div className="text-muted-foreground space-y-1 py-4 text-center text-sm">
            <p>No action configurations yet.</p>
            <p>
              Create a configuration to define exactly how this agent can use an
              action — which parameters are locked in and which the agent can
              choose freely.
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg">
            <Table>
              <TableHeader>
                <TableRow className="border-none bg-primary hover:bg-primary">
                  <TableHead className="font-semibold text-primary-foreground">
                    Name
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Action
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Parameters
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Credential
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Status
                  </TableHead>
                  <TableHead className="w-[100px] font-semibold text-primary-foreground" />
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {configs.map((config) => (
                  <ActionConfigRow
                    key={config.id}
                    config={config}
                    actions={actions}
                    credentials={credentials}
                    onEdit={setEditTarget}
                    onDelete={setDeleteTarget}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <AddActionConfigDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        credentials={credentials}
      />

      {editTarget && (
        <EditActionConfigDialog
          open={!!editTarget}
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
          config={editTarget}
          agentId={agentId}
          actions={actions}
          credentials={credentials}
        />
      )}

      {deleteTarget && (
        <DeleteActionConfigDialog
          open={!!deleteTarget}
          onOpenChange={(open) => {
            if (!open) setDeleteTarget(null);
          }}
          config={deleteTarget}
          agentId={agentId}
        />
      )}
    </Card>
  );
}
