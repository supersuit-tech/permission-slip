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
import { PAID_PLAN_FEATURES, PAID_PLAN_PRICING } from "./constants";
import { FeatureList } from "./FeatureList";

interface UpgradeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isPending: boolean;
}

export function UpgradeConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
}: UpgradeConfirmDialogProps) {
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
              {PAID_PLAN_PRICING}
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
              <Loader2 className="animate-spin" />
            ) : (
              <ArrowUpRight />
            )}
            Continue to Checkout
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
