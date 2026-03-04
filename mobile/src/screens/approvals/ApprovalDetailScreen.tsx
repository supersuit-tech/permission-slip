/**
 * Approval detail screen — displays the full details of a single approval
 * request including agent info, action parameters, risk level, expiry
 * countdown, context, and timeline. For pending approvals, shows
 * approve/deny buttons with haptic feedback.
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
  buildActionSummary,
  safeParams,
  isExpired as checkExpired,
  formatTimestamp,
} from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";
import { ApprovalActions } from "./ApprovalActions";
import { KeyValueList, type KeyValueEntry } from "./KeyValueList";

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalDetail">;

export default function ApprovalDetailScreen({ route, navigation }: Props) {
  const { approval } = route.params;
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();

  const [isApproved, setIsApproved] = useState(false);
  const [isDenied, setIsDenied] = useState(false);
  // Capture the exact timestamp when the user approves/denies so the timeline
  // doesn't flicker with a new Date() on every re-render.
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

  const summary = buildActionSummary(approval.action.type, parameters);

  const timelineEntries: KeyValueEntry[] = useMemo(() => {
    const entries: KeyValueEntry[] = [
      { label: "Created", value: formatTimestamp(approval.created_at) },
      { label: "Expires", value: formatTimestamp(approval.expires_at) },
    ];
    const approvedTime = approval.approved_at ?? (isApproved ? resolvedAt : null);
    if (approvedTime) {
      entries.push({ label: "Approved", value: formatTimestamp(approvedTime) });
    }
    const deniedTime = approval.denied_at ?? (isDenied ? resolvedAt : null);
    if (deniedTime) {
      entries.push({ label: "Denied", value: formatTimestamp(deniedTime) });
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
          <View style={styles.section}>
            <View style={styles.successBanner} accessibilityRole="alert">
              <Text style={styles.successBannerText}>Request Approved</Text>
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
          <View style={styles.section}>
            <View
              style={styles.statusBannerDenied}
              accessibilityRole="alert"
              testID="denied-banner"
            >
              <Text style={styles.statusBannerTextDenied}>Request Denied</Text>
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

        {/* Status banner for already-resolved approvals (loaded from list) */}
        {!isApproved && !isDenied && (
          <>
            {approval.status === "approved" && (
              <View
                style={styles.statusBannerApproved}
                accessibilityRole="alert"
              >
                <Text style={styles.statusBannerTextApproved}>Approved</Text>
              </View>
            )}
            {approval.status === "denied" && (
              <View
                style={styles.statusBannerDenied}
                accessibilityRole="alert"
              >
                <Text style={styles.statusBannerTextDenied}>Denied</Text>
              </View>
            )}
            {approval.status === "cancelled" && (
              <View style={styles.statusBannerCancelled}>
                <Text style={styles.statusBannerText}>Cancelled</Text>
              </View>
            )}
            {expired && (
              <View style={styles.statusBannerCancelled}>
                <Text style={styles.statusBannerText}>Expired</Text>
              </View>
            )}
          </>
        )}

        {/* Agent Info */}
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Agent</Text>
          <View style={styles.agentRow}>
            <View style={styles.agentAvatar}>
              <Text style={styles.agentAvatarText}>
                {agentName.charAt(0).toUpperCase()}
              </Text>
            </View>
            <View style={styles.agentInfo}>
              <Text style={styles.agentName}>{agentName}</Text>
              {agentName !== `Agent ${approval.agent_id}` && (
                <Text style={styles.agentId}>ID: {approval.agent_id}</Text>
              )}
            </View>
          </View>
        </View>

        {/* Action & Risk */}
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Action</Text>
          <View style={styles.card}>
            <View style={styles.actionHeader}>
              <View style={styles.actionTitleRow}>
                <Text style={styles.actionName}>
                  {humanizeActionType(approval.action.type)}
                </Text>
                <RiskBadge level={approval.context.risk_level} />
              </View>
              <Text style={styles.actionType}>{approval.action.type}</Text>
              {approval.action.version && (
                <Text style={styles.actionVersion}>
                  v{approval.action.version}
                </Text>
              )}
            </View>
            {summary !== humanizeActionType(approval.action.type) && (
              <Text style={styles.actionSummary}>{summary}</Text>
            )}

            {approval.context.description && (
              <View style={styles.contextInline}>
                <Text style={styles.contextDescription}>
                  {approval.context.description}
                </Text>
              </View>
            )}

            {approval.context.risk_level &&
              approval.context.risk_level !== "low" &&
              RISK_DESCRIPTIONS[approval.context.risk_level] != null && (
                <View style={styles.riskRow}>
                  <Text style={styles.riskDescription}>
                    {RISK_DESCRIPTIONS[approval.context.risk_level]}
                  </Text>
                </View>
              )}
          </View>
        </View>

        {/* High risk warning */}
        {approval.context.risk_level === "high" && (
          <View style={styles.section}>
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
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>Expiry</Text>
            <View style={styles.card}>
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
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>Parameters</Text>
            <View style={styles.card}>
              <KeyValueList entries={paramEntries} />
            </View>
          </View>
        )}

        {/* Context Details */}
        {contextDetailEntries.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>Additional Context</Text>
            <View style={styles.card}>
              <KeyValueList entries={contextDetailEntries} />
            </View>
          </View>
        )}

        {/* Timeline */}
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Timeline</Text>
          <View style={styles.card}>
            <KeyValueList entries={timelineEntries} />
          </View>
        </View>

        {/* Approval ID */}
        <View style={styles.section}>
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
  low: "Minimal impact, easily reversible",
  medium: "Moderate impact, some consequences",
  high: "Significant impact, hard to reverse",
};


