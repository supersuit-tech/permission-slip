import { Activity, Clock, Loader2, MessageSquare } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useBillingUsage } from "@/hooks/useBillingUsage";
import type { Plan, Subscription } from "@/hooks/useBillingPlan";
import { formatCents, formatDate } from "./formatters";
import { RequestUsageBar } from "./RequestUsageBar";
import { AgentBreakdownTable } from "./AgentBreakdownTable";

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

  const byAgent = usage.breakdown?.by_agent;
  const hasBreakdown = byAgent != null && Object.keys(byAgent).length > 0;
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
                byAgent={byAgent}
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
