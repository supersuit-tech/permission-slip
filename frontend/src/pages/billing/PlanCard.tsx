import { CreditCard } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { BillingPricing, Plan, Subscription } from "@/hooks/useBillingPlan";
import { DetailRow } from "./DetailRow";
import { formatDate } from "./formatters";

interface PlanCardProps {
  plan: Plan;
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

export function PlanCard({ plan, subscription, pricing }: PlanCardProps) {
  const isFree = plan.id === "free";

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
                <span className="text-foreground">{plan.name}</span>
                <Badge variant={isFree ? "outline" : "default"}>
                  {isFree ? "Free" : "Pay-as-you-go"}
                </Badge>
              </div>
            </DetailRow>
            {!isFree && pricing && (
              <DetailRow label="Pricing">
                {pricing.free_request_allowance.toLocaleString()} requests/month included, then {pricing.price_per_request_display}/request
              </DetailRow>
            )}
            <DetailRow label="Status">
              <StatusBadge status={subscription.status} />
            </DetailRow>
            <DetailRow label="Billing Period">
              {formatDate(subscription.current_period_start)} &ndash;{" "}
              {formatDate(subscription.current_period_end)}
            </DetailRow>
          </div>
          {subscription.status === "past_due" && (
            <p className="text-sm text-destructive">
              Your payment method failed. Please update it to avoid service interruption.
            </p>
          )}
          {subscription.grace_period_ends_at && (
            <p className="text-amber-700 dark:text-amber-300 text-xs">
              Your paid plan features are preserved until{" "}
              {formatDate(subscription.grace_period_ends_at)}. After that,
              your account will revert to free plan limits.
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
