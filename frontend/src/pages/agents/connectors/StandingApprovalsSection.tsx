import { useMemo, useState } from "react";
import { Loader2, Plus, ShieldCheck } from "lucide-react";
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
import {
  useStandingApprovals,
  type StandingApproval,
} from "@/hooks/useStandingApprovals";
import { useStandingApprovalTemplates } from "@/hooks/useStandingApprovalTemplates";
import type { StandingApprovalTemplate } from "@/hooks/useStandingApprovalTemplates";
import { useAgentConnectorInstances } from "@/hooks/useAgentConnectorInstances";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { StandingApprovalRow } from "./StandingApprovalRow";
import { AddStandingApprovalDialog } from "./AddStandingApprovalDialog";
import { EditStandingApprovalDialog } from "./EditStandingApprovalDialog";
import { RevokeStandingApprovalDialog } from "./RevokeStandingApprovalDialog";
import { RecommendedTemplatesDialog } from "./RecommendedTemplatesDialog";
import { templateIsApplied } from "./templateMatching";

interface StandingApprovalsSectionProps {
  agentId: number;
  connectorId: string;
  connectorName: string;
  actions: ConnectorAction[];
}

export function StandingApprovalsSection({
  agentId,
  connectorId,
  actions,
}: StandingApprovalsSectionProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [initialTemplateForAdd, setInitialTemplateForAdd] =
    useState<StandingApprovalTemplate | null>(null);
  const [recommendedDialogOpen, setRecommendedDialogOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<StandingApproval | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<StandingApproval | null>(
    null,
  );

  const {
    standingApprovals,
    isLoading,
    error,
    refetch,
  } = useStandingApprovals({ status: "all" });

  const { templates, isLoading: templatesLoading } =
    useStandingApprovalTemplates(connectorId);

  const actionTypeSet = useMemo(
    () => new Set(actions.map((a) => a.action_type)),
    [actions],
  );

  const connectorRules = useMemo(
    () =>
      standingApprovals.filter(
        (sa) => sa.agent_id === agentId && actionTypeSet.has(sa.action_type),
      ),
    [standingApprovals, agentId, actionTypeSet],
  );

  const activeRules = useMemo(
    () => connectorRules.filter((sa) => sa.status === "active"),
    [connectorRules],
  );

  const hasRecommendedTemplates =
    !templatesLoading &&
    templates.some(
      (t) =>
        actionTypeSet.has(t.action_type) &&
        !templateIsApplied(t, activeRules),
    );

  const { instances } = useAgentConnectorInstances(agentId, connectorId);
  const showAccountColumn = instances.length >= 1;

  function openAddDialog(fromTemplate?: StandingApprovalTemplate | null) {
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
          <ShieldCheck className="text-muted-foreground size-5" />
          <CardTitle>Standing Approvals</CardTitle>
        </div>
        {connectorRules.length > 0 && (
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
              Add Standing Approval
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
        ) : connectorRules.length === 0 ? (
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
                    Constraints
                  </TableHead>
                  {showAccountColumn && (
                    <TableHead className="font-semibold text-primary-foreground">
                      Account
                    </TableHead>
                  )}
                  <TableHead className="font-semibold text-primary-foreground">
                    Expires
                  </TableHead>
                  <TableHead className="font-semibold text-primary-foreground">
                    Status
                  </TableHead>
                  <TableHead className="w-[100px] font-semibold text-primary-foreground" />
                </TableRow>
              </TableHeader>
              <TableBody className="[&>tr:nth-child(even)]:bg-muted">
                {connectorRules.map((rule) => (
                  <StandingApprovalRow
                    key={rule.standing_approval_id}
                    agentId={agentId}
                    rule={rule}
                    actions={actions}
                    instances={instances}
                    showAccountColumn={showAccountColumn}
                    onEdit={setEditTarget}
                    onRevoke={setRevokeTarget}
                    onChanged={() => void refetch()}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      <AddStandingApprovalDialog
        open={addDialogOpen}
        onOpenChange={handleAddDialogOpenChange}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        initialTemplate={initialTemplateForAdd}
        onCreated={() => void refetch()}
      />

      <RecommendedTemplatesDialog
        open={recommendedDialogOpen}
        onOpenChange={setRecommendedDialogOpen}
        agentId={agentId}
        connectorId={connectorId}
        actions={actions}
        existingRules={activeRules}
        onCustomize={(template) => {
          openAddDialog(template);
        }}
      />

      {editTarget && (
        <EditStandingApprovalDialog
          open={!!editTarget}
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
          rule={editTarget}
          agentId={agentId}
          connectorId={connectorId}
          actions={actions}
          onUpdated={() => void refetch()}
        />
      )}

      {revokeTarget && (
        <RevokeStandingApprovalDialog
          open={!!revokeTarget}
          onOpenChange={(open) => {
            if (!open) setRevokeTarget(null);
          }}
          rule={revokeTarget}
          onRevoked={() => void refetch()}
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
          Add Standing Approval
        </Button>
        <p className="text-muted-foreground mx-auto max-w-md text-sm">
          Every request from this agent will ask for your approval. Add a
          standing approval to pre-authorize trusted, repetitive actions.
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
