import { useState } from "react";
import {
  Receipt,
  CreditCard,
  ExternalLink,
  Loader2,
  AlertTriangle,
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
import type { Subscription } from "@/hooks/useBillingPlan";
import { formatCents, formatDate, isSafeUrl } from "./formatters";

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
  const { invoices, isLoading } = useBillingInvoices(5);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="text-muted-foreground size-4 animate-spin" />
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
  const [showConfirm, setShowConfirm] = useState(false);

  async function handleDowngrade() {
    try {
      const result = await downgrade();
      setShowConfirm(false);
      const graceMsg = result?.grace_period_ends_at
        ? ` Your paid features remain active until ${formatDate(result.grace_period_ends_at)}.`
        : "";
      toast.success(`Your plan has been downgraded to Free.${graceMsg}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to downgrade. Please try again.",
      );
    }
  }

  if (!showConfirm) {
    return (
      <Button
        variant="outline"
        size="sm"
        onClick={() => setShowConfirm(true)}
      >
        Downgrade to Free
      </Button>
    );
  }

  return (
    <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 space-y-3">
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 size-4 shrink-0 text-destructive" />
        <div className="space-y-1">
          <p className="text-sm font-medium">Are you sure?</p>
          <p className="text-xs text-muted-foreground">
            You&apos;ll lose access to unlimited resources. Your data will be
            preserved for 7 days before free plan limits apply.
          </p>
        </div>
      </div>
      <div className="flex gap-2">
        <Button
          variant="destructive"
          size="sm"
          onClick={handleDowngrade}
          disabled={isDowngrading}
        >
          {isDowngrading && <Loader2 className="animate-spin" />}
          Confirm Downgrade
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowConfirm(false)}
          disabled={isDowngrading}
        >
          Cancel
        </Button>
      </div>
    </div>
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
