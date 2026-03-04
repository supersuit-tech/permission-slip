/**
 * Approval detail screen — displays the full details of a single approval
 * request including agent info, action parameters, risk level, expiry
 * countdown, context, and timeline. Pending approvals show a deny button.
 */
import { useCallback, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useAgents, getAgentDisplayName } from "../../hooks/useAgents";
import { useDenyApproval } from "../../hooks/useDenyApproval";
import { colors } from "../../theme/colors";
import {
  humanizeActionType,
  buildActionSummary,
  secondsUntil,
  safeParams,
  isExpired as checkExpired,
  formatParamValue,
  formatTimestamp,
} from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalDetail">;

export default function ApprovalDetailScreen({ route, navigation }: Props) {
  const { approval } = route.params;
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const [denied, setDenied] = useState(false);

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

  const remaining = secondsUntil(approval.expires_at);
  const isPending = approval.status === "pending";
  const expired = checkExpired(approval.status, approval.expires_at);

  const summary = buildActionSummary(approval.action.type, parameters);

  /** Whether the deny action button should be visible. */
  const showActions = isPending && !expired && !denied;

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
              setDenied(true);
            } catch {
              Alert.alert("Error", "Failed to deny request. Please try again.");
            }
          },
        },
      ],
    );
  }, [denyApproval, approval.approval_id]);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={{ paddingBottom: insets.bottom + 24 }}
    >
      {/* Status banner for resolved approvals */}
      {(approval.status === "approved" && !denied) && (
        <View
          style={styles.statusBannerApproved}
          accessibilityRole="alert"
        >
          <Text style={styles.statusBannerTextApproved}>Approved</Text>
        </View>
      )}
      {(approval.status === "denied" || denied) && (
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
      {expired && !denied && (
        <View style={styles.statusBannerCancelled}>
          <Text style={styles.statusBannerText}>Expired</Text>
        </View>
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
      {isPending && !denied && (
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

      {/* Deny Action */}
      {showActions && (
        <View style={styles.section}>
          <Pressable
            testID="deny-button"
            style={({ pressed }) => [
              styles.denyButton,
              pressed && styles.denyButtonPressed,
              isDenying && styles.denyButtonDisabled,
            ]}
            onPress={handleDeny}
            disabled={isDenying}
            accessibilityRole="button"
            accessibilityLabel="Deny request"
          >
            {isDenying ? (
              <ActivityIndicator
                testID="deny-loading"
                color={colors.error}
                size="small"
              />
            ) : (
              <Text style={styles.denyButtonText}>Deny</Text>
            )}
          </Pressable>
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
          {approval.approved_at && (
            <View style={styles.paramRow}>
              <Text style={styles.paramKey}>Approved</Text>
              <Text style={styles.paramValue}>
                {formatTimestamp(approval.approved_at)}
              </Text>
            </View>
          )}
          {approval.denied_at && (
            <View style={styles.paramRow}>
              <Text style={styles.paramKey}>Denied</Text>
              <Text style={styles.paramValue}>
                {formatTimestamp(approval.denied_at)}
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
  );
}

/** Human-readable explanations shown inline for each risk level. */
const RISK_DESCRIPTIONS: Record<string, string> = {
  low: "Minimal impact, easily reversible",
  medium: "Moderate impact, some consequences",
  high: "Significant impact, hard to reverse",
};


const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.gray50,
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
  denyButton: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.error,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  denyButtonPressed: {
    backgroundColor: colors.riskHighBg,
  },
  denyButtonDisabled: {
    opacity: 0.6,
  },
  denyButtonText: {
    fontSize: 16,
    fontWeight: "600",
    color: colors.error,
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
});
