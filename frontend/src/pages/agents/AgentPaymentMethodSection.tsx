import { useState } from "react";
import { CreditCard, Loader2, Settings } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { usePaymentMethods } from "@/hooks/usePaymentMethods";
import type { PaymentMethod } from "@/hooks/usePaymentMethods";
import {
  useAgentPaymentMethod,
  useAssignAgentPaymentMethod,
  useRemoveAgentPaymentMethod,
} from "@/hooks/useAgentPaymentMethod";
import { formatBrand, formatExpiry } from "@/lib/paymentMethodUtils";
import { ManagePaymentMethodsDialog } from "./ManagePaymentMethodsDialog";

interface AgentPaymentMethodSectionProps {
  agentId: number;
}

const selectClassName =
  "border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 text-sm";

export function AgentPaymentMethodSection({
  agentId,
}: AgentPaymentMethodSectionProps) {
  const [manageDialogOpen, setManageDialogOpen] = useState(false);

  const {
    paymentMethods,
    isLoading: pmLoading,
    error: pmError,
  } = usePaymentMethods();
  const {
    binding,
    isLoading: bindingLoading,
    error: bindingError,
  } = useAgentPaymentMethod(agentId);
  const { assign, isPending: assigning } = useAssignAgentPaymentMethod();
  const { remove, isPending: removing } = useRemoveAgentPaymentMethod();

  const isLoading = pmLoading || bindingLoading;
  const error = pmError ?? bindingError;
  const busy = assigning || removing;

  const currentValue = binding?.payment_method_id ?? "";

  async function handleChange(value: string) {
    if (busy) return;

    try {
      if (value === "") {
        await remove({ agentId });
        toast.success("Payment method unassigned from this agent.");
      } else {
        await assign({ agentId, paymentMethodId: value });
        toast.success("Payment method assigned to this agent.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update payment method.",
      );
    }
  }

  function cardLabel(pm: PaymentMethod): string {
    const brand = formatBrand(pm.brand);
    const base = pm.label
      ? `${pm.label} — ${brand} ••${pm.last4}`
      : `${brand} ••${pm.last4}`;
    return `${base} (exp ${formatExpiry(pm.exp_month, pm.exp_year)})`;
  }

  return (
    <>
      <Card>
        <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <CreditCard className="size-5" />
              Payment Method
              <Badge variant="secondary" className="text-xs font-normal">
                Optional
              </Badge>
            </CardTitle>
            <p className="text-muted-foreground mt-1 text-xs">
              Only needed if you want this agent to use connector actions that
              make purchases.
            </p>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading payment method"
            >
              <Loader2
                className="text-muted-foreground size-6 animate-spin"
                aria-hidden="true"
              />
            </div>
          ) : error ? (
            <p className="text-destructive text-sm">{error}</p>
          ) : paymentMethods.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <CreditCard className="text-muted-foreground mb-3 size-10" />
              <p className="text-muted-foreground mb-3 text-sm">
                No payment methods added yet. You only need a payment method if
                this agent will use connector actions that make purchases.
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setManageDialogOpen(true)}
              >
                <Settings className="size-3" />
                Add payment method
              </Button>
            </div>
          ) : (
            <>
              <div className="mb-4 rounded-lg border p-3">
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Label
                      htmlFor="agent-payment-method-select"
                      className="text-sm font-medium"
                    >
                      Assigned Payment Method
                    </Label>
                    {currentValue ? (
                      <Badge
                        variant="secondary"
                        className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                      >
                        Assigned
                      </Badge>
                    ) : (
                      <Badge variant="destructive">Not set</Badge>
                    )}
                  </div>
                  <p className="text-muted-foreground text-xs">
                    {currentValue
                      ? "This agent will use the selected payment method for purchases."
                      : "No payment method assigned. Only needed if this agent uses connector actions that make purchases."}
                  </p>
                  <select
                    id="agent-payment-method-select"
                    className={selectClassName}
                    value={currentValue}
                    onChange={(e) => handleChange(e.target.value)}
                    disabled={busy}
                  >
                    <option value="">
                      {currentValue
                        ? "None (unassigned)"
                        : "Select a payment method…"}
                    </option>
                    {paymentMethods.map((pm) => (
                      <option key={pm.id} value={pm.id}>
                        {cardLabel(pm)}
                        {pm.is_default ? " (default)" : ""}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="flex justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setManageDialogOpen(true)}
                >
                  <Settings className="size-3" />
                  Manage payment methods
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <ManagePaymentMethodsDialog
        open={manageDialogOpen}
        onOpenChange={setManageDialogOpen}
      />
    </>
  );
}
