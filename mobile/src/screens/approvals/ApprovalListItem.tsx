import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { colors } from "../../theme/colors";
import type { ApprovalSummary } from "../../hooks/useApprovals";

interface ApprovalListItemProps {
  approval: ApprovalSummary;
  onPress: (approvalId: string) => void;
}

const riskColors: Record<string, string> = {
  low: colors.riskLow,
  medium: colors.riskMedium,
  high: colors.riskHigh,
};

const statusStyles: Record<string, { bg: string; text: string }> = {
  pending: { bg: colors.pendingBg, text: colors.pendingText },
  approved: { bg: colors.approvedBg, text: colors.approvedText },
  denied: { bg: colors.deniedBg, text: colors.deniedText },
};

/** Formats "email.send" → "Email: Send" */
function formatActionType(type: string): string {
  const parts = type.split(".");
  if (parts.length < 2) return type;
  const category = parts[0] ?? "";
  const operation = parts[1] ?? "";
  return `${capitalize(category)}: ${capitalize(operation.replace(/_/g, " "))}`;
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** Returns a human-readable relative time string (e.g. "5m ago", "2h ago"). */
function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;

  if (diffMs < 0) return "just now";

  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

/** Returns a human-readable time-remaining string, or null if already expired. */
function timeRemaining(dateStr: string): { text: string; urgent: boolean } | null {
  const now = Date.now();
  const expires = new Date(dateStr).getTime();
  const diffMs = expires - now;

  if (diffMs <= 0) return { text: "Expired", urgent: true };

  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return { text: "< 1m left", urgent: true };
  if (minutes < 60) return { text: `${minutes}m left`, urgent: minutes <= 5 };

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return { text: `${hours}h left`, urgent: false };

  const days = Math.floor(hours / 24);
  return { text: `${days}d left`, urgent: false };
}

export default function ApprovalListItem({ approval, onPress }: ApprovalListItemProps) {
  const risk = approval.context.risk_level ?? "low";
  const riskColor = riskColors[risk] ?? colors.gray500;
  const badge = statusStyles[approval.status] ?? statusStyles.pending;
  const expiry = approval.status === "pending"
    ? timeRemaining(approval.expires_at)
    : null;

  return (
    <TouchableOpacity
      testID={`approval-item-${approval.approval_id}`}
      accessibilityRole="button"
      accessibilityLabel={`${approval.context.description}, ${approval.status}`}
      activeOpacity={0.7}
      style={styles.container}
      onPress={() => onPress(approval.approval_id)}
    >
      <View style={styles.header}>
        <Text style={styles.actionType} numberOfLines={1}>
          {formatActionType(approval.action.type)}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: badge.bg }]}>
          <Text style={[styles.statusText, { color: badge.text }]}>
            {capitalize(approval.status)}
          </Text>
        </View>
      </View>

      <Text style={styles.description} numberOfLines={2}>
        {approval.context.description}
      </Text>

      <View style={styles.footer}>
        <View style={styles.riskContainer}>
          <View style={[styles.riskDot, { backgroundColor: riskColor }]} />
          <Text style={styles.riskLabel}>{capitalize(risk)} risk</Text>
        </View>
        <View style={styles.footerRight}>
          {expiry && (
            <Text
              style={[styles.expiry, expiry.urgent && styles.expiryUrgent]}
            >
              {expiry.text}
            </Text>
          )}
          <Text style={styles.time}>{timeAgo(approval.created_at)}</Text>
        </View>
      </View>
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    marginHorizontal: 16,
    marginBottom: 10,
    borderWidth: 1,
    borderColor: colors.gray200,
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 8,
  },
  actionType: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
    flex: 1,
    marginRight: 8,
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 6,
  },
  statusText: {
    fontSize: 12,
    fontWeight: "600",
  },
  description: {
    fontSize: 14,
    color: colors.gray500,
    lineHeight: 20,
    marginBottom: 10,
  },
  footer: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
  },
  riskContainer: {
    flexDirection: "row",
    alignItems: "center",
  },
  riskDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 6,
  },
  riskLabel: {
    fontSize: 12,
    color: colors.gray500,
  },
  footerRight: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  expiry: {
    fontSize: 12,
    color: colors.gray500,
  },
  expiryUrgent: {
    color: colors.warning,
    fontWeight: "600",
  },
  time: {
    fontSize: 12,
    color: colors.gray400,
  },
});
