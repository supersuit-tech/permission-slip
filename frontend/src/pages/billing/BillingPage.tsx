import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { useAuth } from "@/auth/AuthContext";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import { PlanCard } from "./PlanCard";
import { UsageSummaryCard } from "./UsageSummaryCard";
import { UpgradeCTA } from "./UpgradeCTA";
import { UsageDashboard } from "./UsageDashboard";
import { PlanDetailsCard } from "./PlanDetailsCard";
import { BillingPageSkeleton } from "./BillingPageSkeleton";
import { UpgradeSuccessBanner } from "./UpgradeSuccessBanner";
import { activateUpgrade } from "./activateUpgrade";
import { CouponRedemptionCard } from "./CouponRedemptionCard";

// Retry delays (ms) after activate call if plan hasn't changed yet.
const RETRY_DELAYS = [1000, 2000, 3000];

export function BillingPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();
  const { session } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSuccess, setShowSuccess] = useState(
    searchParams.get("upgraded") === "true",
  );
  const activateRef = useRef(false);

  const isPaidPlan =
    billingPlan != null &&
    billingPlan.plan.id !== "free" &&
    billingPlan.plan.id !== "free_pro";

  // Retry refetch a few times if the plan hasn't changed after activate.
  const retryUntilUpgraded = useCallback(async () => {
    for (const delay of RETRY_DELAYS) {
      await new Promise((r) => setTimeout(r, delay));
      const result = await refetch();
      if (
        result.data?.plan.id !== "free" &&
        result.data?.plan.id !== "free_pro"
      )
        return;
    }
  }, [refetch]);

  // After returning from Stripe checkout, call the activate endpoint to
  // confirm the upgrade directly with Stripe. This doesn't rely on
  // webhooks, so it works even in test/dev environments.
  useEffect(() => {
    if (!showSuccess || activateRef.current) return;
    const sessionId = searchParams.get("session_id");
    if (!sessionId || !session?.access_token) {
      // No session ID (old URL format) — fall back to a simple refetch.
      void refetch();
      return;
    }
    activateRef.current = true;
    void activateUpgrade(sessionId, session.access_token)
      .then(() => refetch())
      .then((result) => {
        if (
          result?.data?.plan.id === "free" ||
          result?.data?.plan.id === "free_pro"
        ) {
          void retryUntilUpgraded();
        }
      });
  }, [showSuccess, searchParams, session?.access_token, refetch, retryUntilUpgraded]);

  function dismissSuccess() {
    setShowSuccess(false);
    const next = new URLSearchParams(searchParams);
    next.delete("upgraded");
    next.delete("session_id");
    setSearchParams(next, { replace: true });
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
            effectiveLimits={billingPlan.effective_limits}
            subscription={billingPlan.subscription}
            pricing={billingPlan.pricing}
          />
          <UsageSummaryCard
            usage={billingPlan.usage}
            plan={billingPlan.plan}
            effectiveLimits={billingPlan.effective_limits}
            pricing={billingPlan.pricing}
          />
          <UsageDashboard
            plan={billingPlan.plan}
            effectiveLimits={billingPlan.effective_limits}
            subscription={billingPlan.subscription}
            pricing={billingPlan.pricing}
          />
          {billingPlan.coupon_redemption_enabled &&
            billingPlan.plan.id !== "free_pro" && <CouponRedemptionCard />}
          {billingPlan.subscription.can_upgrade && (
            <UpgradeCTA plan={billingPlan.plan} pricing={billingPlan.pricing} />
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
