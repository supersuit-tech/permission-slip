import { CreditCard } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { BillingPricing, Plan, PlanLimits, Subscription } from "@/hooks/useBillingPlan";
import { DetailRow } from "./DetailRow";
import { formatDate } from "./formatters";

interface PlanCardProps {
  plan: Plan;
  effectiveLimits: PlanLimits;
  subscription: Subscription;
  pricing?: BillingPricing;
}

function StatusBadge({ status }: { status: Subscription["status"] }) {
  if (status === "active") {
    return <Badge className="bg-emerald-600 text-white">Active</Badge>;
  }
  if (status === "past_due") {
    return <Badge variant="destructive">Past Due</Badge>;
  }
  return <Badge variant="secondary">Cancelled</Badge>;
}

export function PlanCard({ plan, effectiveLimits, subscription, pricing }: PlanCardProps) {
  const isFree = plan.id === "free";
  const isFreePro = plan.id === "free_pro";

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CreditCard className="text-muted-foreground size-5" />
          <CardTitle>Current Plan</CardTitle>
        </div>
        <CardDescription>
          Your subscription and billing period details.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="rounded-lg border p-4 space-y-3">
            <DetailRow label="Plan">
              <div className="flex items-center gap-2">
                <span>{plan.name}</span>
                <Badge variant={isFree ? "outline" : "default"}>
                  {isFree ? "Free" : isFreePro ? "Complimentary Pro" : "Pay-as-you-go"}
                </Badge>
              </div>
            </DetailRow>
            {!isFree && !isFreePro && pricing && (
              <DetailRow label="Pricing">
                <span className="text-muted-foreground">{pricing.free_request_allowance.toLocaleString()} requests/month included, then {pricing.price_per_request_display}/request</span>
              </DetailRow>
            )}
            {isFreePro && (
              <DetailRow label="Pricing">
                <span className="text-muted-foreground">Unlimited usage at no charge</span>
              </DetailRow>
            )}
            <DetailRow label="Status">
              <StatusBadge status={subscription.status} />
            </DetailRow>
            <DetailRow label="Billing Period">
              <span className="text-muted-foreground">
                {formatDate(subscription.current_period_start)} &ndash;{" "}
                {formatDate(subscription.current_period_end)}
              </span>
            </DetailRow>
          </div>
          {subscription.status === "past_due" && (
            <p className="text-sm text-destructive">
              Your payment method failed. Please update it to avoid service interruption.
            </p>
          )}
          {subscription.quota_entitlements_until && (
            <p className="text-amber-700 dark:text-amber-300 text-xs">
              Paid-plan usage limits (requests, agents, approvals, credentials) stay in
              effect until {formatDate(subscription.quota_entitlements_until)}. After that,
              free plan limits apply to new usage; existing resources are not removed.
            </p>
          )}
          {subscription.grace_period_ends_at && (
            <p className="text-amber-700 dark:text-amber-300 text-xs">
              Extended audit log retention stays available until{" "}
              {formatDate(subscription.grace_period_ends_at)} so you can export older
              activity. After that, retention matches the free plan ({plan.audit_retention_days}{" "}
              days).
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
