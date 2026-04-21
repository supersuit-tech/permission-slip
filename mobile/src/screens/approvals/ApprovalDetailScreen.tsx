/**
 * Approval detail screen — displays the full details of a single approval
 * request with a hero header, action parameters, timeline, and
 * approve/deny buttons. Uses subtle shadows and visual hierarchy
 * for a polished, first-class feel.
 */
import { useCallback, useMemo, useState } from "react";
import {
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import * as Haptics from "expo-haptics";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useAgents, getAgentDisplayName } from "../../hooks/useAgents";
import { useApproveApproval } from "../../hooks/useApproveApproval";
import { useDenyApproval } from "../../hooks/useDenyApproval";
import { colors } from "../../theme/colors";
import {
  humanizeActionType,
  humanizeConnectorPrefix,
  connectorInstanceLabelFromAction,
  buildActionSummary,
  safeParams,
  isExpired as checkExpired,
  formatTimestamp,
} from "./approvalUtils";
import { HeroHeader } from "./HeroHeader";
import { StatusPill } from "./StatusPill";
import { CountdownBadge } from "./CountdownBadge";
import { ApprovalActions } from "./ApprovalActions";
import { KeyValueList, type KeyValueEntry } from "./KeyValueList";
import { TimelineView, type TimelineEntry } from "./TimelineView";

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalDetail">;

