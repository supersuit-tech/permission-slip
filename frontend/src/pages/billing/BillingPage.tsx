import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import { PlanCard } from "./PlanCard";
import { UsageSummaryCard } from "./UsageSummaryCard";
import { UpgradeCTA } from "./UpgradeCTA";
import { UsageDashboard } from "./UsageDashboard";
import { PlanDetailsCard } from "./PlanDetailsCard";
import { BillingPageSkeleton } from "./BillingPageSkeleton";

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
        <BillingPageSkeleton />
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
          <UsageDashboard
            plan={billingPlan.plan}
            subscription={billingPlan.subscription}
          />
          {billingPlan.subscription.can_upgrade && (
            <UpgradeCTA plan={billingPlan.plan} />
          )}
          {billingPlan.subscription.can_downgrade && (
            <PlanDetailsCard subscription={billingPlan.subscription} />
          )}
        </>
      ) : null}
    </div>
  );
}
