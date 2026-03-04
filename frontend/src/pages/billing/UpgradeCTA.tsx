import { useState } from "react";
import { ArrowUpRight, Check, Loader2, Sparkles } from "lucide-react";
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

const FREE_FEATURES = [
  "1,000 requests/month",
  "3 agents",
  "5 standing approvals",
  "5 credentials",
  "7-day audit retention",
];

const PAID_FEATURES = [
  "Unlimited requests (pay per use)",
  "Unlimited agents",
  "Unlimited standing approvals",
  "Unlimited credentials",
  "90-day audit retention",
];

function FeatureList({ features, variant }: { features: string[]; variant: "free" | "paid" }) {
  return (
    <ul className="space-y-2">
      {features.map((feature) => (
        <li key={feature} className="flex items-start gap-2 text-sm">
          <Check className={`mt-0.5 size-4 shrink-0 ${variant === "paid" ? "text-emerald-600" : "text-muted-foreground"}`} />
          <span>{feature}</span>
        </li>
      ))}
    </ul>
  );
}

export function UpgradeCTA() {
  const { upgrade, isUpgrading } = useUpgradePlan();
  const [isRedirecting, setIsRedirecting] = useState(false);

  async function handleUpgrade() {
    try {
      const result = await upgrade();
      if (result?.checkout_url) {
        setIsRedirecting(true);
        window.location.href = result.checkout_url;
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to initiate upgrade. Please try again.",
      );
    }
  }

  const isPending = isUpgrading || isRedirecting;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Sparkles className="text-muted-foreground size-5" />
          <CardTitle>Upgrade Your Plan</CardTitle>
        </div>
        <CardDescription>
          Unlock unlimited resources with Pay-as-you-go.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-6 sm:grid-cols-2">
          <div className="rounded-lg border p-4 space-y-3">
            <h3 className="text-sm font-semibold">Free</h3>
            <p className="text-xs text-muted-foreground">Your current plan</p>
            <FeatureList features={FREE_FEATURES} variant="free" />
          </div>
          <div className="rounded-lg border border-primary/30 bg-primary/5 p-4 space-y-3">
            <h3 className="text-sm font-semibold">Pay-as-you-go</h3>
            <p className="text-xs text-muted-foreground">$0.005/request after free tier</p>
            <FeatureList features={PAID_FEATURES} variant="paid" />
          </div>
        </div>
        <div className="mt-6 flex justify-end">
          <Button onClick={handleUpgrade} disabled={isPending}>
            {isPending ? (
              <Loader2 className="animate-spin" />
            ) : (
              <ArrowUpRight />
            )}
            {isRedirecting ? "Redirecting…" : "Upgrade to Pay-as-you-go"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
