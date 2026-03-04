import { Clock } from "lucide-react";
import { Link } from "react-router-dom";

interface RetentionInfo {
  days: number;
  grace_period_ends_at?: string | null;
}

interface RetentionBannerProps {
  retention: RetentionInfo;
}

function formatGracePeriodDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function RetentionBanner({ retention }: RetentionBannerProps) {
  const { days, grace_period_ends_at } = retention;

  if (grace_period_ends_at) {
    return (
      <div className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800 flex items-start gap-2 rounded-lg border px-4 py-3">
        <Clock className="text-amber-600 dark:text-amber-400 mt-0.5 size-4 shrink-0" />
        <p className="text-amber-800 dark:text-amber-200 text-sm">
          Your 90-day audit history is preserved until{" "}
          <span className="font-medium">
            {formatGracePeriodDate(grace_period_ends_at)}
          </span>
          . After that, retention will drop to 7 days.{" "}
          <Link to="/billing" className="font-medium underline">
            Upgrade
          </Link>{" "}
          to keep 90-day history.
        </p>
      </div>
    );
  }

  // Only show the banner for limited retention (free-tier)
  if (days >= 90) return null;

  return (
    <div className="bg-muted/50 flex items-center gap-2 rounded-lg border px-4 py-3">
      <Clock className="text-muted-foreground size-4 shrink-0" />
      <p className="text-muted-foreground text-sm">
        Showing last {days} days (Free plan).{" "}
        <Link to="/billing" className="text-foreground font-medium underline">
          Upgrade
        </Link>{" "}
        for 90-day history.
      </p>
    </div>
  );
}
