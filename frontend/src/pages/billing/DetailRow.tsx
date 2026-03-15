import type { ReactNode } from "react";

interface DetailRowProps {
  label: string;
  children: ReactNode;
}

/**
 * A label–value row for billing detail cards.
 *
 * Uses flex-wrap so that long values (e.g. pricing descriptions) wrap
 * below the label on narrow screens instead of colliding with it.
 */
export function DetailRow({ label, children }: DetailRowProps) {
  return (
    <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <span className="text-sm font-medium shrink-0">{label}</span>
      <div className="text-sm text-muted-foreground text-right">{children}</div>
    </div>
  );
}
