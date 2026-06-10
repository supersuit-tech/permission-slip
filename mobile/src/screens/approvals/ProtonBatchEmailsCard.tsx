/**
 * Per-message preview for Proton Mail batch actions (archive, mark read,
 * move, delete, …). Each email is its own collapsible row so approvers can
 * inspect every message in the batch individually.
 */
import { useState } from "react";
import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { colors } from "../../theme/colors";
import { formatTimestamp } from "./approvalUtils";
import type { ProtonBatchEmail } from "./protonBatchEmailsUtils";

interface ProtonBatchEmailsCardProps {
  emails: ProtonBatchEmail[];
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

function EmailRow({
  email,
  expanded,
  onToggle,
}: {
  email: ProtonBatchEmail;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <View style={styles.row}>
      <TouchableOpacity
        onPress={onToggle}
        accessibilityRole="button"
        accessibilityState={{ expanded }}
        accessibilityLabel={`Toggle details for ${email.subject || "email " + email.uid}`}
        testID={`batch-email-toggle-${email.uid}`}
        style={styles.rowHeader}
      >
        <Text style={[styles.chevron, expanded && styles.chevronExpanded]}>
          {"▸"}
        </Text>
        <Text style={styles.subject} numberOfLines={expanded ? undefined : 1}>
          {email.subject || "(No subject)"}
        </Text>
      </TouchableOpacity>
      {expanded && (
        <View style={styles.rowDetails} testID={`batch-email-details-${email.uid}`}>
          <AddressLine label="From" values={email.from} />
          <AddressLine label="To" values={email.to} />
          {email.date ? (
            <Text style={styles.metaLine} selectable>
              {formatTimestamp(email.date)}
            </Text>
          ) : null}
          <Text style={styles.metaLine} selectable>
            <Text style={styles.metaLabel}>Message id: </Text>
            {email.uid}
          </Text>
        </View>
      )}
    </View>
  );
}

export function ProtonBatchEmailsCard({
  emails,
  testID = "proton-batch-emails-card",
}: ProtonBatchEmailsCardProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (uid: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(uid)) {
        next.delete(uid);
      } else {
        next.add(uid);
      }
      return next;
    });
  };

  return (
    <View style={styles.card} testID={testID}>
      {emails.map((email, index) => (
        <View key={email.uid}>
          {index > 0 && <View style={styles.divider} />}
          <EmailRow
            email={email}
            expanded={expanded.has(email.uid)}
            onToggle={() => toggle(email.uid)}
          />
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    overflow: "hidden",
  },
  divider: {
    height: 1,
    backgroundColor: colors.gray200,
  },
  row: {
    paddingVertical: 2,
  },
  rowHeader: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: 14,
    paddingVertical: 10,
  },
  chevron: {
    color: colors.gray400,
    fontSize: 13,
    width: 14,
  },
  chevronExpanded: {
    transform: [{ rotate: "90deg" }],
  },
  subject: {
    flex: 1,
    color: colors.gray900,
    fontSize: 14,
    fontWeight: "600",
    lineHeight: 20,
  },
  rowDetails: {
    paddingHorizontal: 14,
    paddingBottom: 12,
    paddingLeft: 36,
    gap: 4,
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
