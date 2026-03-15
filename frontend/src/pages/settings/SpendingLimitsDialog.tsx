import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useUpdatePaymentMethod } from "@/hooks/usePaymentMethods";
import type { PaymentMethod } from "@/hooks/usePaymentMethods";

interface SpendingLimitsDialogProps {
  paymentMethod: PaymentMethod;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SpendingLimitsDialog({
  paymentMethod,
  open,
  onOpenChange,
}: SpendingLimitsDialogProps) {
  const { updatePaymentMethod, isLoading } = useUpdatePaymentMethod();

  // Convert cents to dollars for display.
  const [perTxDollars, setPerTxDollars] = useState(
    paymentMethod.per_transaction_limit != null
      ? String(paymentMethod.per_transaction_limit / 100)
      : "",
  );
  const [monthlyDollars, setMonthlyDollars] = useState(
    paymentMethod.monthly_limit != null
      ? String(paymentMethod.monthly_limit / 100)
      : "",
  );
  const [label, setLabel] = useState(paymentMethod.label ?? "");
  const [validationError, setValidationError] = useState<string | null>(null);

  function validate(): boolean {
    const perTx = perTxDollars ? parseFloat(perTxDollars) : null;
    const monthly = monthlyDollars ? parseFloat(monthlyDollars) : null;

    if (perTx !== null && perTx < 0) {
      setValidationError("Per-transaction limit cannot be negative.");
      return false;
    }
    if (monthly !== null && monthly < 0) {
      setValidationError("Monthly limit cannot be negative.");
      return false;
    }
    if (perTx !== null && monthly !== null && perTx > monthly) {
      setValidationError(
        "Per-transaction limit cannot exceed the monthly limit.",
      );
      return false;
    }

    setValidationError(null);
    return true;
  }

  async function handleSave() {
    if (!validate()) return;

    try {
      const perTxCents = perTxDollars
        ? Math.round(parseFloat(perTxDollars) * 100)
        : undefined;
      const monthlyCents = monthlyDollars
        ? Math.round(parseFloat(monthlyDollars) * 100)
        : undefined;

      await updatePaymentMethod({
        id: paymentMethod.id,
        label: label || undefined,
        clear_label: !label && !!paymentMethod.label,
        per_transaction_limit: perTxCents,
        monthly_limit: monthlyCents,
        clear_per_transaction_limit:
          !perTxDollars && paymentMethod.per_transaction_limit != null,
        clear_monthly_limit:
          !monthlyDollars && paymentMethod.monthly_limit != null,
      });

      toast.success("Card settings updated.");
      onOpenChange(false);
    } catch {
      toast.error("Failed to update card settings.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            Card Settings &mdash;{" "}
            {paymentMethod.label ?? paymentMethod.last4}
          </DialogTitle>
          <DialogDescription>
            Rename this card and set optional spending limits to control how much
            agents can charge. Leave limits blank for no limit.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="pm-label">Card Nickname</Label>
            <Input
              id="pm-label"
              placeholder="e.g. Personal Visa, Groceries Card"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-muted-foreground text-xs">
              A friendly name to identify this card at a glance.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="per-tx-limit">Per-Transaction Limit ($)</Label>
            <Input
              id="per-tx-limit"
              type="number"
              min="0"
              step="0.01"
              placeholder="No limit"
              value={perTxDollars}
              onChange={(e) => {
                setPerTxDollars(e.target.value);
                setValidationError(null);
              }}
              disabled={isLoading}
            />
            <p className="text-muted-foreground text-xs">
              Maximum amount per individual agent transaction.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="monthly-limit">Monthly Limit ($)</Label>
            <Input
              id="monthly-limit"
              type="number"
              min="0"
              step="0.01"
              placeholder="No limit"
              value={monthlyDollars}
              onChange={(e) => {
                setMonthlyDollars(e.target.value);
                setValidationError(null);
              }}
              disabled={isLoading}
            />
            <p className="text-muted-foreground text-xs">
              Maximum total spend across all agent transactions in the last 30
              days.
            </p>
          </div>

          {validationError && (
            <p className="text-destructive text-sm">{validationError}</p>
          )}
        </div>

        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isLoading}>
            {isLoading ? (
              <>
                <Loader2 className="mr-2 size-4 animate-spin" />
                Saving...
              </>
            ) : (
              "Save"
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
