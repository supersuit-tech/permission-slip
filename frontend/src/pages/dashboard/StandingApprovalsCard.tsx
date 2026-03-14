import { useMemo, useState } from "react";
import { Loader2, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LimitBadge } from "@/components/LimitBadge";
import { UpgradePrompt } from "@/components/UpgradePrompt";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardFooter,
} from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  useStandingApprovals,
  type StandingApproval,
} from "@/hooks/useStandingApprovals";
import { useAgents, type Agent } from "@/hooks/useAgents";
import type { ActionConfiguration } from "@/hooks/useActionConfigs";
import { useActionConfigMap } from "@/hooks/useActionConfigMap";
import { useResourceLimit } from "@/hooks/useResourceLimit";
import { getAgentDisplayName } from "@/lib/agents";
import { RevokeStandingApprovalDialog } from "./RevokeStandingApprovalDialog";
import { CreateStandingApprovalDialog } from "./CreateStandingApprovalDialog";
import { ConstraintsSummary } from "./ConstraintsSummary";

function formatExpiresIn(expiresAt: string | null | undefined): string {
  if (!expiresAt) return "\u2014";

  const exp = new Date(expiresAt);
  if (Number.isNaN(exp.getTime())) return "\u2014";

  const now = new Date();
  const diffMs = exp.getTime() - now.getTime();

  if (diffMs <= 0) return "Expired";

  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  if (diffHours < 24) return `${diffHours}h`;

  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d`;
}

function ExecutionBadge({
  executionCount,
  maxExecutions,
}: {
  executionCount: number;
  maxExecutions?: number | null;
}) {
  if (maxExecutions == null) {
    return (
      <span className="text-muted-foreground text-xs">
        {executionCount} / &infin;
      </span>
    );
  }
  const ratio = executionCount / maxExecutions;
  const variant = ratio >= 0.9 ? "destructive" : "secondary";
  return (
    <Badge variant={variant}>
      {executionCount} / {maxExecutions}
    </Badge>
  );
}

function resolveAgentName(
  agentId: number,
  agentMap: Map<number, Agent>,
): string {
  const agent = agentMap.get(agentId);
  return agent ? getAgentDisplayName(agent) : `Agent ${agentId}`;
}

function resolveActionConfigName(
  sourceId: string | null | undefined,
  configMap: Map<string, ActionConfiguration>,
): string | null {
  if (!sourceId) return null;
  const config = configMap.get(sourceId);
  return config?.name ?? null;
}

function StandingApprovalRow({
  sa,
  onRevoke,
  agentMap,
  configMap,
}: {
  sa: StandingApproval;
  onRevoke: (sa: StandingApproval) => void;
  agentMap: Map<number, Agent>;
  configMap: Map<string, ActionConfiguration>;
}) {
  const agentName = resolveAgentName(sa.agent_id, agentMap);
  const configName = resolveActionConfigName(
    sa.source_action_configuration_id,
    configMap,
  );

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div>
          <span>{sa.action_type}</span>
          {configName && (
            <span className="text-muted-foreground block text-xs">
              {configName}
            </span>
          )}
        </div>
      </TableCell>
      <TableCell className="max-w-[200px] truncate text-xs">
        {agentName}
      </TableCell>
      <TableCell>
        <ConstraintsSummary constraints={sa.constraints} />
      </TableCell>
      <TableCell>
        <ExecutionBadge
          executionCount={sa.execution_count}
          maxExecutions={sa.max_executions}
        />
      </TableCell>
      <TableCell>{formatExpiresIn(sa.expires_at)}</TableCell>
      <TableCell className="text-right">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onRevoke(sa)}
        >
          Revoke
        </Button>
      </TableCell>
    </TableRow>
  );
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <ShieldCheck className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No standing approvals yet
      </p>
      <p className="text-muted-foreground mb-4 max-w-xs text-xs">
        Tired of approving the same action? Create a standing approval to
        let an agent repeat it automatically — with limits you control.
      </p>
      <Button size="sm" onClick={onCreate}>
        Create Standing Approval
      </Button>
    </div>
  );
}

export function StandingApprovalsCard() {
  const { standingApprovals, isLoading, error, refetch } =
    useStandingApprovals();
  const { agents } = useAgents();
  const agentIds = useMemo(
    () => standingApprovals.map((sa) => sa.agent_id),
    [standingApprovals],
  );
  const configMap = useActionConfigMap(agentIds);
  const agentMap = useMemo(() => {
    const map = new Map<number, Agent>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);
  const {
    max: maxStandingApprovals,
    current: standingApprovalCount,
    atLimit,
    hasData: hasBillingData,
  } = useResourceLimit("max_standing_approvals", standingApprovals.length);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<StandingApproval | null>(
    null,
  );

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CardTitle>Standing Approvals</CardTitle>
          {hasBillingData && (
            <LimitBadge
              current={standingApprovalCount}
              max={maxStandingApprovals}
              resource="standing approvals"
            />
          )}
        </div>
      </CardHeader>
      <CardContent className="px-3 md:px-6">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <p className="text-destructive mb-2 text-sm">{error}</p>
            <Button variant="ghost" size="sm" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        ) : standingApprovals.length === 0 ? (
          <EmptyState onCreate={() => setCreateDialogOpen(true)} />
        ) : (
          <StandingApprovalsTable
            approvals={standingApprovals}
            onRevoke={(sa) => setRevokeTarget(sa)}
            agentMap={agentMap}
            configMap={configMap}
          />
        )}
      </CardContent>
      {atLimit ? (
        <CardFooter>
          <UpgradePrompt feature="Upgrade to create more standing approvals." />
        </CardFooter>
      ) : standingApprovals.length > 0 ? (
        <CardFooter>
          <Button
            className="w-full sm:w-auto"
            onClick={() => setCreateDialogOpen(true)}
          >
            Create Standing Approval
          </Button>
        </CardFooter>
      ) : null}

      <CreateStandingApprovalDialog
        agents={agents}
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />

      {revokeTarget && (
        <RevokeStandingApprovalDialog
          standingApprovalId={revokeTarget.standing_approval_id}
          actionType={revokeTarget.action_type}
          agentId={revokeTarget.agent_id}
          open={!!revokeTarget}
          onOpenChange={(open) => {
            if (!open) setRevokeTarget(null);
          }}
        />
      )}
    </Card>
  );
}

function StandingApprovalsTable({
  approvals,
  onRevoke,
  agentMap,
  configMap,
}: {
  approvals: StandingApproval[];
  onRevoke: (sa: StandingApproval) => void;
  agentMap: Map<number, Agent>;
  configMap: Map<string, ActionConfiguration>;
}) {
  return (
    <div className="-mx-3 overflow-x-auto sm:mx-0">
      <div className="min-w-[600px] overflow-hidden rounded-lg sm:min-w-0">
        <Table>
          <TableHeader>
            <TableRow className="border-none bg-primary hover:bg-primary">
              <TableHead className="font-semibold text-primary-foreground">
                Action
              </TableHead>
              <TableHead className="font-semibold text-primary-foreground">
                Agent
              </TableHead>
              <TableHead className="font-semibold text-primary-foreground">
                Constraints
              </TableHead>
              <TableHead className="font-semibold text-primary-foreground">
                Executions
              </TableHead>
              <TableHead className="font-semibold text-primary-foreground">
                Expires
              </TableHead>
              <TableHead className="font-semibold text-primary-foreground">
                <span className="sr-only">Actions</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody className="[&>tr:nth-child(even)]:bg-muted">
            {approvals.map((sa) => (
              <StandingApprovalRow
                key={sa.standing_approval_id}
                sa={sa}
                onRevoke={onRevoke}
                agentMap={agentMap}
                configMap={configMap}
              />
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
