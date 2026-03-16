import { useState } from "react";
import {
  AlertTriangle,
  CreditCard,
  Loader2,
  Pencil,
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AddCardDialog } from "../settings/AddCardDialog";
import { SpendingLimitsDialog } from "../settings/SpendingLimitsDialog";
import type { PaymentMethod } from "@/hooks/usePaymentMethods";
import { formatBrand, formatExpiry } from "@/lib/paymentMethodUtils";

type ExpiryStatus = "expired" | "expiring_soon" | "ok";

function getExpiryStatus(expMonth: number, expYear: number): ExpiryStatus {
  const now = new Date();
  const currentMonth = now.getMonth() + 1;
  const currentYear = now.getFullYear();

  if (
    expYear < currentYear ||
    (expYear === currentYear && expMonth < currentMonth)
  ) {
    return "expired";
  }

  const expiryDate = new Date(expYear, expMonth, 0);
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
      <p className="text-muted-foreground text-sm">
        {formatCents(monthlySpend)} of {formatCents(monthlyLimit)} used
        {remaining > 0 && ` · ${formatCents(remaining)} remaining`}
      </p>
    </div>
  );
}

export interface ManagePaymentMethodsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ManagePaymentMethodsDialog({
  open,
  onOpenChange,
}: ManagePaymentMethodsDialogProps) {
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
      const result = await deletePaymentMethod(pm.id);
      if (result?.affected_agents && result.affected_agents > 0) {
        toast.success(
          `Card ending in ${pm.last4} removed. ${result.affected_agents} agent(s) had their payment method unassigned.`,
        );
      } else {
        toast.success(`Card ending in ${pm.last4} removed.`);
      }
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
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <div className="flex items-center gap-2">
              <CreditCard className="text-muted-foreground size-5" />
              <DialogTitle>Payment Methods</DialogTitle>
              {paymentMethods.length > 0 && (
                <Badge variant="outline" className="ml-1">
                  {paymentMethods.length}
                  {maxAllowed > 0 && `/${maxAllowed}`}
                </Badge>
              )}
            </div>
            <DialogDescription>
              Manage stored payment methods for agent-initiated purchases.
            </DialogDescription>
          </DialogHeader>

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
              purchases on your behalf.
            </p>
          ) : (
            <div className="space-y-3">
              {paymentMethods.map((pm) => (
                <div
                  key={pm.id}
                  className="flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <div className="flex items-center gap-2">
                      {pm.label ? (
                        <p className="text-sm font-medium">{pm.label}</p>
                      ) : (
                        <p className="text-sm font-medium">
                          {formatBrand(pm.brand)} ending in {pm.last4}
                        </p>
                      )}
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
                    <p className="text-muted-foreground text-sm">
                      {pm.label
                        ? `${formatBrand(pm.brand)} ending in ${pm.last4} · `
                        : ""}
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
                      variant="outline"
                      size="sm"
                      onClick={() => setLimitsDialogPM(pm)}
                    >
                      <Pencil className="mr-1 size-3" />
                      Edit
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

          <div className="flex justify-end pt-2">
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
        </DialogContent>
      </Dialog>

      <AddCardDialog open={addDialogOpen} onOpenChange={setAddDialogOpen} />

      {limitsDialogPM && (
        <SpendingLimitsDialog
          paymentMethod={limitsDialogPM}
          open={!!limitsDialogPM}
          onOpenChange={(o) => !o && setLimitsDialogPM(null)}
        />
      )}
    </>
  );
}
