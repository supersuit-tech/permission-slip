import { BarChart3 } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { Plan, UsageSummary } from "@/hooks/useBillingPlan";

interface UsageSummaryCardProps {
  usage: UsageSummary;
  plan: Plan;
}

interface UsageRowProps {
  label: string;
  current: number;
  limit: number | null;
}

function UsageRow({ label, current, limit }: UsageRowProps) {
  const isUnlimited = limit === null;
  const percentage = isUnlimited ? 0 : limit > 0 ? Math.min((current / limit) * 100, 100) : 0;
  const isNearLimit = !isUnlimited && limit > 0 && percentage >= 80;
  const isAtLimit = !isUnlimited && limit > 0 && current >= limit;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">{label}</span>
        <span className="text-muted-foreground tabular-nums">
          {current.toLocaleString()} / {isUnlimited ? "Unlimited" : limit.toLocaleString()}
        </span>
      </div>
      {!isUnlimited && (
        <div className="h-2 w-full overflow-hidden rounded-full bg-muted" role="progressbar" aria-valuenow={current} aria-valuemin={0} aria-valuemax={limit}>
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

export function UsageSummaryCard({ usage, plan }: UsageSummaryCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <BarChart3 className="text-muted-foreground size-5" />
          <CardTitle>Usage</CardTitle>
        </div>
        <CardDescription>
          Current resource usage against your plan limits.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <UsageRow
            label="Requests"
            current={usage.requests}
            limit={plan.max_requests_per_month ?? null}
          />
          <UsageRow
            label="Agents"
            current={usage.agents}
            limit={plan.max_agents ?? null}
          />
          <UsageRow
            label="Standing Approvals"
            current={usage.standing_approvals}
            limit={plan.max_standing_approvals ?? null}
          />
          <UsageRow
            label="Credentials"
            current={usage.credentials}
            limit={plan.max_credentials ?? null}
          />
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium">Audit Retention</span>
            <span className="text-muted-foreground">
              {plan.audit_retention_days} days
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