const styles = StyleSheet.create({
  outerContainer: {
    flex: 1,
    backgroundColor: colors.gray50,
  },
  container: {
    flex: 1,
  },
  successBanner: {
    backgroundColor: colors.riskLowBg,
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
    borderWidth: 1,
    borderColor: "#A7F3D0",
    marginBottom: 16,
  },
  successBannerText: {
    fontSize: 16,
    fontWeight: "700",
    color: colors.success,
  },
  statusBannerApproved: {
    backgroundColor: colors.riskLowBg,
    paddingVertical: 10,
    alignItems: "center",
  },
  statusBannerDenied: {
    backgroundColor: colors.riskHighBg,
    paddingVertical: 10,
    alignItems: "center",
  },
  statusBannerCancelled: {
    backgroundColor: colors.gray100,
    paddingVertical: 10,
    alignItems: "center",
  },
  statusBannerText: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.gray700,
  },
  statusBannerTextApproved: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.success,
  },
  statusBannerTextDenied: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.error,
  },
  section: {
    paddingHorizontal: 20,
    marginTop: 20,
  },
  sectionLabel: {
    fontSize: 12,
    fontWeight: "600",
    color: colors.gray400,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    padding: 16,
    borderWidth: 1,
    borderColor: colors.gray200,
  },
  agentRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
  },
  agentAvatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: colors.gray200,
    alignItems: "center",
    justifyContent: "center",
  },
  agentAvatarText: {
    fontSize: 16,
    fontWeight: "700",
    color: colors.gray700,
  },
  agentInfo: {
    flex: 1,
  },
  agentName: {
    fontSize: 16,
    fontWeight: "600",
    color: colors.gray900,
  },
  agentId: {
    fontSize: 12,
    color: colors.gray400,
    marginTop: 2,
  },
  actionHeader: {
    marginBottom: 4,
  },
  actionTitleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    flexWrap: "wrap",
  },
  actionName: {
    fontSize: 17,
    fontWeight: "700",
    color: colors.gray900,
  },
  actionType: {
    fontSize: 12,
    color: colors.gray400,
    fontFamily: "monospace",
    marginTop: 2,
  },
  actionVersion: {
    fontSize: 11,
    color: colors.gray400,
    marginTop: 2,
  },
  actionSummary: {
    fontSize: 14,
    color: colors.gray500,
    marginTop: 8,
    lineHeight: 20,
  },
  contextInline: {
    marginTop: 10,
    paddingTop: 10,
    borderTopWidth: 1,
    borderTopColor: colors.gray100,
  },
  contextDescription: {
    fontSize: 15,
    color: colors.gray700,
    lineHeight: 22,
  },
  riskRow: {
    marginTop: 8,
  },
  riskDescription: {
    fontSize: 13,
    color: colors.gray500,
    fontStyle: "italic",
  },
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
  expiryRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  expiryLabel: {
    fontSize: 14,
    color: colors.gray500,
  },
  footerLabel: {
    fontSize: 11,
    color: colors.gray400,
    textAlign: "center",
    fontFamily: "monospace",
  },
  doneButton: {
    marginTop: 16,
    backgroundColor: colors.gray900,
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: "center",
  },
  doneButtonText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
  actionBar: {
    backgroundColor: colors.white,
    borderTopWidth: 1,
    borderTopColor: colors.gray200,
  },
});
