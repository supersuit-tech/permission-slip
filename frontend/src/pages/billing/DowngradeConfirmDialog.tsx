import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { UsageSummary } from "@/hooks/useBillingPlan";
import { freePlan, paidPlan, formatLimit } from "@/config/plans";
import { FREE_PLAN_LIMITS } from "./constants";
import { buildLimitWarnings } from "./downgradeUtils";
import { LimitWarningsList } from "./LimitWarningsList";

interface DowngradeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isPending: boolean;
  error: string | null;
  usage: UsageSummary | null;
  /** When free-tier resource caps start applying after downgrade (end of current billing period). */
  freeLimitsApplyDate: string;
}

export function DowngradeConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
  error,
  usage,
  freeLimitsApplyDate,
}: DowngradeConfirmDialogProps) {
  const warnings = buildLimitWarnings(usage);

  return (
    <Dialog open={open} onOpenChange={isPending ? undefined : onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Downgrade to Free</DialogTitle>
          <DialogDescription>
            Are you sure you want to downgrade? You&apos;ll lose access to
            unlimited resources.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <LimitWarningsList warnings={warnings} limitSuffix={`after ${freeLimitsApplyDate}`} />

          <div className="rounded-lg border p-4 space-y-1">
            <p className="text-sm font-medium">What changes</p>
            <ul className="space-y-1 text-sm text-muted-foreground">
              <li>
                Until {freeLimitsApplyDate}, paid-plan usage limits still apply (unlimited requests and resources on
                your current plan).
              </li>
              <li>
                After {freeLimitsApplyDate}, free tier caps apply to new usage ({FREE_PLAN_LIMITS.agents.limit}{" "}
                agents, {FREE_PLAN_LIMITS.standing_approvals.limit} approvals,{" "}
                {FREE_PLAN_LIMITS.credentials.limit} credentials, {formatLimit(freePlan.max_requests_per_month)}{" "}
                requests/month). Existing resources are kept; reduce them before adding new ones if you are over the cap.
              </li>
              <li>
                A separate 7-day window after downgrade keeps extended audit log retention ({paidPlan.audit_retention_days}{" "}
                days) so you can export older activity before it matches the free plan ({freePlan.audit_retention_days}{" "}
                days).
              </li>
            </ul>
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isPending}
          >
            {isPending && <Loader2 className="animate-spin" aria-hidden="true" />}
            Confirm Downgrade
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