export default function ApprovalDetailScreen({ route, navigation }: Props) {
  const { approval } = route.params;
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();

  const [isApproved, setIsApproved] = useState(false);
  const [isDenied, setIsDenied] = useState(false);
  const [resolvedAt, setResolvedAt] = useState<string | null>(null);

  const {
    approveApproval,
    isPending: isApproving,
  } = useApproveApproval();
  const {
    denyApproval,
    isPending: isDenying,
  } = useDenyApproval();

  const agent = useMemo(
    () => agents.find((a) => a.agent_id === approval.agent_id),
    [agents, approval.agent_id],
  );
  const agentName = agent
    ? getAgentDisplayName(agent)
    : `Agent ${approval.agent_id}`;

  const parameters = safeParams(approval.action.parameters);
  const paramEntries: KeyValueEntry[] = useMemo(
    () => Object.entries(parameters).map(([label, value]) => ({ label, value })),
    [parameters],
  );
  const contextDetailEntries: KeyValueEntry[] = useMemo(() => {
    const details = safeParams(approval.context.details);
    return Object.entries(details).map(([label, value]) => ({ label, value }));
  }, [approval.context.details]);

  const isPending = approval.status === "pending" && !isApproved && !isDenied;
  const expired = checkExpired(approval.status, approval.expires_at);
  const canAct = isPending && !expired;

  const summary = buildActionSummary(approval.action.type, parameters, undefined, approval.resource_details as Record<string, unknown> | undefined);
  const actionName = humanizeActionType(approval.action.type);
  const instanceLabel = connectorInstanceLabelFromAction(
    approval.action as { _connector_instance_label?: unknown },
  );
  const connectorDisplayName = instanceLabel
    ? `${humanizeConnectorPrefix(approval.action.type)} (${instanceLabel})`
    : null;

  // Derive display status for the hero header
  const displayStatus = useMemo(() => {
    if (isApproved) return "approved" as const;
    if (isDenied) return "denied" as const;
    if (expired) return "expired" as const;
    if (approval.status === "cancelled") return "cancelled" as const;
    if (approval.status === "approved") return "approved" as const;
    if (approval.status === "denied") return "denied" as const;
    return "pending" as const;
  }, [approval.status, isApproved, isDenied, expired]);

  const timelineEntries: TimelineEntry[] = useMemo(() => {
    const entries: TimelineEntry[] = [
      { label: "Created", value: formatTimestamp(approval.created_at) },
      { label: "Expires", value: formatTimestamp(approval.expires_at) },
    ];
    const approvedTime = approval.approved_at ?? (isApproved ? resolvedAt : null);
    if (approvedTime) {
      entries.push({
        label: "Approved",
        value: formatTimestamp(approvedTime),
        dotColor: "success",
      });
    }
    const deniedTime = approval.denied_at ?? (isDenied ? resolvedAt : null);
    if (deniedTime) {
      entries.push({
        label: "Denied",
        value: formatTimestamp(deniedTime),
        dotColor: "error",
      });
    }
    if (approval.cancelled_at) {
      entries.push({ label: "Cancelled", value: formatTimestamp(approval.cancelled_at) });
    }
    return entries;
  }, [approval, isApproved, isDenied, resolvedAt]);

  const handleApprove = useCallback(async () => {
    try {
      await approveApproval(approval.approval_id);
      setResolvedAt(new Date().toISOString());
      setIsApproved(true);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch (err) {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
      const message =
        err instanceof Error ? err.message : "Failed to approve request";
      Alert.alert("Approval Failed", message);
    }
  }, [approveApproval, approval.approval_id]);

  const handleDeny = useCallback(() => {
    Alert.alert(
      "Deny Request",
      "Are you sure you want to deny this request?",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Deny",
          style: "destructive",
          onPress: async () => {
            try {
              await denyApproval(approval.approval_id);
              setResolvedAt(new Date().toISOString());
              setIsDenied(true);
              Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
            } catch (err) {
              Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
              const message =
                err instanceof Error ? err.message : "Failed to deny request";
              Alert.alert("Denial Failed", message);
            }
          },
        },
      ],
    );
  }, [denyApproval, approval.approval_id]);

  const handleDone = useCallback(() => {
    navigation.goBack();
  }, [navigation]);

  return (
    <View style={styles.outerContainer}>
      <ScrollView
        style={styles.container}
        contentContainerStyle={{ paddingBottom: canAct ? 8 : insets.bottom + 24 }}
      >
        {/* Success state — shown after successful approval */}
        {isApproved && (
          <View style={styles.confirmationSection}>
            <View style={styles.confirmationCard} accessibilityRole="alert">
              <StatusPill status="approved" />
              <Text style={styles.confirmationText}>Request Approved</Text>
            </View>
            <TouchableOpacity
              style={styles.doneButton}
              onPress={handleDone}
              accessibilityLabel="Done, go back to list"
              accessibilityRole="button"
              testID="done-button"
            >
              <Text style={styles.doneButtonText}>Done</Text>
            </TouchableOpacity>
          </View>
        )}

        {/* Denied confirmation */}
        {isDenied && (
          <View style={styles.confirmationSection}>
            <View
              style={styles.confirmationCard}
              accessibilityRole="alert"
              testID="denied-banner"
            >
              <StatusPill status="denied" />
              <Text style={styles.confirmationText}>Request Denied</Text>
            </View>
            <TouchableOpacity
              style={styles.doneButton}
              onPress={handleDone}
              accessibilityLabel="Done, go back to list"
              accessibilityRole="button"
              testID="done-button"
            >
              <Text style={styles.doneButtonText}>Done</Text>
            </TouchableOpacity>
          </View>
        )}

        {/* Hero Header — action info, status, agent, timestamp */}
        <HeroHeader
          actionName={actionName}
          actionType={approval.action.type}
          connectorDisplayName={connectorDisplayName}
          actionVersion={approval.action.version}
          summary={summary}
          riskLevel={approval.context.risk_level}
          displayStatus={displayStatus}
          showStatus={!isApproved && !isDenied}
          riskDescription={
            approval.context.risk_level && approval.context.risk_level !== "low"
              ? RISK_DESCRIPTIONS[approval.context.risk_level]
              : null
          }
          agentName={agentName}
          createdAt={approval.created_at}
          contextDescription={approval.context.description}
        />

        {/* High risk warning */}
        {approval.context.risk_level === "high" && (
          <View style={styles.sectionMinor}>
            <View
              style={styles.highRiskWarning}
              accessibilityRole="alert"
            >
              <Text style={styles.highRiskWarningText}>
                This is a high-risk action. Review the details carefully before
                approving.
              </Text>
            </View>
          </View>
        )}

        {/* Expiry Countdown */}
        {approval.status === "pending" && !isApproved && !isDenied && (
          <View style={styles.sectionMinor}>
            <Text style={styles.sectionLabel}>Expiry</Text>
            <View style={styles.cardElevated}>
              <View style={styles.expiryRow}>
                <CountdownBadge expiresAt={approval.expires_at} />
                <Text style={styles.expiryLabel}>
                  {expired ? "This request has expired" : "remaining"}
                </Text>
              </View>
            </View>
          </View>
        )}

        {/* Parameters */}
        {paramEntries.length > 0 && (
          <View style={styles.sectionMajor}>
            <Text style={styles.sectionLabel}>Parameters</Text>
            <View style={styles.cardElevated}>
              <KeyValueList entries={paramEntries} />
            </View>
          </View>
        )}

        {/* Context Details */}
        {contextDetailEntries.length > 0 && (
          <View style={styles.sectionMinor}>
            <Text style={styles.sectionLabel}>Additional Context</Text>
            <View style={styles.cardElevated}>
              <KeyValueList entries={contextDetailEntries} />
            </View>
          </View>
        )}

        {/* Timeline */}
        <View style={styles.sectionMajor}>
          <Text style={styles.sectionLabel}>Timeline</Text>
          <View style={styles.cardElevated}>
            <TimelineView entries={timelineEntries} />
          </View>
        </View>

        {/* Approval ID */}
        <View style={styles.sectionMajor}>
          <Text style={styles.footerLabel}>
            ID: {approval.approval_id}
          </Text>
        </View>
      </ScrollView>

      {/* Action buttons — fixed at bottom for pending approvals */}
      {canAct && (
        <View style={[styles.actionBar, { paddingBottom: insets.bottom + 8 }]}>
          <ApprovalActions
            onApprove={handleApprove}
            onDeny={handleDeny}
            isApproving={isApproving}
            isDenying={isDenying}
            disabled={expired}
          />
        </View>
      )}
    </View>
  );
}

