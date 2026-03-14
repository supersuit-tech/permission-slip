import { PRICE_PER_REQUEST } from "./constants";

interface RequestUsageBarProps {
  total: number;
  limit: number | null;
  /** Free request allowance for paid plans (e.g. 250). */
  included?: number;
}

/**
 * Segmented bar for paid plans showing free vs billed requests.
 * Green portion = free allowance, amber = billed overage.
 */
function PaidRequestBar({ total, included }: { total: number; included: number }) {
  const overage = Math.max(0, total - included);
  const hasOverage = overage > 0;

  // Bar is measured against included allowance (e.g. 250).
  // When under: green bar grows toward 100%.
  // When over: green fills 100%, amber extends proportionally.
  const freePercent = Math.min((total / included) * 100, 100);
  // Overage segment as proportion of total (so bar width stays readable).
  const overagePercent = hasOverage ? (overage / total) * 100 : 0;
  const freeSegmentPercent = hasOverage ? 100 - overagePercent : freePercent;

  const label = hasOverage
    ? `${total.toLocaleString()} total (${overage.toLocaleString()} billed)`
    : `${total.toLocaleString()} / ${included.toLocaleString()} free`;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">Requests this period</span>
        <span className="text-muted-foreground tabular-nums">{label}</span>
      </div>
      <div
        className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-valuenow={Math.min(total, included)}
        aria-valuemin={0}
        aria-valuemax={included}
        aria-label={`Requests: ${label}`}
      >
        <div
          className="h-full rounded-l-full bg-primary transition-all"
          style={{ width: `${freeSegmentPercent}%` }}
        />
        {hasOverage && (
          <div
            className="h-full rounded-r-full bg-amber-500 transition-all"
            style={{ width: `${overagePercent}%` }}
          />
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        {hasOverage
          ? `${included.toLocaleString()} free + ${overage.toLocaleString()} at ${PRICE_PER_REQUEST}/request`
          : `First ${included.toLocaleString()} requests/month are free, then ${PRICE_PER_REQUEST}/request`}
      </p>
    </div>
  );
}

export function RequestUsageBar({ total, limit, included }: RequestUsageBarProps) {
  // Paid plan with free allowance: show segmented bar.
  if (limit === null && included != null && included > 0) {
    return <PaidRequestBar total={total} included={included} />;
  }

  const isUnlimited = limit === null;
  const percentage = isUnlimited
    ? 0
    : limit > 0
      ? Math.min((total / limit) * 100, 100)
      : 0;
  const isNearLimit = !isUnlimited && limit > 0 && percentage >= 80;
  const isAtLimit = !isUnlimited && limit > 0 && total >= limit;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium">Requests this period</span>
        <span className="text-muted-foreground tabular-nums">
          {total.toLocaleString()} /{" "}
          {isUnlimited ? "Unlimited" : limit.toLocaleString()}
        </span>
      </div>
      {!isUnlimited && (
        <div
          className="h-2.5 w-full overflow-hidden rounded-full bg-muted"
          role="progressbar"
          aria-valuenow={Math.min(total, limit)}
          aria-valuemin={0}
          aria-valuemax={limit}
          aria-label={`Requests: ${total.toLocaleString()} of ${limit.toLocaleString()} used`}
        >
          <div
            className={`h-full rounded-full transition-all ${
              isAtLimit
                ? "bg-destructive"
                : isNearLimit
                  ? "bg-amber-500"
                  : "bg-primary"
            }`}
            style={{ width: `${percentage}%` }}
          />
        </div>
      )}
    </div>
  );
}
