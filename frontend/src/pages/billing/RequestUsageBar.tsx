interface RequestUsageBarProps {
  total: number;
  limit: number | null;
}

export function RequestUsageBar({ total, limit }: RequestUsageBarProps) {
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
