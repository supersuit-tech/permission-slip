import { useMemo, useState } from "react";
import { useStandingApprovalsForConfigs } from "@/hooks/useStandingApprovalsForConfigs";
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
  TableCell,
} from "@/components/ui/table";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import { useActionConfigTemplates } from "@/hooks/useActionConfigTemplates";
import type { ActionConfigTemplate } from "@/hooks/useActionConfigTemplates";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import type { ApprovalMode } from "./RecommendedTemplatesDialog";
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
  onConfigsChanged?: () => void;
}

export function ActionConfigurationsSection({
  agentId,
  connectorId,
  actions,
  configs,
  isLoading,
  error,
  onConfigsChanged,
}: ActionConfigurationsSectionProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [initialTemplateForAdd, setInitialTemplateForAdd] =
    useState<ActionConfigTemplate | null>(null);
  const [initialApprovalModeForAdd, setInitialApprovalModeForAdd] =
    useState<ApprovalMode | undefined>(undefined);
  const [recommendedDialogOpen, setRecommendedDialogOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<ActionConfiguration | null>(
    null,
  );
  const [deleteTarget, setDeleteTarget] = useState<ActionConfiguration | null>(
    null,
  );

  const { templates, isLoading: templatesLoading } =
    useActionConfigTemplates(connectorId);

  const actionTypeSet = useMemo(
    () => new Set(actions.map((a) => a.action_type)),
    [actions],
  );
  const hasRecommendedTemplates =
    !templatesLoading &&
    templates.some((t) => actionTypeSet.has(t.action_type));

  const configIds = useMemo(() => configs.map((c) => c.id), [configs]);
  const {
    byConfigId: standingByConfig,
    error: standingError,
    refetch: refetchStanding,
  } = useStandingApprovalsForConfigs(configIds);

  function openAddDialog(fromTemplate?: ActionConfigTemplate | null, approvalMode?: ApprovalMode) {
    setInitialTemplateForAdd(fromTemplate ?? null);
    setInitialApprovalModeForAdd(approvalMode);
    setAddDialogOpen(true);
  }

  function handleAddDialogOpenChange(open: boolean) {
    if (!open) {
      setInitialTemplateForAdd(null);
      setInitialApprovalModeForAdd(undefined);
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
          <EmptyState
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
                  <TableHead className="font-semibold text-primary-foreground">
                    Standing Approval
                  </TableHead>
                  <TableHead className="w-[100px] font-semibold text-primary-foreground" />
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {standingError && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="text-destructive bg-destructive/5 py-2 text-sm"
                    >
                      {standingError}
                    </TableCell>
                  </TableRow>
                )}
                {configs.map((config) => (
                  <ActionConfigRow
                    key={config.id}
                    agentId={agentId}
                    config={config}
                    actions={actions}
                    standingRows={standingByConfig.get(config.id) ?? []}
                    onStandingSuccess={() => {
                      void refetchStanding();
                      onConfigsChanged?.();
                    }}
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
        initialApprovalMode={initialApprovalModeForAdd}
      />

      <RecommendedTemplatesDialog
        open={recommendedDialogOpen}
        onOpenChange={setRecommendedDialogOpen}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        onCustomize={(template, approvalMode) => {
          openAddDialog(template, approvalMode);
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

function EmptyState({
  onAddCustom,
  onBrowseRecommendedTemplates,
  showRecommendedLink,
  actionsDisabled,
}: {
  onAddCustom: () => void;
  onBrowseRecommendedTemplates: () => void;
  showRecommendedLink: boolean;
  actionsDisabled: boolean;
}) {
  return (
    <div className="space-y-4 py-4 text-center">
      <div className="space-y-3">
        <Button size="lg" onClick={onAddCustom} disabled={actionsDisabled}>
          <Plus className="size-4" />
          Add Configuration
        </Button>
        <p className="text-muted-foreground mx-auto max-w-md text-sm">
          Define which actions this agent can use and lock in parameter values
          or mark them as wildcards to give the agent freedom.
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
    </div>
  );
}
