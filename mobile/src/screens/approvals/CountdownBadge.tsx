/**
 * Countdown badge and shared timer infrastructure for approval expiry.
 * Uses a single shared setInterval so multiple CountdownBadge instances
 * (e.g. in a list) don't each create their own timer.
 */
import { useEffect, useState } from "react";
import { StyleSheet, Text } from "react-native";
import { colors } from "../../theme/colors";
import { secondsUntil, formatCountdown } from "./approvalUtils";

/** Returns a color based on remaining seconds. */
function urgencyColor(seconds: number): string {
  if (seconds <= 0) return colors.gray400;
  if (seconds <= 60) return colors.error;
  if (seconds <= 120) return colors.warning;
  return colors.gray500;
}

/**
 * Shared 1-second ticker so all countdown consumers share a single interval
 * instead of each component creating its own setInterval.
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
 * a single shared interval (no per-component timers).
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

interface CountdownBadgeProps {
  expiresAt: string;
}

/**
 * Displays a live-updating countdown (e.g. "4:32") that changes color
 * based on urgency: gray (expired), red (<=60s), amber (<=120s).
 */
export function CountdownBadge({ expiresAt }: CountdownBadgeProps) {
  const remaining = useCountdown(expiresAt);
  const isExpired = remaining <= 0;

  return (
    <Text
      style={[styles.text, { color: urgencyColor(remaining) }]}
      accessibilityLabel={
        isExpired ? "Expired" : `${formatCountdown(remaining)} remaining`
      }
    >
      {isExpired ? "Expired" : formatCountdown(remaining)}
    </Text>
  );
}

const styles = StyleSheet.create({
  text: {
    fontSize: 12,
    fontWeight: "600",
  },
});
