import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
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
import { UpgradeSuccessBanner } from "./UpgradeSuccessBanner";

export function BillingPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSuccess, setShowSuccess] = useState(
    searchParams.get("upgraded") === "true",
  );

  // Refetch billing data when returning from Stripe with upgraded=true
  // to ensure the plan card reflects the new subscription.
  useEffect(() => {
    if (showSuccess) {
      void refetch();
    }
  }, [showSuccess, refetch]);

  function dismissSuccess() {
    setShowSuccess(false);
    searchParams.delete("upgraded");
    setSearchParams(searchParams, { replace: true });
  }

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

      {showSuccess && <UpgradeSuccessBanner onDismiss={dismissSuccess} />}

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
            <PlanDetailsCard
              subscription={billingPlan.subscription}
              usage={billingPlan.usage}
            />
          )}
        </>
      ) : null}
    </div>
  );
}
