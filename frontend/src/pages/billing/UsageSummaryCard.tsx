import { BarChart3 } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { BillingPlanResponse, BillingPricing, Plan, UsageSummary } from "@/hooks/useBillingPlan";
import { useBillingUsage } from "@/hooks/useBillingUsage";
import { FREE_REQUEST_ALLOWANCE, PRICE_PER_REQUEST } from "./constants";
import { DetailRow } from "./DetailRow";

type EffectiveLimits = BillingPlanResponse["effective_limits"];

interface UsageSummaryCardProps {
  usage: UsageSummary;
  plan: Plan;
  pricing?: BillingPricing;
  effectiveLimits?: EffectiveLimits;
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
        <div className="h-2 w-full overflow-hidden rounded-full bg-muted" role="progressbar" aria-valuenow={Math.min(current, limit)} aria-valuemin={0} aria-valuemax={limit} aria-label={`${label}: ${current.toLocaleString()} of ${limit.toLocaleString()} used`}>
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

/** Request usage row for paid plans — shows progress against the free allowance. */
function PaidRequestRow({ current, included, priceDisplay }: { current: number; included: number; priceDisplay: string }) {
  const allowance = included;
  const overage = Math.max(0, current - allowance);
  const hasOverage = overage > 0;
  const percentage = Math.min((current / allowance) * 100, 100);

  const label = hasOverage
    ? `${current.toLocaleString()} total (${overage.toLocaleString()} billed)`
    : `${current.toLocaleString()} / ${allowance.toLocaleString()} free`;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">Requests</span>
        <span className="text-muted-foreground tabular-nums">{label}</span>
      </div>
      <div className="flex h-2 w-full overflow-hidden rounded-full bg-muted" role="progressbar" aria-valuenow={Math.min(current, allowance)} aria-valuemin={0} aria-valuemax={allowance} aria-label={`Requests: ${label}`}>
        <div
          className="h-full rounded-l-full transition-all bg-primary"
          style={{ width: `${hasOverage ? ((allowance / current) * 100) : percentage}%` }}
        />
        {hasOverage && (
          <div
            className="h-full rounded-r-full bg-amber-500 transition-all"
            style={{ width: `${(overage / current) * 100}%` }}
          />
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        {hasOverage
          ? `${allowance.toLocaleString()} free + ${overage.toLocaleString()} at ${priceDisplay}/request`
          : `First ${allowance.toLocaleString()} requests/month are free, then ${priceDisplay}/request`}
      </p>
    </div>
  );
}

export function UsageSummaryCard({ usage, plan, pricing, effectiveLimits }: UsageSummaryCardProps) {
  const isPaid = plan.id !== "free" && plan.id !== "free_pro";
  const { usage: usageDetail } = useBillingUsage();
  // Use server-provided included value, fall back to config constant.
  const included = usageDetail?.requests.included ?? FREE_REQUEST_ALLOWANCE;
  const priceDisplay = pricing?.price_per_request_display ?? PRICE_PER_REQUEST;

  // During quota grace period, use effective limits (paid plan limits) instead of
  // the current plan's limits. This ensures the UI shows the limits actually enforced.
  const limits = effectiveLimits ?? plan;
  const hasGrace = !!effectiveLimits;
  // Show paid-style request row when user is on paid plan OR in quota grace period
  // (effective limits have no request cap).
  const showPaidRequests = isPaid || (hasGrace && limits.max_requests_per_month == null);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <BarChart3 className="text-muted-foreground size-5" />
          <CardTitle>Usage</CardTitle>
        </div>
        <CardDescription>
          Current resource usage against your {hasGrace ? "effective" : "plan"} limits.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {showPaidRequests ? (
            <PaidRequestRow current={usage.requests} included={included} priceDisplay={priceDisplay} />
          ) : (
            <UsageRow
              label="Requests"
              current={usage.requests}
              limit={limits.max_requests_per_month ?? null}
            />
          )}
          <UsageRow
            label="Agents"
            current={usage.agents}
            limit={limits.max_agents ?? null}
          />
          <UsageRow
            label="Standing Approvals"
            current={usage.standing_approvals}
            limit={limits.max_standing_approvals ?? null}
          />
          <UsageRow
            label="Credentials"
            current={usage.credentials}
            limit={limits.max_credentials ?? null}
          />
          <DetailRow label="Audit Retention">
            <span className="text-muted-foreground">{plan.audit_retention_days} days</span>
          </DetailRow>
        </div>
      </CardContent>
    </Card>
  );
}
