import { useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Loader2, Plus, Settings, Zap } from "lucide-react";
import { toast } from "sonner";
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
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { WILDCARD_ACTION_TYPE } from "./ActionConfigFormFields";
import { ActionConfigRow } from "./ActionConfigRow";
import { AddActionConfigDialog } from "./AddActionConfigDialog";
import { EditActionConfigDialog } from "./EditActionConfigDialog";
import { DeleteActionConfigDialog } from "./DeleteActionConfigDialog";
import { RecommendedTemplatesDialog } from "./RecommendedTemplatesDialog";

interface ActionConfigurationsSectionProps {
  agentId: number;
  connectorId: string;
  connectorName: string;
  actions: ConnectorAction[];
  configs: ActionConfiguration[];
  isLoading: boolean;
  error: string | null;
}

export function ActionConfigurationsSection({
  agentId,
  connectorId,
  connectorName,
  actions,
  configs,
  isLoading,
  error,
}: ActionConfigurationsSectionProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [initialTemplateForAdd, setInitialTemplateForAdd] =
    useState<ActionConfigTemplate | null>(null);
  const [recommendedDialogOpen, setRecommendedDialogOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<ActionConfiguration | null>(
    null,
  );
  const [deleteTarget, setDeleteTarget] = useState<ActionConfiguration | null>(
    null,
  );
  const [showAdvanced, setShowAdvanced] = useState(false);

  const { templates, isLoading: templatesLoading } =
    useActionConfigTemplates(connectorId);

  const actionTypeSet = useMemo(
    () => new Set(actions.map((a) => a.action_type)),
    [actions],
  );
  const hasRecommendedTemplates =
    !templatesLoading &&
    templates.some((t) => actionTypeSet.has(t.action_type));

  const hasWildcardConfig = configs.some(
    (c) => c.action_type === WILDCARD_ACTION_TYPE,
  );

  const { createActionConfig, isPending: isEnablingAll } =
    useCreateActionConfig();

  const handleEnableAll = async () => {
    try {
      await createActionConfig({
        agent_id: agentId,
        connector_id: connectorId,
        action_type: WILDCARD_ACTION_TYPE,
        name: `All ${connectorName} Actions`,
        parameters: {},
      });
      toast.success(`All ${connectorName} actions enabled`);
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to enable all actions",
      );
    }
  };

  function openAddDialog(fromTemplate?: ActionConfigTemplate | null) {
    setInitialTemplateForAdd(fromTemplate ?? null);
    setAddDialogOpen(true);
  }

  function handleAddDialogOpenChange(open: boolean) {
    if (!open) {
      setInitialTemplateForAdd(null);
    }
    setAddDialogOpen(open);
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Settings className="text-muted-foreground size-5" />
          <CardTitle>Action Configurations</CardTitle>
        </div>
        {configs.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 self-start sm:self-center">
            {!hasWildcardConfig && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="shrink-0"
                onClick={handleEnableAll}
                disabled={isEnablingAll || actions.length === 0}
              >
                {isEnablingAll ? (
                  <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                ) : (
                  <Zap className="size-4" aria-hidden="true" />
                )}
                Enable All Actions
              </Button>
            )}
            {hasRecommendedTemplates && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="shrink-0"
                onClick={() => setRecommendedDialogOpen(true)}
                disabled={actions.length === 0}
              >
                Recommended Templates
              </Button>
            )}
            <Button
              type="button"
              size="sm"
              className="shrink-0"
              onClick={() => openAddDialog()}
              disabled={actions.length === 0}
            >
              <Plus className="size-4" />
              Add Configuration
            </Button>
          </div>
        )}
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
            onAddCustom={() => openAddDialog()}
            onBrowseRecommendedTemplates={() =>
              setRecommendedDialogOpen(true)
            }
            showRecommendedLink={hasRecommendedTemplates}
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
        onOpenChange={handleAddDialogOpenChange}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        initialTemplate={initialTemplateForAdd}
      />

      <RecommendedTemplatesDialog
        open={recommendedDialogOpen}
        onOpenChange={setRecommendedDialogOpen}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        onCustomize={(template) => {
          openAddDialog(template);
        }}
      />

      {editTarget && (
        <EditActionConfigDialog
          open={!!editTarget}
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
          config={editTarget}
          agentId={agentId}
          connectorId={connectorId}
          actions={actions}
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
  onBrowseRecommendedTemplates,
  showRecommendedLink,
  actionsDisabled,
}: {
  onEnableAll: () => void;
  isEnablingAll: boolean;
  showAdvanced: boolean;
  onToggleAdvanced: () => void;
  onAddCustom: () => void;
  onBrowseRecommendedTemplates: () => void;
  showRecommendedLink: boolean;
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
        {showRecommendedLink && (
          <div>
            <button
              type="button"
              onClick={onBrowseRecommendedTemplates}
              disabled={actionsDisabled}
              className="text-muted-foreground hover:text-foreground text-sm underline-offset-4 transition-colors hover:underline disabled:pointer-events-none disabled:opacity-50"
            >
              Or start from a recommended template →
            </button>
          </div>
        )}
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
