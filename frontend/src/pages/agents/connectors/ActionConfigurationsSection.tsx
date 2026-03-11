import { useState } from "react";
import { ChevronDown, ChevronRight, Loader2, Plus, Settings, Zap } from "lucide-react";
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
import { useCreateActionConfig } from "@/hooks/useCreateActionConfig";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { CredentialSummary } from "@/hooks/useCredentials";
import { ActionConfigRow } from "./ActionConfigRow";
import { AddActionConfigDialog } from "./AddActionConfigDialog";
import { EditActionConfigDialog } from "./EditActionConfigDialog";
import { DeleteActionConfigDialog } from "./DeleteActionConfigDialog";

interface ActionConfigurationsSectionProps {
  agentId: number;
  connectorId: string;
  connectorName: string;
  actions: ConnectorAction[];
  credentials: CredentialSummary[];
  configs: ActionConfiguration[];
  isLoading: boolean;
  error: string | null;
}

export function ActionConfigurationsSection({
  agentId,
  connectorId,
  connectorName,
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
  const [showAdvanced, setShowAdvanced] = useState(false);

  const { createActionConfig, isPending: isEnablingAll } =
    useCreateActionConfig();

  const handleEnableAll = async () => {
    await createActionConfig({
      agent_id: agentId,
      connector_id: connectorId,
      action_type: "*",
      name: `All ${connectorName} Actions`,
      parameters: {},
    });
  };

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
          <EnableAllEmptyState
            onEnableAll={handleEnableAll}
            isEnablingAll={isEnablingAll}
            showAdvanced={showAdvanced}
            onToggleAdvanced={() => setShowAdvanced((v) => !v)}
            onAddCustom={() => setAddDialogOpen(true)}
            actionsDisabled={actions.length === 0}
          />
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

function EnableAllEmptyState({
  onEnableAll,
  isEnablingAll,
  showAdvanced,
  onToggleAdvanced,
  onAddCustom,
  actionsDisabled,
}: {
  onEnableAll: () => void;
  isEnablingAll: boolean;
  showAdvanced: boolean;
  onToggleAdvanced: () => void;
  onAddCustom: () => void;
  actionsDisabled: boolean;
}) {
  return (
    <div className="space-y-4 py-4 text-center">
      <div className="space-y-3">
        <Button
          size="lg"
          onClick={onEnableAll}
          disabled={isEnablingAll || actionsDisabled}
        >
          {isEnablingAll ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Zap className="size-4" />
          )}
          Enable All Actions
        </Button>
        <p className="text-muted-foreground mx-auto max-w-md text-sm">
          Your agent can use any action from this connector. Every action still
          requires your approval before it runs.
        </p>
      </div>

      <div className="pt-2">
        <button
          type="button"
          onClick={onToggleAdvanced}
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs transition-colors"
        >
          {showAdvanced ? (
            <ChevronDown className="size-3" />
          ) : (
            <ChevronRight className="size-3" />
          )}
          Advanced: configure individual actions
        </button>
        {showAdvanced && (
          <div className="mt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onAddCustom}
              disabled={actionsDisabled}
            >
              <Plus className="size-4" />
              Add Custom Configuration
            </Button>
            <p className="text-muted-foreground mt-1 text-xs">
              Lock specific parameters or restrict which actions your agent can
              use.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
