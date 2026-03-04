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
}

const FREE_LIMITS = {
  agents: { label: "agents", limit: 3 },
  standing_approvals: { label: "standing approvals", limit: 5 },
  credentials: { label: "credentials", limit: 5 },
} as const;

function buildLimitWarnings(usage: UsageSummary | null): LimitWarning[] {
  if (!usage) return [];
  const warnings: LimitWarning[] = [];
  if (usage.agents > FREE_LIMITS.agents.limit) {
    warnings.push({
      resource: FREE_LIMITS.agents.label,
      current: usage.agents,
      limit: FREE_LIMITS.agents.limit,
    });
  }
  if (usage.standing_approvals > FREE_LIMITS.standing_approvals.limit) {
    warnings.push({
      resource: FREE_LIMITS.standing_approvals.label,
      current: usage.standing_approvals,
      limit: FREE_LIMITS.standing_approvals.limit,
    });
  }
  if (usage.credentials > FREE_LIMITS.credentials.limit) {
    warnings.push({
      resource: FREE_LIMITS.credentials.label,
      current: usage.credentials,
      limit: FREE_LIMITS.credentials.limit,
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
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
                  You&apos;re over free plan limits
                </p>
              </div>
              <ul className="ml-6 space-y-1">
                {warnings.map((w) => (
                  <li key={w.resource} className="text-sm text-amber-800 dark:text-amber-200">
                    You have {w.current} {w.resource}. Free tier allows {w.limit}.
                    You&apos;ll need to remove {w.current - w.limit} before downgrading.
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="rounded-lg border p-4 space-y-1">
            <p className="text-sm font-medium">What changes</p>
            <ul className="space-y-1 text-sm text-muted-foreground">
              <li>Audit retention drops from 90 days to 7 days</li>
              <li>Resource limits will be enforced (3 agents, 5 approvals, 5 credentials)</li>
              <li>1,000 request/month limit</li>
            </ul>
            <p className="mt-2 text-xs text-muted-foreground">
              A 7-day grace period preserves your 90-day audit retention so you
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
            {isPending && <Loader2 className="animate-spin" />}
            Confirm Downgrade
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
