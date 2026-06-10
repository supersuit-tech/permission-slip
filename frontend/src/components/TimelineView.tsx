/**
 * Vertical timeline for approval lifecycle timestamps.
 * Terminal events (approved/denied) use semantic dot colors.
 */
export interface TimelineEntry {
  label: string;
  value: string;
  dotColor?: "success" | "error";
}

interface TimelineViewProps {
  entries: TimelineEntry[];
}

const DOT_COLORS = {
  success: "bg-emerald-500",
  error: "bg-destructive",
  default: "bg-muted-foreground/40",
} as const;

export function TimelineView({ entries }: TimelineViewProps) {
  return (
    <div className="pl-1">
      {entries.map((entry, index) => {
        const isLast = index === entries.length - 1;
        const dotClass =
          entry.dotColor === "success"
            ? DOT_COLORS.success
            : entry.dotColor === "error"
              ? DOT_COLORS.error
              : DOT_COLORS.default;

        return (
          <div key={`${index}-${entry.label}`} className="flex min-h-8">
            <div className="flex w-5 shrink-0 flex-col items-center">
              <span
                className={`mt-1.5 size-2 shrink-0 rounded-full ${dotClass}`}
                aria-hidden="true"
              />
              {!isLast && (
                <span className="bg-border my-0.5 w-0.5 flex-1" aria-hidden="true" />
              )}
            </div>
            <div className="mb-3 ml-2 flex flex-1 items-start justify-between gap-3">
              <span className="text-muted-foreground text-sm font-medium">
                {entry.label}
              </span>
              <span className="text-foreground shrink-0 text-right text-sm">
                {entry.value}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
