import { useState, useMemo, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import { Loader2, Bot, Settings, Eye } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CopyableCode } from "@/components/CopyableCode";
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
import { AgentStatusBadge } from "@/components/AgentStatusBadge";
import { formatRelativeTime } from "@/lib/utils";
import { useAgents, type Agent } from "@/hooks/useAgents";
import { useResourceLimit } from "@/hooks/useResourceLimit";
import { getAgentDisplayName } from "@/lib/agents";
import { InviteCodeDialog } from "./InviteCodeDialog";
import { ReviewPendingAgentDialog } from "./ReviewPendingAgentDialog";
import { RegistrationSuccessDialog } from "./RegistrationSuccessDialog";

type StatusFilter = "active" | "all" | "pending" | "deactivated";

/**
 * Detects when a previously-pending agent transitions to "registered" so we
 * can show the post-registration instructions dialog automatically.
 */
function useRegistrationSuccess(agents: Agent[]) {
  const prevPendingIdsRef = useRef<Set<number>>(new Set());
  const [newlyRegistered, setNewlyRegistered] = useState<Agent | null>(null);

  useEffect(() => {
    const currentPendingIds = new Set(
      agents.filter((a) => a.status === "pending").map((a) => a.agent_id),
    );

    // On the first render, just record the pending set — don't trigger a dialog.
    // Also skip detection if a success dialog is already showing so we don't
    // overwrite it mid-display; the next transition will be caught after dismiss.
    if (
      !newlyRegistered &&
      (prevPendingIdsRef.current.size > 0 || currentPendingIds.size > 0)
    ) {
      for (const agent of agents) {
        if (
          agent.status === "registered" &&
          prevPendingIdsRef.current.has(agent.agent_id)
        ) {
          setNewlyRegistered(agent);
          break;
        }
      }
    }

    prevPendingIdsRef.current = currentPendingIds;
  }, [agents, newlyRegistered]);

  return {
    newlyRegistered,
    clearNewlyRegistered: () => setNewlyRegistered(null),
  };
}

function EmptyState({ onAddAgent }: { onAddAgent: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <Bot className="text-muted-foreground mb-3 size-10" />
      <p className="text-muted-foreground mb-1 text-sm font-medium">
        No registered agents yet
      </p>
      <p className="text-muted-foreground mb-4 max-w-xs text-xs">
        Register an agent to start controlling what it can do. You&apos;ll get
        an invite code to share with the agent.
      </p>
      <Button size="sm" onClick={onAddAgent}>
        Add an Agent
      </Button>
    </div>
  );
}

function AgentRow({ agent }: { agent: Agent }) {
  const isPending = agent.status === "pending";
  const [reviewOpen, setReviewOpen] = useState(false);

  return (
    <TableRow>
      <TableCell className="font-medium">
        {getAgentDisplayName(agent)}
      </TableCell>
      <TableCell>
        <div className="flex flex-col gap-1">
          <AgentStatusBadge status={agent.status} />
          {isPending && agent.confirmation_code && (
            <CopyableCode code={agent.confirmation_code} />
          )}
        </div>
      </TableCell>
      <TableCell>
        {isPending
          ? agent.expires_at
            ? `Expires ${formatRelativeTime(agent.expires_at)}`
            : "Awaiting verification"
          : formatRelativeTime(agent.last_active_at)}
      </TableCell>
      <TableCell>{isPending ? "-" : (agent.request_count_30d ?? 0)}</TableCell>
      <TableCell className="text-right">
        {isPending ? (
          <>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setReviewOpen(true)}
            >
              <Eye className="mr-1 size-4" />
              Review
            </Button>
            <ReviewPendingAgentDialog
              agent={agent}
              open={reviewOpen}
              onOpenChange={setReviewOpen}
            />
          </>
        ) : (
          <Button variant="ghost" size="sm" asChild>
            <Link to={`/agents/${agent.agent_id}`}>
              <Settings className="mr-1 size-4" />
              Configure
            </Link>
          </Button>
        )}
      </TableCell>
    </TableRow>
  );
}

function StatusFilterTabs({
  value,
  onChange,
}: {
  value: StatusFilter;
  onChange: (v: StatusFilter) => void;
}) {
  const filters: { label: string; value: StatusFilter }[] = [
    { label: "Active", value: "active" },
    { label: "Pending", value: "pending" },
    { label: "Deactivated", value: "deactivated" },
    { label: "All", value: "all" },
  ];

  return (
    <div className="flex gap-1" role="tablist" aria-label="Filter agents by status">
      {filters.map((f) => (
        <Button
          key={f.value}
          variant={value === f.value ? "default" : "ghost"}
          size="sm"
          role="tab"
          aria-selected={value === f.value}
          onClick={() => onChange(f.value)}
        >
          {f.label}
        </Button>
      ))}
    </div>
  );
}

export function RegisteredAgentsCard() {
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("active");
  const { agents, isLoading, error, refetch } = useAgents();
  const { max: maxAgents, current: agentCount, atLimit, hasData: hasBillingData } =
    useResourceLimit("max_agents", agents.length);
  const { newlyRegistered, clearNewlyRegistered } =
    useRegistrationSuccess(agents);

  const filteredAgents = useMemo(() => {
    if (statusFilter === "all") return agents;
    const statusMap: Record<StatusFilter, string> = {
      active: "registered",
      pending: "pending",
      deactivated: "deactivated",
      all: "",
    };
    const targetStatus = statusMap[statusFilter];
    return agents.filter((a) => a.status === targetStatus);
  }, [agents, statusFilter]);

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <CardTitle>Registered Agents</CardTitle>
            {hasBillingData && (
              <LimitBadge
                current={agentCount}
                max={maxAgents}
                resource="agents"
              />
            )}
          </div>
          {agents.length > 0 && (
            <StatusFilterTabs value={statusFilter} onChange={setStatusFilter} />
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
        ) : agents.length === 0 ? (
          <EmptyState onAddAgent={() => setInviteDialogOpen(true)} />
        ) : filteredAgents.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <p className="text-muted-foreground text-sm">
              No {statusFilter} agents found.
            </p>
          </div>
        ) : (
          <AgentsTable agents={filteredAgents} />
        )}
      </CardContent>
      {atLimit ? (
        <CardFooter>
          <UpgradePrompt feature="Upgrade to add more agents." />
        </CardFooter>
      ) : agents.length > 0 ? (
        <CardFooter>
          <Button
            className="w-full sm:w-auto"
            onClick={() => setInviteDialogOpen(true)}
          >
            Add an Agent
          </Button>
        </CardFooter>
      ) : null}

      <InviteCodeDialog
        open={inviteDialogOpen}
        onOpenChange={(open) => {
          setInviteDialogOpen(open);
          if (!open) refetch();
        }}
      />

      {newlyRegistered && (
        <RegistrationSuccessDialog
          agent={newlyRegistered}
          open={true}
          onOpenChange={(open) => {
            if (!open) clearNewlyRegistered();
          }}
        />
      )}
    </Card>
  );
}

function AgentsTable({ agents }: { agents: Agent[] }) {
  return (
    <div className="overflow-hidden rounded-lg">
      <Table>
        <TableHeader>
          <TableRow className="border-none bg-primary hover:bg-primary">
            <TableHead className="font-semibold text-primary-foreground">
              Agent
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              Status
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              Last Active
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              Requests
            </TableHead>
            <TableHead className="font-semibold text-primary-foreground">
              <span className="sr-only">Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody className="[&>tr:nth-child(even)]:bg-muted">
          {agents.map((agent) => (
            <AgentRow key={agent.agent_id} agent={agent} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
