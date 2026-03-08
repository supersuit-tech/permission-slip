import { useCallback, useEffect, useRef, useState } from "react";
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

// Delays (in ms) between each poll attempt while waiting for the webhook
// to activate the paid plan after Stripe checkout.
const POLL_DELAYS = [1000, 2000, 3000, 5000];

export function BillingPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSuccess, setShowSuccess] = useState(
    searchParams.get("upgraded") === "true",
  );
  const pollRef = useRef(false);

  const isPaidPlan = billingPlan != null && billingPlan.plan.id !== "free";

  // Poll for the plan change after returning from Stripe checkout.
  // The Stripe webhook that upgrades the plan is async, so we retry a
  // few times with increasing delays until the plan reflects the upgrade.
  const pollForUpgrade = useCallback(async () => {
    if (pollRef.current) return;
    pollRef.current = true;
    for (const delay of POLL_DELAYS) {
      await new Promise((r) => setTimeout(r, delay));
      const result = await refetch();
      if (result.data?.plan.id !== "free") break;
    }
    pollRef.current = false;
  }, [refetch]);

  // When returning from Stripe with upgraded=true, kick off an immediate
  // refetch plus background polling to wait for the webhook.
  useEffect(() => {
    if (!showSuccess) return;
    void refetch().then((result) => {
      if (result.data?.plan.id === "free") {
        void pollForUpgrade();
      }
    });
  }, [showSuccess, refetch, pollForUpgrade]);

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

      {showSuccess && (
        <UpgradeSuccessBanner onDismiss={dismissSuccess} upgraded={isPaidPlan} />
      )}

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
