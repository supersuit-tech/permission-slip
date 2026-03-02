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

/** Live countdown hook: recalculates every second. */
export function useCountdown(expiresAt: string): number {
  const [remaining, setRemaining] = useState(() => secondsUntil(expiresAt));

  useEffect(() => {
    const timer = setInterval(() => {
      setRemaining(secondsUntil(expiresAt));
    }, 1_000);
    return () => clearInterval(timer);
  }, [expiresAt]);

  return remaining;
}

interface CountdownBadgeProps {
  expiresAt: string;
}

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
