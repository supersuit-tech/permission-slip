import { useState, useEffect } from "react";
import { Clock } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { CopyableCode } from "@/components/CopyableCode";

/** Returns seconds remaining until `expiresAt`, clamped to 0. */
export function secondsUntil(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now();
  return Math.max(0, Math.floor(diff / 1_000));
}

/** Formats seconds as "M:SS". */
export function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

/** Returns a Tailwind color class based on remaining seconds. */
export function urgencyColor(seconds: number): string {
  if (seconds <= 0) return "text-muted-foreground";
  if (seconds <= 60) return "text-destructive";
  if (seconds <= 120) return "text-orange-500";
  return "text-muted-foreground";
}

/**
 * Shared 1-second ticker so all countdown consumers share a single interval
 * instead of each row creating its own setInterval.
 */
const tickerSubscribers = new Set<() => void>();
let tickerTimerId: ReturnType<typeof setInterval> | null = null;

function startTicker() {
  if (tickerTimerId != null) return;
  tickerTimerId = setInterval(() => {
    tickerSubscribers.forEach((cb) => cb());
  }, 1_000);
}

function stopTickerIfIdle() {
  if (tickerSubscribers.size === 0 && tickerTimerId != null) {
    clearInterval(tickerTimerId);
    tickerTimerId = null;
  }
}

/**
 * Returns seconds remaining until `expiresAt`, re-evaluated every second via
 * a single shared interval (no per-row timers).
 */
export function useCountdown(expiresAt: string): number {
  const [remaining, setRemaining] = useState(() => secondsUntil(expiresAt));

  useEffect(() => {
    const cb = () => setRemaining(secondsUntil(expiresAt));
    tickerSubscribers.add(cb);
    startTicker();

    return () => {
      tickerSubscribers.delete(cb);
      stopTickerIfIdle();
    };
  }, [expiresAt]);

  return remaining;
}

export function CountdownBadge({ expiresAt }: { expiresAt: string }) {
  const remaining = useCountdown(expiresAt);
  const isExpired = remaining <= 0;

  return (
    <span
      aria-label={isExpired ? "Expired" : `${formatCountdown(remaining)} remaining`}
      className={`inline-flex items-center gap-1 text-xs font-medium ${urgencyColor(remaining)}`}
    >
      <Clock className="size-3" aria-hidden="true" />
      {isExpired ? "Expired" : formatCountdown(remaining)}
    </span>
  );
}

export function RiskBadge({ level }: { level?: "low" | "medium" | "high" | null }) {
  if (!level) return null;

  const variant =
    level === "high"
      ? "destructive-soft"
      : level === "medium"
        ? "warning-soft"
        : "success-soft";

  return (
    <Badge variant={variant} className="rounded-full text-[10px] capitalize">
      {level}
    </Badge>
  );
}

/** Confirmation code display used during agent registration. */
export function ConfirmationCodeBanner({
  code,
  copyable = false,
  description = "Share this code with the agent to authorize the action",
}: {
  code: string;
  copyable?: boolean;
  description?: string;
}) {
  return (
    <div className="bg-primary/10 border-primary/20 rounded-md border px-4 py-3 text-center">
      <p className="text-muted-foreground mb-1 text-xs">Confirmation code</p>
      {copyable ? (
        <CopyableCode
          code={code}
          className="text-lg font-bold tracking-widest"
        />
      ) : (
        <p className="font-mono text-lg font-bold tracking-widest">{code}</p>
      )}
      <p className="text-muted-foreground mt-1 text-xs">
        {description}
      </p>
    </div>
  );
}
