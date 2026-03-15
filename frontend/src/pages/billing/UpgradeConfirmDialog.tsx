import { ArrowUpRight, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { BillingPricing } from "@/hooks/useBillingPlan";
import { PAID_PLAN_FEATURES, PAID_PLAN_PRICING } from "./constants";
import { FeatureList } from "./FeatureList";

interface UpgradeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isPending: boolean;
  pricing?: BillingPricing;
}

export function UpgradeConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
  pricing,
}: UpgradeConfirmDialogProps) {
  const pricingText = pricing
    ? `First ${pricing.free_request_allowance.toLocaleString()} requests/month are free. After that, ${pricing.price_per_request_display}/request.`
    : PAID_PLAN_PRICING;

  return (
    <Dialog open={open} onOpenChange={isPending ? undefined : onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upgrade to Pay-as-you-go</DialogTitle>
          <DialogDescription>
            You&apos;ll be redirected to Stripe to enter your payment details.
            No charges until you exceed the free tier.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg border bg-primary/5 p-4 space-y-3">
            <h3 className="text-sm font-semibold">What you get</h3>
            <FeatureList features={PAID_PLAN_FEATURES} />
          </div>

          <div className="rounded-lg border p-4 space-y-1">
            <h3 className="text-sm font-semibold">Pricing</h3>
            <p className="text-sm text-muted-foreground">
              {pricingText}
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button onClick={onConfirm} disabled={isPending}>
            {isPending ? (
              <Loader2 className="animate-spin" aria-hidden="true" />
            ) : (
              <ArrowUpRight aria-hidden="true" />
            )}
            Continue to Checkout
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
