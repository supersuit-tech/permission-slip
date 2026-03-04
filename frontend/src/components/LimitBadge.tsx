import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface LimitBadgeProps {
  current: number;
  max: number | null;
  resource: string;
}

/**
 * Displays a resource usage counter relative to its plan limit.
 * Shows "2 / 3 agents" (free tier with limit) or "5 agents" (paid, no limit).
 * Colors: red at limit, amber ≥80%, default otherwise.
 */
export function LimitBadge({ current, max, resource }: LimitBadgeProps) {
  if (max == null) {
    return (
      <Badge
        variant="outline"
        className="text-xs font-normal"
        aria-label={`${current} ${resource} used`}
      >
        {current} {resource}
      </Badge>
    );
  }

  const atLimit = current >= max;
  const nearLimit = max > 0 && current / max >= 0.8;

  const remaining = max - current;
  const ariaLabel = atLimit
    ? `${resource} limit reached (${current} of ${max})`
    : `${current} of ${max} ${resource} used, ${remaining} remaining`;

  return (
    <Badge
      variant="outline"
      className={cn(
        "text-xs font-normal",
        atLimit &&
          "border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400",
        !atLimit &&
          nearLimit &&
          "border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400",
      )}
      aria-label={ariaLabel}
    >
      {current} / {max} {resource}
    </Badge>
  );
}
