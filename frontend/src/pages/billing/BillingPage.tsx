import { Link } from "react-router-dom";
import { ArrowLeft, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import { PlanCard } from "./PlanCard";
import { UsageSummaryCard } from "./UsageSummaryCard";
import { UpgradeCTA } from "./UpgradeCTA";
import { PlanDetailsCard } from "./PlanDetailsCard";

export function BillingPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/"
          className="text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Back to Dashboard"
        >
          <ArrowLeft className="size-5" />
        </Link>
        <h1 className="text-2xl font-bold tracking-tight">Billing</h1>
      </div>

      {isLoading ? (
        <div
          className="flex items-center justify-center py-16"
          role="status"
          aria-label="Loading billing information"
        >
          <Loader2 className="text-muted-foreground size-5 animate-spin" />
        </div>
      ) : error ? (
        <div className="space-y-3">
          <FormError error={error} />
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            Try Again
          </Button>
        </div>
      ) : billingPlan ? (
        <>
          <PlanCard
            plan={billingPlan.plan}
            subscription={billingPlan.subscription}
          />
          <UsageSummaryCard
            usage={billingPlan.usage}
            plan={billingPlan.plan}
          />
          {billingPlan.subscription.can_upgrade && <UpgradeCTA />}
          {billingPlan.subscription.can_downgrade && (
            <PlanDetailsCard subscription={billingPlan.subscription} />
          )}
        </>
      ) : null}
    </div>
  );
}
