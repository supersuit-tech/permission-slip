import { useState } from "react";
import { CreditCard, Loader2, Plus, Star, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  usePaymentMethods,
  useDeletePaymentMethod,
  useUpdatePaymentMethod,
} from "@/hooks/usePaymentMethods";
import { InlineConfirmButton } from "@/components/InlineConfirmButton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AddCardDialog } from "./AddCardDialog";
import { SpendingLimitsDialog } from "./SpendingLimitsDialog";
import type { PaymentMethod } from "@/hooks/usePaymentMethods";

function formatExpiry(month: number, year: number): string {
  return `${String(month).padStart(2, "0")}/${String(year).slice(-2)}`;
}

function formatBrand(brand: string): string {
  const brands: Record<string, string> = {
    visa: "Visa",
    mastercard: "Mastercard",
    amex: "Amex",
    discover: "Discover",
    diners: "Diners",
    jcb: "JCB",
    unionpay: "UnionPay",
  };
  return brands[brand] ?? brand;
}

function formatCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function PaymentMethodSection() {
  const { paymentMethods, isLoading, error } = usePaymentMethods();
  const { deletePaymentMethod, isLoading: isDeleting } =
    useDeletePaymentMethod();
  const { updatePaymentMethod } = useUpdatePaymentMethod();
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [limitsDialogPM, setLimitsDialogPM] = useState<PaymentMethod | null>(
    null,
  );

  async function handleDelete(pm: PaymentMethod) {
    try {
      await deletePaymentMethod(pm.id);
      toast.success(
        `Card ending in ${pm.last4} removed.`,
      );
    } catch {
      toast.error("Failed to remove payment method.");
    }
  }

  async function handleSetDefault(pm: PaymentMethod) {
    try {
      await updatePaymentMethod({ id: pm.id, is_default: true });
      toast.success(`Card ending in ${pm.last4} set as default.`);
    } catch {
      toast.error("Failed to set default payment method.");
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CreditCard className="text-muted-foreground size-5" />
              <CardTitle>Payment Methods</CardTitle>
              {paymentMethods.length > 0 && (
                <Badge variant="outline" className="ml-1">
                  {paymentMethods.length}
                </Badge>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAddDialogOpen(true)}
            >
              <Plus className="size-4" />
              Add Card
            </Button>
          </div>
          <CardDescription>
            Stored payment methods for agent-initiated purchases. Card data is
            held securely by Stripe &mdash; we never see your full card number.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading payment methods"
            >
              <Loader2 className="text-muted-foreground size-5 animate-spin" />
            </div>
          ) : error ? (
            <p className="text-destructive text-sm">{error}</p>
          ) : paymentMethods.length === 0 ? (
            <p className="text-muted-foreground py-4 text-center text-sm">
              No payment methods stored yet. Add a card so agents can make
              purchases on your behalf without ever seeing your card details.
            </p>
          ) : (
            <div className="space-y-3">
              {paymentMethods.map((pm) => (
                <div
                  key={pm.id}
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">
                        {formatBrand(pm.brand)} ending in {pm.last4}
                      </p>
                      {pm.is_default && (
                        <Badge variant="secondary" className="text-xs">
                          Default
                        </Badge>
                      )}
                    </div>
                    <p className="text-muted-foreground text-xs">
                      {pm.label ? `${pm.label} · ` : ""}
                      Expires {formatExpiry(pm.exp_month, pm.exp_year)}
                      {pm.per_transaction_limit != null &&
                        ` · Per-tx limit: ${formatCents(pm.per_transaction_limit)}`}
                      {pm.monthly_limit != null &&
                        ` · Monthly limit: ${formatCents(pm.monthly_limit)}`}
                      {pm.monthly_spend != null &&
                        pm.monthly_limit != null &&
                        ` (${formatCents(pm.monthly_spend)} used)`}
                    </p>
                  </div>
                  <div className="flex items-center gap-1">
                    {!pm.is_default && (
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Set as default"
                        onClick={() => handleSetDefault(pm)}
                      >
                        <Star className="text-muted-foreground size-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setLimitsDialogPM(pm)}
                    >
                      Limits
                    </Button>
                    <InlineConfirmButton
                      confirmLabel="Remove"
                      isProcessing={isDeleting}
                      onConfirm={() => handleDelete(pm)}
                    >
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove card ending in ${pm.last4}`}
                      >
                        <Trash2 className="text-muted-foreground size-4" />
                      </Button>
                    </InlineConfirmButton>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <AddCardDialog open={addDialogOpen} onOpenChange={setAddDialogOpen} />

      {limitsDialogPM && (
        <SpendingLimitsDialog
          paymentMethod={limitsDialogPM}
          open={!!limitsDialogPM}
          onOpenChange={(open) => !open && setLimitsDialogPM(null)}
        />
      )}
    </>
  );
}
