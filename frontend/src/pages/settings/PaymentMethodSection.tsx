import { useState } from "react";
import {
  AlertTriangle,
  CreditCard,
  Loader2,
  Plus,
  Star,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { formatCents } from "@/lib/utils";
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

type ExpiryStatus = "expired" | "expiring_soon" | "ok";

function getExpiryStatus(expMonth: number, expYear: number): ExpiryStatus {
  const now = new Date();
  const currentMonth = now.getMonth() + 1; // 1-indexed
  const currentYear = now.getFullYear();

  // Card expires at the end of the expiry month
  if (
    expYear < currentYear ||
    (expYear === currentYear && expMonth < currentMonth)
  ) {
    return "expired";
  }

  // Expiring within 30 days
  const expiryDate = new Date(expYear, expMonth, 0); // last day of exp month
  const daysUntilExpiry = Math.ceil(
    (expiryDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24),
  );
  if (daysUntilExpiry <= 30) {
    return "expiring_soon";
  }

  return "ok";
}

function ExpiryBadge({
  expMonth,
  expYear,
}: {
  expMonth: number;
  expYear: number;
}) {
  const status = getExpiryStatus(expMonth, expYear);

  if (status === "expired") {
    return (
      <Badge variant="destructive" className="text-xs">
        Expired
      </Badge>
    );
  }

  if (status === "expiring_soon") {
    return (
      <Badge
        variant="outline"
        className="border-yellow-500 text-xs text-yellow-600 dark:text-yellow-400"
      >
        <AlertTriangle className="mr-1 size-3" />
        Expiring soon
      </Badge>
    );
  }

  return null;
}

function SpendProgressBar({
  monthlySpend,
  monthlyLimit,
}: {
  monthlySpend: number;
  monthlyLimit: number;
}) {
  const pct = Math.min((monthlySpend / monthlyLimit) * 100, 100);
  const remaining = Math.max(monthlyLimit - monthlySpend, 0);

  let barColor = "bg-blue-500";
  if (pct >= 85) barColor = "bg-red-500";
  else if (pct >= 60) barColor = "bg-yellow-500";

  return (
    <div className="mt-1.5 space-y-1">
      <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
        <div
          className={`h-full rounded-full transition-all ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <p className="text-muted-foreground text-xs">
        {formatCents(monthlySpend)} of {formatCents(monthlyLimit)} used
        {remaining > 0 && ` · ${formatCents(remaining)} remaining`}
      </p>
    </div>
  );
}

export function PaymentMethodSection() {
  const { paymentMethods, maxAllowed, isLoading, error } = usePaymentMethods();
  const { deletePaymentMethod, isLoading: isDeleting } =
    useDeletePaymentMethod();
  const { updatePaymentMethod } = useUpdatePaymentMethod();
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [limitsDialogPM, setLimitsDialogPM] = useState<PaymentMethod | null>(
    null,
  );

  const atCardLimit = maxAllowed > 0 && paymentMethods.length >= maxAllowed;

  async function handleDelete(pm: PaymentMethod) {
    try {
      await deletePaymentMethod(pm.id);
      toast.success(`Card ending in ${pm.last4} removed.`);
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
                  {maxAllowed > 0 && `/${maxAllowed}`}
                </Badge>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setAddDialogOpen(true)}
              disabled={atCardLimit}
              title={
                atCardLimit
                  ? `Maximum of ${maxAllowed} payment methods reached`
                  : undefined
              }
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
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">
                        {formatBrand(pm.brand)} ending in {pm.last4}
                      </p>
                      {pm.is_default && (
                        <Badge variant="secondary" className="text-xs">
                          Default
                        </Badge>
                      )}
                      <ExpiryBadge
                        expMonth={pm.exp_month}
                        expYear={pm.exp_year}
                      />
                    </div>
                    <p className="text-muted-foreground text-xs">
                      {pm.label ? `${pm.label} · ` : ""}
                      Expires {formatExpiry(pm.exp_month, pm.exp_year)}
                      {pm.per_transaction_limit != null &&
                        ` · Per-tx limit: ${formatCents(pm.per_transaction_limit)}`}
                    </p>
                    {pm.monthly_limit != null && pm.monthly_spend != null && (
                      <SpendProgressBar
                        monthlySpend={pm.monthly_spend}
                        monthlyLimit={pm.monthly_limit}
                      />
                    )}
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      title={
                        pm.is_default
                          ? "Default payment method"
                          : "Set as default"
                      }
                      onClick={() => !pm.is_default && handleSetDefault(pm)}
                      disabled={pm.is_default}
                    >
                      <Star
                        className={`size-4 ${pm.is_default ? "fill-yellow-400 text-yellow-400" : "text-muted-foreground"}`}
                      />
                    </Button>
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
