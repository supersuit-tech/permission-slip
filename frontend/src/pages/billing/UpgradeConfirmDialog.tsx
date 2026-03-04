import { ArrowUpRight, Check, Loader2 } from "lucide-react";
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
            <ul className="space-y-2">
              {PAID_PLAN_FEATURES.map((benefit) => (
                <li key={benefit} className="flex items-start gap-2 text-sm">
                  <Check className="mt-0.5 size-4 shrink-0 text-emerald-600" />
                  <span>{benefit}</span>
                </li>
              ))}
            </ul>
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
