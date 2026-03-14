import { useState } from "react";
import { ArrowUpRight, Loader2, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useUpgradePlan } from "@/hooks/useUpgradePlan";
import type { Plan } from "@/hooks/useBillingPlan";
import { paidPlan } from "@/config/plans";
import { isStripeUrl } from "./formatters";
import { PAID_PLAN_FEATURES } from "./constants";
import { FeatureList } from "./FeatureList";
import { UpgradeConfirmDialog } from "./UpgradeConfirmDialog";

interface UpgradeCTAProps {
  plan: Plan;
}

function formatLimit(value: number | null | undefined, unit: string): string {
  if (value === null || value === undefined) return `Unlimited ${unit}`;
  return `${value.toLocaleString()} ${unit}`;
}

function buildCurrentFeatures(plan: Plan): string[] {
  return [
    `${formatLimit(plan.max_requests_per_month, "requests/month")}`,
    `${formatLimit(plan.max_agents, "agents")}`,
    `${formatLimit(plan.max_standing_approvals, "standing approvals")}`,
    `${formatLimit(plan.max_credentials, "credentials")}`,
    `${plan.audit_retention_days}-day audit retention`,
  ];
}

const PAID_FEATURES = [
  "Unlimited requests (pay per use)",
  ...PAID_PLAN_FEATURES,
];

export function UpgradeCTA({ plan }: UpgradeCTAProps) {
  const { upgrade, isUpgrading } = useUpgradePlan();
  const [isRedirecting, setIsRedirecting] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const currentFeatures = buildCurrentFeatures(plan);

  async function handleUpgrade() {
    try {
      const result = await upgrade();
      if (!result?.checkout_url) {
        throw new Error("No checkout URL returned. Please try again.");
      }
      if (!isStripeUrl(result.checkout_url)) {
        throw new Error("Invalid checkout URL received");
      }
      setIsRedirecting(true);
      window.location.href = result.checkout_url;
    } catch (err) {
      setShowConfirm(false);
      toast.error(
        err instanceof Error ? err.message : "Failed to initiate upgrade. Please try again.",
      );
    }
  }

  const isPending = isUpgrading || isRedirecting;

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Sparkles className="text-muted-foreground size-5" aria-hidden="true" />
            <CardTitle>Upgrade Your Plan</CardTitle>
          </div>
          <CardDescription>
            Unlock unlimited resources with Pay-as-you-go.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 sm:grid-cols-2">
            <div className="rounded-lg border p-4 space-y-3">
              <h3 className="text-sm font-semibold">{plan.name}</h3>
              <p className="text-xs text-muted-foreground">Your current plan</p>
              <FeatureList features={currentFeatures} variant="muted" />
            </div>
            <div className="rounded-lg border border-primary/30 bg-primary/5 p-4 space-y-3">
              <h3 className="text-sm font-semibold">Pay-as-you-go</h3>
              <p className="text-xs text-muted-foreground">${(paidPlan.price_per_request_millicents / 100_000).toFixed(3)}/request after free tier</p>
              <FeatureList features={PAID_FEATURES} />
            </div>
          </div>
          <div className="mt-6 flex justify-end">
            <Button onClick={() => setShowConfirm(true)} disabled={isPending}>
              {isPending ? (
                <Loader2 className="animate-spin" aria-hidden="true" />
              ) : (
                <ArrowUpRight aria-hidden="true" />
              )}
              {isRedirecting ? "Redirecting…" : "Upgrade to Pay-as-you-go"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <UpgradeConfirmDialog
        open={showConfirm}
        onOpenChange={setShowConfirm}
        onConfirm={handleUpgrade}
        isPending={isPending}
      />
    </>
  );
}
