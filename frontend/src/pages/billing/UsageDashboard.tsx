import { useMemo } from "react";
import { Link } from "react-router-dom";
import { Activity, Clock, Loader2, MessageSquare } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useBillingUsage } from "@/hooks/useBillingUsage";
import { useAgents } from "@/hooks/useAgents";
import type { Plan, Subscription } from "@/hooks/useBillingPlan";
import { formatCents, formatDate } from "./formatters";

interface UsageDashboardProps {
  plan: Plan;
  subscription: Subscription;
}

function daysRemaining(periodEnd: string): number {
  const end = new Date(periodEnd);
  const now = new Date();
  const diff = end.getTime() - now.getTime();
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)));
}

function RequestUsageBar({
  total,
  limit,
}: {
  total: number;
  limit: number | null;
}) {
  const isUnlimited = limit === null;
  const percentage = isUnlimited
    ? 0
    : limit > 0
      ? Math.min((total / limit) * 100, 100)
      : 0;
  const isNearLimit = !isUnlimited && limit > 0 && percentage >= 80;
  const isAtLimit = !isUnlimited && limit > 0 && total >= limit;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">Requests this period</span>
        <span className="text-muted-foreground tabular-nums">
          {total.toLocaleString()} /{" "}
          {isUnlimited ? "Unlimited" : limit.toLocaleString()}
        </span>
      </div>
      {!isUnlimited && (
        <div
          className="h-2.5 w-full overflow-hidden rounded-full bg-muted"
          role="progressbar"
          aria-valuenow={Math.min(total, limit)}
          aria-valuemin={0}
          aria-valuemax={limit}
          aria-label={`Requests: ${total.toLocaleString()} of ${limit.toLocaleString()} used`}
        >
          <div
            className={`h-full rounded-full transition-all ${
              isAtLimit
                ? "bg-destructive"
                : isNearLimit
                  ? "bg-amber-500"
                  : "bg-primary"
            }`}
            style={{ width: `${percentage}%` }}
          />
        </div>
      )}
    </div>
  );
}

interface AgentBreakdownTableProps {
  byAgent: Record<string, number>;
  totalRequests: number;
  showCost: boolean;
  costCents: number;
}

function AgentBreakdownTable({
  byAgent,
  totalRequests,
  showCost,
  costCents,
}: AgentBreakdownTableProps) {
  const { agents } = useAgents();

  const agentNameMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const agent of agents) {
      const meta = agent.metadata as Record<string, unknown> | undefined;
      const name =
        typeof meta?.name === "string" ? meta.name : `Agent ${agent.agent_id}`;
      map.set(String(agent.agent_id), name);
    }
    return map;
  }, [agents]);

  const sorted = Object.entries(byAgent).sort(([, a], [, b]) => b - a);
  const maxCount = sorted.length > 0 ? (sorted[0]?.[1] ?? 0) : 0;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Agent</TableHead>
          <TableHead className="text-right">Requests</TableHead>
          <TableHead className="text-right">% of total</TableHead>
          {showCost && <TableHead className="text-right">Est. cost</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {sorted.map(([agentId, count]) => {
          const pctNum =
            totalRequests > 0 ? (count / totalRequests) * 100 : 0;
          const pct = pctNum.toFixed(1);
          const barWidth =
            maxCount > 0 ? Math.max(2, (count / maxCount) * 100) : 0;
          const agentCost =
            totalRequests > 0
              ? Math.round((count / totalRequests) * costCents)
              : 0;

          return (
            <TableRow key={agentId}>
              <TableCell className="font-medium">
                <div className="space-y-1">
                  <Link
                    to={`/agents/${agentId}`}
                    className="text-primary hover:underline"
                  >
                    {agentNameMap.get(agentId) ?? `Agent ${agentId}`}
                  </Link>
                  <div className="h-1 w-full max-w-[120px] overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary/40"
                      style={{ width: `${barWidth}%` }}
                    />
                  </div>
                </div>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {count.toLocaleString()}
              </TableCell>
              <TableCell className="text-right tabular-nums text-muted-foreground">
                {pct}%
              </TableCell>
              {showCost && (
                <TableCell className="text-right tabular-nums text-muted-foreground">
                  {formatCents(agentCost)}
                </TableCell>
              )}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

export function UsageDashboard({ plan, subscription }: UsageDashboardProps) {
  const { usage, isLoading, error } = useBillingUsage();
  const isFree = plan.id === "free";
  const days = daysRemaining(subscription.current_period_end);
  const periodLabel = `${formatDate(subscription.current_period_start)} — ${formatDate(subscription.current_period_end)}`;

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Activity className="text-muted-foreground size-5" />
            <CardTitle>Usage Details</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="text-muted-foreground size-5 animate-spin" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error || !usage) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Activity className="text-muted-foreground size-5" />
            <CardTitle>Usage Details</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            {error ?? "No usage data available for this period."}
          </p>
        </CardContent>
      </Card>
    );
  }

  const hasBreakdown =
    usage.breakdown?.by_agent &&
    Object.keys(usage.breakdown.by_agent).length > 0;

  const totalCostCents = usage.requests.cost_cents + usage.sms.cost_cents;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Activity className="text-muted-foreground size-5" />
          <CardTitle>Usage Details</CardTitle>
        </div>
        <CardDescription>{periodLabel}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {/* Usage overview */}
          <div className="space-y-4">
            <RequestUsageBar
              total={usage.requests.total}
              limit={plan.max_requests_per_month ?? null}
            />

            <div className="grid grid-cols-2 gap-4">
              <div className="rounded-lg border p-3 space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Clock className="size-3.5" />
                  Days remaining
                </div>
                <p className="text-lg font-semibold tabular-nums">{days}</p>
              </div>

              {isFree ? (
                <div className="rounded-lg border p-3 space-y-1">
                  <p className="text-xs text-muted-foreground">
                    Requests remaining
                  </p>
                  <p className="text-lg font-semibold tabular-nums">
                    {plan.max_requests_per_month != null
                      ? Math.max(
                          0,
                          plan.max_requests_per_month - usage.requests.total,
                        ).toLocaleString()
                      : "—"}
                  </p>
                </div>
              ) : (
                <div className="rounded-lg border p-3 space-y-1">
                  <p className="text-xs text-muted-foreground">
                    Estimated cost
                  </p>
                  <p className="text-lg font-semibold tabular-nums">
                    {formatCents(totalCostCents)}
                  </p>
                </div>
              )}
            </div>

            {/* Overage notice for free tier */}
            {isFree && usage.requests.overage > 0 && (
              <p className="text-sm text-destructive">
                You have exceeded your monthly limit by{" "}
                {usage.requests.overage.toLocaleString()} requests. Upgrade to
                continue making requests.
              </p>
            )}
          </div>

          {/* Per-agent breakdown */}
          {hasBreakdown && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Usage by Agent</h3>
              <AgentBreakdownTable
                byAgent={usage.breakdown!.by_agent!}
                totalRequests={usage.requests.total}
                showCost={!isFree}
                costCents={totalCostCents}
              />
            </div>
          )}

          {/* SMS usage (paid tier only) */}
          {!isFree && usage.sms.total > 0 && (
            <div className="space-y-2">
              <h3 className="flex items-center gap-1.5 text-sm font-medium">
                <MessageSquare className="size-4" />
                SMS Usage
              </h3>
              <div className="rounded-lg border p-3 space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Messages sent</span>
                  <span className="tabular-nums">
                    {usage.sms.total.toLocaleString()}
                  </span>
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">SMS cost</span>
                  <span className="tabular-nums">
                    {formatCents(usage.sms.cost_cents)}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
