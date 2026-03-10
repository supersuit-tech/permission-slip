import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { useAuth } from "@/auth/AuthContext";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import { PlanCard } from "../billing/PlanCard";
import { UsageSummaryCard } from "../billing/UsageSummaryCard";
import { UpgradeCTA } from "../billing/UpgradeCTA";
import { UsageDashboard } from "../billing/UsageDashboard";
import { PlanDetailsCard } from "../billing/PlanDetailsCard";
import { BillingPageSkeleton } from "../billing/BillingPageSkeleton";
import { UpgradeSuccessBanner } from "../billing/UpgradeSuccessBanner";
import { activateUpgrade } from "../billing/activateUpgrade";
import { PaymentMethodSection } from "./PaymentMethodSection";
import { DataRetentionSection } from "./DataRetentionSection";

const RETRY_DELAYS = [1000, 2000, 3000];
const ACTIVATION_KEY = "ps-billing-activation-done";

export function BillingSettingsPage() {
  const { billingPlan, isLoading, error, refetch } = useBillingPlan();
  const { session } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();

  const sessionId = searchParams.get("session_id");
  const alreadyActivated = sessionId
    ? sessionStorage.getItem(ACTIVATION_KEY) === sessionId
    : false;

  const [showSuccess, setShowSuccess] = useState(
    searchParams.get("upgraded") === "true",
  );
  const activateRef = useRef(alreadyActivated);

  const isPaidPlan = billingPlan != null && billingPlan.plan.id !== "free";

  const retryUntilUpgraded = useCallback(async () => {
    for (const delay of RETRY_DELAYS) {
      await new Promise((r) => setTimeout(r, delay));
      const result = await refetch();
      if (result.data?.plan.id !== "free") return;
    }
  }, [refetch]);

  useEffect(() => {
    if (!showSuccess || activateRef.current) return;
    const sessionId = searchParams.get("session_id");
    if (!sessionId || !session?.access_token) {
      void refetch();
      return;
    }
    activateRef.current = true;
    if (sessionId) sessionStorage.setItem(ACTIVATION_KEY, sessionId);
    void activateUpgrade(sessionId, session.access_token)
      .then(() => refetch())
      .then((result) => {
        if (result?.data?.plan.id === "free") {
          void retryUntilUpgraded();
        }
      })
      .catch(() => {
        // Activation call failed — fall back to polling so the webhook
        // can still promote the plan.
        void retryUntilUpgraded();
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
    <>
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

      <PaymentMethodSection />
      <DataRetentionSection />
    </>
  );
}