/** Human-readable explanations shown inline for each risk level. */
const RISK_DESCRIPTIONS: Record<string, string> = {
  medium: "Moderate impact, some consequences",
  high: "Significant impact, hard to reverse",
};

const styles = StyleSheet.create({
  outerContainer: {
    flex: 1,
    backgroundColor: colors.primaryBg,
  },
  container: {
    flex: 1,
  },
  // --- Confirmation (post-approve/deny) ---
  confirmationSection: {
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 4,
  },
  confirmationCard: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
  },
  confirmationText: {
    fontSize: 16,
    fontWeight: "700",
    color: colors.gray900,
  },
  // --- Sections ---
  sectionMajor: {
    paddingHorizontal: 20,
    marginTop: 24,
  },
  sectionMinor: {
    paddingHorizontal: 20,
    marginTop: 12,
  },
  sectionLabel: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray400,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  // --- Elevated card (shadow instead of border) ---
  cardElevated: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 2,
  },
  // --- Expiry ---
  expiryRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  expiryLabel: {
    fontSize: 14,
    color: colors.gray500,
  },
  // --- High risk ---
  highRiskWarning: {
    backgroundColor: colors.riskHighBg,
    borderRadius: 12,
    padding: 14,
    borderWidth: 1,
    borderColor: "#FECACA",
  },
  highRiskWarningText: {
    fontSize: 13,
    color: colors.riskHigh,
    fontWeight: "500",
  },
  // --- Footer ---
  footerLabel: {
    fontSize: 11,
    color: colors.gray400,
    textAlign: "center",
    fontFamily: "monospace",
  },
  // --- Done button ---
  doneButton: {
    marginTop: 16,
    backgroundColor: colors.primary,
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: "center",
  },
  doneButtonText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
  // --- Action bar ---
  actionBar: {
    backgroundColor: colors.white,
    borderTopWidth: 1,
    borderTopColor: colors.gray200,
  },
});
