import { useEffect, useRef, useState } from "react";
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

export function BillingPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();
  const { session } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSuccess, setShowSuccess] = useState(
    searchParams.get("upgraded") === "true",
  );
  const activateRef = useRef(false);

  const isPaidPlan = billingPlan != null && billingPlan.plan.id !== "free";

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
    void activateUpgrade(sessionId, session.access_token).then(() => refetch());
  }, [showSuccess, searchParams, session?.access_token, refetch]);

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
