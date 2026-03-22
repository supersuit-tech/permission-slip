import { Link } from "react-router-dom";
import { AlertTriangle, Loader2 } from "lucide-react";
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
  const hasWarnings = warnings.length > 0;

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
          {hasWarnings && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 space-y-2 dark:border-amber-800 dark:bg-amber-950">
              <div className="flex items-start gap-2">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden="true" />
                <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
                  You&apos;re over free plan limits
                </p>
              </div>
              <ul className="ml-6 space-y-1.5">
                {warnings.map((w) => (
                  <li key={w.resource} className="text-sm text-amber-800 dark:text-amber-200">
                    You have {w.current} {w.resource}. Free tier allows {w.limit} after {freeLimitsApplyDate}.{" "}
                    <Link
                      to={w.managePath}
                      className="underline font-medium hover:text-amber-900 dark:hover:text-amber-100"
                    >
                      Manage {w.resource}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}

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
