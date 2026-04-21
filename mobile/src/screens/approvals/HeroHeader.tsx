/**
 * Hero header for the approval detail screen — consolidates status,
 * action info, and agent metadata into a prominent top section.
 */
import { StyleSheet, Text, View } from "react-native";
import { colors } from "../../theme/colors";
import { RiskBadge } from "./RiskBadge";
import { StatusPill } from "./StatusPill";
import { getAvatarColors, formatTimestamp } from "./approvalUtils";

interface HeroHeaderProps {
  actionName: string;
  actionType: string;
  /** e.g. "Slack (Engineering)" when a connector instance is targeted */
  connectorDisplayName?: string | null;
  actionVersion?: string;
  summary: string;
  riskLevel?: "low" | "medium" | "high" | null;
  /** Display status — resolved from approval status + local state. */
  displayStatus: "approved" | "denied" | "cancelled" | "expired" | "pending";
  /** Whether to show the status pill. False during live confirmation flow to avoid duplicate pills. */
  showStatus?: boolean;
  /** Risk description text shown for medium/high risk levels. */
  riskDescription?: string | null;
  agentName: string;
  createdAt: string;
  contextDescription?: string | null;
}

export function HeroHeader({
  actionName,
  actionType,
  connectorDisplayName,
  actionVersion,
  summary,
  riskLevel,
  displayStatus,
  showStatus: showStatusProp,
  riskDescription,
  agentName,
  createdAt,
  contextDescription,
}: HeroHeaderProps) {
  const avatarColors = getAvatarColors(agentName);
  const initial = agentName.charAt(0).toUpperCase();
  const showStatus = displayStatus !== "pending" && (showStatusProp ?? true);

  return (
    <View style={styles.container}>
      {/* Title row: action name + risk badge + status pill */}
      <View style={styles.topRow}>
        <View style={styles.titleArea}>
          <View style={styles.titleRow}>
            <Text style={styles.title}>{actionName}</Text>
            <RiskBadge level={riskLevel} />
          </View>
          <Text style={styles.actionType}>
            {actionType}
            {actionVersion ? `  v${actionVersion}` : ""}
          </Text>
          {connectorDisplayName ? (
            <Text style={styles.connectorLine}>{connectorDisplayName}</Text>
          ) : null}
        </View>
        {showStatus && <StatusPill status={displayStatus} />}
      </View>

      {/* Summary (if different from action name) */}
      {summary !== actionName && (
        <Text style={styles.summary}>{summary}</Text>
      )}

      {/* Risk description for medium/high risk */}
      {riskDescription ? (
        <Text style={styles.riskDescription}>{riskDescription}</Text>
      ) : null}

      {/* Context description */}
      {contextDescription ? (
        <View style={styles.descriptionDivider}>
          <Text style={styles.description}>{contextDescription}</Text>
        </View>
      ) : null}

      {/* Metadata chip row: avatar + agent name | timestamp */}
      <View style={styles.metadataRow}>
        <View style={styles.metadataChip}>
          <View
            style={[styles.smallAvatar, { backgroundColor: avatarColors.bg }]}
          >
            <Text style={[styles.smallAvatarText, { color: avatarColors.text }]}>
              {initial}
            </Text>
          </View>
          <Text style={styles.metadataText}>{agentName}</Text>
        </View>
        <View style={styles.metadataDot} />
        <Text style={styles.metadataText}>{formatTimestamp(createdAt)}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: colors.white,
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 16,
    // Subtle shadow to separate hero from content below
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.08,
    shadowRadius: 12,
    elevation: 3,
  },
  topRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
  },
  titleArea: {
    flex: 1,
    marginRight: 12,
  },
  titleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    flexWrap: "wrap",
  },
  title: {
    fontSize: 20,
    fontWeight: "700",
    color: colors.gray900,
  },
  actionType: {
    fontSize: 12,
    color: colors.gray400,
    fontFamily: "monospace",
    marginTop: 4,
  },
  connectorLine: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.gray700,
    marginTop: 6,
  },
  summary: {
    fontSize: 15,
    color: colors.gray500,
    lineHeight: 22,
    marginTop: 10,
  },
  riskDescription: {
    fontSize: 13,
    color: colors.gray500,
    fontStyle: "italic",
    marginTop: 6,
  },
  descriptionDivider: {
    marginTop: 12,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: colors.gray100,
  },
  description: {
    fontSize: 15,
    color: colors.gray700,
    lineHeight: 22,
  },
  metadataRow: {
    flexDirection: "row",
    alignItems: "center",
    marginTop: 16,
    gap: 8,
  },
  metadataChip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  metadataDot: {
    width: 3,
    height: 3,
    borderRadius: 1.5,
    backgroundColor: colors.gray300,
  },
  metadataText: {
    fontSize: 13,
    color: colors.gray500,
  },
  smallAvatar: {
    width: 22,
    height: 22,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center",
  },
  smallAvatarText: {
    fontSize: 11,
    fontWeight: "700",
  },
});
