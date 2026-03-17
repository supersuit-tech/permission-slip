/**
 * Compact status pill badge — shows approval status as a colored pill.
 * Follows the same visual pattern as RiskBadge.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";

type Status = "approved" | "denied" | "cancelled" | "expired" | "pending";

interface StatusPillProps {
  status: Status;
}

export function StatusPill({ status }: StatusPillProps) {
  const config = STATUS_STYLES[status];
  return (
    <View
      style={[styles.badge, { backgroundColor: config.bg }]}
      accessibilityLabel={`Status: ${status}`}
    >
      <Text style={[styles.text, { color: config.text }]}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Text>
    </View>
  );
}

const STATUS_STYLES: Record<Status, { bg: string; text: string }> = {
  approved: { bg: colors.approvedBg, text: colors.approvedText },
  denied: { bg: colors.deniedBg, text: colors.deniedText },
  pending: { bg: colors.pendingBg, text: colors.pendingText },
  cancelled: { bg: colors.cancelledBg, text: colors.cancelledText },
  expired: { bg: colors.cancelledBg, text: colors.cancelledText },
} as const;

const styles = StyleSheet.create({
  badge: {
    paddingHorizontal: 10,
    paddingVertical: 3,
    borderRadius: 12,
  },
  text: {
    fontSize: 12,
    fontWeight: "600",
  },
});
