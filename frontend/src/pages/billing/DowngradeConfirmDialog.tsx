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

interface DowngradeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isPending: boolean;
  error: string | null;
  usage: UsageSummary | null;
}

interface LimitWarning {
  resource: string;
  current: number;
  limit: number;
  managePath: string;
}

/** Compare current usage against free plan limits and return warnings for resources over the limit. */
function buildLimitWarnings(usage: UsageSummary | null): LimitWarning[] {
  if (!usage) return [];
  const warnings: LimitWarning[] = [];
  if (usage.agents > FREE_PLAN_LIMITS.agents.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.agents.label,
      current: usage.agents,
      limit: FREE_PLAN_LIMITS.agents.limit,
      managePath: FREE_PLAN_LIMITS.agents.managePath,
    });
  }
  if (usage.standing_approvals > FREE_PLAN_LIMITS.standing_approvals.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.standing_approvals.label,
      current: usage.standing_approvals,
      limit: FREE_PLAN_LIMITS.standing_approvals.limit,
      managePath: FREE_PLAN_LIMITS.standing_approvals.managePath,
    });
  }
  if (usage.credentials > FREE_PLAN_LIMITS.credentials.limit) {
    warnings.push({
      resource: FREE_PLAN_LIMITS.credentials.label,
      current: usage.credentials,
      limit: FREE_PLAN_LIMITS.credentials.limit,
      managePath: FREE_PLAN_LIMITS.credentials.managePath,
    });
  }
  return warnings;
}

export function DowngradeConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
  error,
  usage,
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
                    You have {w.current} {w.resource}. Free tier allows {w.limit}.{" "}
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
              <li>Audit retention drops from {paidPlan.audit_retention_days} days to {freePlan.audit_retention_days} days</li>
              <li>
                Resource limits will be enforced ({FREE_PLAN_LIMITS.agents.limit} agents,{" "}
                {FREE_PLAN_LIMITS.standing_approvals.limit} approvals,{" "}
                {FREE_PLAN_LIMITS.credentials.limit} credentials)
              </li>
              <li>{formatLimit(freePlan.max_requests_per_month!)} request/month limit</li>
            </ul>
            <p className="mt-2 text-xs text-muted-foreground">
              A 7-day grace period preserves your {paidPlan.audit_retention_days}-day audit retention so you
              can export data before it reverts.
            </p>
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
            disabled={isPending || hasWarnings}
          >
            {isPending && <Loader2 className="animate-spin" aria-hidden="true" />}
            Confirm Downgrade
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
