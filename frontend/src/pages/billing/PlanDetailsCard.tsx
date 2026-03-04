import { useState } from "react";
import {
  Receipt,
  CreditCard,
  ExternalLink,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useDowngradePlan } from "@/hooks/useDowngradePlan";
import { useBillingUsage } from "@/hooks/useBillingUsage";
import { useBillingInvoices } from "@/hooks/useBillingInvoices";
import { useBillingPlan } from "@/hooks/useBillingPlan";
import type { Subscription } from "@/hooks/useBillingPlan";
import { formatCents, formatDate, isSafeUrl } from "./formatters";
import { DowngradeConfirmDialog } from "./DowngradeConfirmDialog";

interface PlanDetailsCardProps {
  subscription: Subscription;
}

function CostEstimate() {
  const { usage, isLoading } = useBillingUsage();

  if (isLoading) {
    return <span className="text-sm text-muted-foreground">Calculating…</span>;
  }

  if (!usage) {
    return <span className="text-sm text-muted-foreground">—</span>;
  }

  const totalCents = usage.requests.cost_cents + usage.sms.cost_cents;
  return <span className="text-sm text-muted-foreground tabular-nums">{formatCents(totalCents)}</span>;
}

function InvoicesList() {
  const { invoices, isLoading, error, refetch } = useBillingInvoices(5);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-between text-sm">
        <span className="text-destructive">Failed to load invoices.</span>
        <Button variant="ghost" size="sm" onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (invoices.length === 0) {
    return <p className="text-sm text-muted-foreground">No invoices yet.</p>;
  }

  return (
    <div className="space-y-2">
      {invoices.map((invoice) => (
        <div key={invoice.id} className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">
            {formatDate(invoice.date)}
          </span>
          <div className="flex items-center gap-3">
            <span className="tabular-nums">{formatCents(invoice.amount_cents)}</span>
            {invoice.stripe_invoice_url && isSafeUrl(invoice.stripe_invoice_url) && (
              <a
                href={invoice.stripe_invoice_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
                aria-label={`View invoice from ${formatDate(invoice.date)}`}
              >
                <ExternalLink className="size-3.5" />
              </a>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function DowngradeSection() {
  const { downgrade, isDowngrading } = useDowngradePlan();
  const { billingPlan } = useBillingPlan();
  const [showConfirm, setShowConfirm] = useState(false);
  const [downgradeError, setDowngradeError] = useState<string | null>(null);

  async function handleDowngrade() {
    setDowngradeError(null);
    try {
      const result = await downgrade();
      setShowConfirm(false);
      const graceMsg = result?.grace_period_ends_at
        ? ` Your paid features remain active until ${formatDate(result.grace_period_ends_at)}.`
        : "";
      toast.success(`Your plan has been downgraded to Free.${graceMsg}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to downgrade. Please try again.";
      setDowngradeError(message);
    }
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => {
          setDowngradeError(null);
          setShowConfirm(true);
        }}
      >
        Downgrade to Free
      </Button>
      <DowngradeConfirmDialog
        open={showConfirm}
        onOpenChange={setShowConfirm}
        onConfirm={handleDowngrade}
        isPending={isDowngrading}
        error={downgradeError}
        usage={billingPlan?.usage ?? null}
      />
    </>
  );
}

export function PlanDetailsCard({ subscription }: PlanDetailsCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Receipt className="text-muted-foreground size-5" />
          <CardTitle>Plan Details</CardTitle>
        </div>
        <CardDescription>
          Billing estimates, payment info, and invoices.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="rounded-lg border p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Estimated Cost (this month)</span>
              <CostEstimate />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Payment Method</span>
              <span className="text-sm text-muted-foreground">
                {subscription.has_payment_method ? (
                  <span className="inline-flex items-center gap-1.5">
                    <CreditCard className="size-3.5" />
                    On file
                  </span>
                ) : (
                  "None"
                )}
              </span>
            </div>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium">Recent Invoices</h3>
            <InvoicesList />
          </div>

          {subscription.can_downgrade && (
            <div className="border-t pt-4">
              <DowngradeSection />
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
