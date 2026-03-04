/**
 * Approval detail screen — displays the full details of a single approval
 * request including agent info, action parameters, risk level, expiry
 * countdown, context, and timeline. For pending approvals, shows
 * approve/deny buttons. After approval, displays the confirmation code.
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
  formatParamValue,
  formatTimestamp,
} from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";
import { ConfirmationCodeCard } from "./ConfirmationCodeCard";
import { ApprovalActions } from "./ApprovalActions";

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalDetail">;

export default function ApprovalDetailScreen({ route, navigation }: Props) {
  const { approval } = route.params;
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();

  const [confirmationCode, setConfirmationCode] = useState<string | null>(null);
  const [isDenied, setIsDenied] = useState(false);

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
  const paramEntries = Object.entries(parameters);
  const contextDetails = safeParams(approval.context.details);
  const contextDetailEntries = Object.entries(contextDetails);

  const isPending = approval.status === "pending" && !confirmationCode && !isDenied;
  const expired = checkExpired(approval.status, approval.expires_at);
  const canAct = isPending && !expired;

  const summary = buildActionSummary(approval.action.type, parameters);

  const handleApprove = useCallback(async () => {
    try {
      const result = await approveApproval(approval.approval_id);
      setConfirmationCode(result.confirmation_code);
    } catch (err) {
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
              setIsDenied(true);
            } catch (err) {
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
        {/* Confirmation code — shown after successful approval */}
        {confirmationCode && (
          <View style={styles.section}>
            <View style={styles.successBanner} accessibilityRole="alert">
              <Text style={styles.successBannerText}>Request Approved</Text>
            </View>
            <View style={styles.codeSection}>
              <ConfirmationCodeCard code={confirmationCode} />
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
        {!confirmationCode && !isDenied && (
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

            {/* Context description — inline with the action for quick scanning */}
            {approval.context.description && (
              <View style={styles.contextInline}>
                <Text style={styles.contextDescription}>
                  {approval.context.description}
                </Text>
              </View>
            )}

            {/* Risk explanation — inline for high/medium risk */}
            {approval.context.risk_level &&
              approval.context.risk_level !== "low" && (
                <View style={styles.riskRow}>
                  <Text style={styles.riskDescription}>
                    {RISK_DESCRIPTIONS[approval.context.risk_level]}
                  </Text>
                </View>
              )}
          </View>
        </View>

        {/* High risk warning — prominent, outside the card */}
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
        {approval.status === "pending" && !confirmationCode && !isDenied && (
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
              {paramEntries.map(([key, value], index) => {
                const formatted = formatParamValue(value);
                const isLong = formatted.length > 40 || formatted.includes("\n");
                const isLast = index === paramEntries.length - 1;
                return (
                  <View
                    key={key}
                    style={[
                      isLong ? styles.paramRowVertical : styles.paramRow,
                      isLast && styles.paramRowLast,
                    ]}
                  >
                    <Text style={styles.paramKey}>{key}</Text>
                    <Text
                      style={
                        isLong ? styles.paramValueFull : styles.paramValue
                      }
                      selectable
                    >
                      {formatted}
                    </Text>
                  </View>
                );
              })}
            </View>
          </View>
        )}

        {/* Context Details */}
        {contextDetailEntries.length > 0 && (
          <View style={styles.section}>
            <Text style={styles.sectionLabel}>Additional Context</Text>
            <View style={styles.card}>
              {contextDetailEntries.map(([key, value], index) => {
                const formatted = formatParamValue(value);
                const isLong = formatted.length > 40 || formatted.includes("\n");
                const isLast = index === contextDetailEntries.length - 1;
                return (
                  <View
                    key={key}
                    style={[
                      isLong ? styles.paramRowVertical : styles.paramRow,
                      isLast && styles.paramRowLast,
                    ]}
                  >
                    <Text style={styles.paramKey}>{key}</Text>
                    <Text
                      style={
                        isLong ? styles.paramValueFull : styles.paramValue
                      }
                      selectable
                    >
                      {formatted}
                    </Text>
                  </View>
                );
              })}
            </View>
          </View>
        )}

        {/* Timestamps */}
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Timeline</Text>
          <View style={styles.card}>
            <View style={styles.paramRow}>
              <Text style={styles.paramKey}>Created</Text>
              <Text style={styles.paramValue}>
                {formatTimestamp(approval.created_at)}
              </Text>
            </View>
            <View style={styles.paramRow}>
              <Text style={styles.paramKey}>Expires</Text>
              <Text style={styles.paramValue}>
                {formatTimestamp(approval.expires_at)}
              </Text>
            </View>
            {(approval.approved_at ?? (confirmationCode ? new Date().toISOString() : null)) && (
              <View style={styles.paramRow}>
                <Text style={styles.paramKey}>Approved</Text>
                <Text style={styles.paramValue}>
                  {formatTimestamp(approval.approved_at ?? new Date().toISOString())}
                </Text>
              </View>
            )}
            {(approval.denied_at ?? (isDenied ? new Date().toISOString() : null)) && (
              <View style={styles.paramRow}>
                <Text style={styles.paramKey}>Denied</Text>
                <Text style={styles.paramValue}>
                  {formatTimestamp(approval.denied_at ?? new Date().toISOString())}
                </Text>
              </View>
            )}
            {approval.cancelled_at && (
              <View style={styles.paramRow}>
                <Text style={styles.paramKey}>Cancelled</Text>
                <Text style={styles.paramValue}>
                  {formatTimestamp(approval.cancelled_at)}
                </Text>
              </View>
            )}
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
  codeSection: {
    marginTop: 0,
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
  paramRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  paramRowVertical: {
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  paramRowLast: {
    borderBottomWidth: 0,
  },
  paramKey: {
    fontSize: 13,
    fontWeight: "500",
    color: colors.gray500,
    marginRight: 12,
    flexShrink: 0,
  },
  paramValue: {
    fontSize: 13,
    color: colors.gray900,
    flex: 1,
    textAlign: "right",
  },
  paramValueFull: {
    fontSize: 13,
    color: colors.gray900,
    marginTop: 4,
    lineHeight: 19,
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
