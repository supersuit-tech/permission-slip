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

interface EndQuotaGraceConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  isPending: boolean;
  error: string | null;
  usage: UsageSummary | null;
  /** When paid-plan quotas would stop if the user waits (end of cancelled period). */
  paidEntitlementsEndDate: string;
}

interface LimitWarning {
  resource: string;
  current: number;
  limit: number;
  managePath: string;
}

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

export function EndQuotaGraceConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  isPending,
  error,
  usage,
  paidEntitlementsEndDate,
}: EndQuotaGraceConfirmDialogProps) {
  const warnings = buildLimitWarnings(usage);
  const hasWarnings = warnings.length > 0;

  return (
    <Dialog open={open} onOpenChange={isPending ? undefined : onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Downgrade to free limits now?</DialogTitle>
          <DialogDescription>
            You&apos;re already on the Free plan, but paid-plan usage limits still apply until{" "}
            {paidEntitlementsEndDate}. Confirming ends those entitlements immediately so free-tier
            caps apply right away.
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
            <p className="text-sm font-medium">If you continue</p>
            <ul className="space-y-1 text-sm text-muted-foreground">
              <li>
                Free plan caps apply right away to new usage ({FREE_PLAN_LIMITS.agents.limit}{" "}
                agents, {FREE_PLAN_LIMITS.standing_approvals.limit} approvals,{" "}
                {FREE_PLAN_LIMITS.credentials.limit} credentials, {formatLimit(freePlan.max_requests_per_month)}{" "}
                requests/month). Existing resources stay; you may need to remove some before adding new ones.
              </li>
              <li>
                Your 7-day extended audit retention window (if still active) is unchanged — it started when you
                cancelled pay-as-you-go ({paidPlan.audit_retention_days} days vs {freePlan.audit_retention_days}{" "}
                on free).
              </li>
              <li>
                If you wait until {paidEntitlementsEndDate} instead, nothing changes until then.
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
            Downgrade now
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
