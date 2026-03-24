import { Link } from "react-router-dom";
import { AlertTriangle } from "lucide-react";
import type { LimitWarning } from "./downgradeUtils";

interface LimitWarningsListProps {
  warnings: LimitWarning[];
  /** Optional suffix appended after the limit count (e.g. "after March 30"). */
  limitSuffix?: string;
}

/** Shared warning banner for resources exceeding free plan limits. */
export function LimitWarningsList({ warnings, limitSuffix }: LimitWarningsListProps) {
  if (warnings.length === 0) return null;

  return (
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
            You have {w.current} {w.resource}. Free tier allows {w.limit}{limitSuffix ? ` ${limitSuffix}` : ""}.{" "}
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
  );
}
