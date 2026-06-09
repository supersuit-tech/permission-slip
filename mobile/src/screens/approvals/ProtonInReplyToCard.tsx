/**
 * Metadata-only preview of the email being replied to for protonmail.reply_email.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";
import { formatTimestamp } from "./approvalUtils";
import type { ProtonInReplyToMetadata } from "./protonInReplyToUtils";

interface ProtonInReplyToCardProps {
  metadata: ProtonInReplyToMetadata | null;
  testID?: string;
}

function AddressLine({ label, values }: { label: string; values: string[] }) {
  if (values.length === 0) return null;
  return (
    <Text style={styles.metaLine} selectable>
      <Text style={styles.metaLabel}>{label}: </Text>
      {values.join(", ")}
    </Text>
  );
}

export function ProtonInReplyToCard({
  metadata,
  testID = "proton-in-reply-to-card",
}: ProtonInReplyToCardProps) {
  if (!metadata) {
    return (
      <View style={styles.emptyCard} testID={testID}>
        <Text style={styles.emptyText}>
          No source email details were included with this reply request.
        </Text>
      </View>
    );
  }

  return (
    <View style={styles.card} testID={testID}>
      <Text style={styles.sectionEyebrow}>In reply to</Text>
      <Text style={styles.subject} selectable>
        {metadata.subject || "(No subject)"}
      </Text>
      <AddressLine label="From" values={metadata.from} />
      <AddressLine label="To" values={metadata.to} />
      {metadata.date ? (
        <Text style={styles.metaLine} selectable>
          {formatTimestamp(metadata.date)}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    padding: 14,
    gap: 6,
  },
  emptyCard: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    borderStyle: "dashed",
    padding: 14,
  },
  emptyText: {
    color: colors.gray500,
    fontSize: 14,
    lineHeight: 20,
  },
  sectionEyebrow: {
    color: colors.gray500,
    fontSize: 11,
    fontWeight: "600",
    letterSpacing: 0.6,
    textTransform: "uppercase",
  },
  subject: {
    color: colors.gray900,
    fontSize: 16,
    fontWeight: "600",
    lineHeight: 22,
  },
  metaLine: {
    color: colors.gray500,
    fontSize: 13,
    lineHeight: 18,
  },
  metaLabel: {
    color: colors.gray900,
    fontWeight: "500",
  },
});
