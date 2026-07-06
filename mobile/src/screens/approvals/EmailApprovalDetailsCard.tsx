import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";
import { formatTimestamp } from "./approvalUtils";
import {
  hasPartialEmailApprovalDetails,
  parseEmailApprovalDetails,
} from "./emailApprovalDetails";

interface EmailApprovalDetailsCardProps {
  actionType: string;
  resourceDetails?: Record<string, unknown> | null;
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.row}>
      <Text style={styles.label}>{label}</Text>
      <Text style={styles.value}>{value}</Text>
    </View>
  );
}

export function EmailApprovalDetailsCard({
  actionType,
  resourceDetails,
}: EmailApprovalDetailsCardProps) {
  const details = parseEmailApprovalDetails(actionType, resourceDetails);
  if (!hasPartialEmailApprovalDetails(details)) {
    return null;
  }

  const dateLabel =
    details?.date != null
      ? (() => {
          const formatted = formatTimestamp(details.date);
          return formatted.length > 0 ? formatted : details.date;
        })()
      : null;

  return (
    <View style={styles.card} testID="email-approval-details">
      {details?.from != null && <DetailRow label="From" value={details.from} />}
      {details?.subject != null && <DetailRow label="Subject" value={details.subject} />}
      {dateLabel != null && <DetailRow label="Date" value={dateLabel} />}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
    gap: 10,
  },
  row: {
    gap: 2,
  },
  label: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray400,
    textTransform: "uppercase",
    letterSpacing: 0.5,
  },
  value: {
    fontSize: 14,
    color: colors.gray900,
    lineHeight: 20,
  },
});
