import { useMemo } from "react";
import {
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useAgents, getAgentDisplayName } from "../../hooks/useAgents";
import { colors } from "../../theme/colors";
import {
  humanizeActionType,
  buildActionSummary,
  secondsUntil,
} from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalDetail">;

export default function ApprovalDetailScreen({ route }: Props) {
  const { approval } = route.params;
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();

  const agent = useMemo(
    () => agents.find((a) => a.agent_id === approval.agent_id),
    [agents, approval.agent_id],
  );
  const agentName = agent
    ? getAgentDisplayName(agent)
    : `Agent ${approval.agent_id}`;

  const parameters = approval.action.parameters as Record<string, unknown>;
  const paramEntries = Object.entries(parameters);
  const contextDetails = approval.context.details as
    | Record<string, unknown>
    | undefined;
  const contextDetailEntries = contextDetails
    ? Object.entries(contextDetails)
    : [];

  const remaining = secondsUntil(approval.expires_at);
  const isPending = approval.status === "pending";
  const isExpired = isPending && remaining <= 0;

  const summary = buildActionSummary(approval.action.type, parameters);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={{ paddingBottom: insets.bottom + 24 }}
    >
      {/* Status banner for resolved approvals */}
      {approval.status === "approved" && (
        <View style={styles.statusBannerApproved}>
          <Text style={styles.statusBannerText}>Approved</Text>
        </View>
      )}
      {approval.status === "denied" && (
        <View style={styles.statusBannerDenied}>
          <Text style={styles.statusBannerText}>Denied</Text>
        </View>
      )}
      {approval.status === "cancelled" && (
        <View style={styles.statusBannerCancelled}>
          <Text style={styles.statusBannerText}>Cancelled</Text>
        </View>
      )}
      {isExpired && (
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
        </View>
      </View>

      {/* Context description */}
      {approval.context.description && (
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Context</Text>
          <View style={styles.card}>
            <Text style={styles.contextDescription}>
              {approval.context.description}
            </Text>
          </View>
        </View>
      )}

      {/* Risk Level detail */}
      {approval.context.risk_level && (
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Risk Level</Text>
          <View style={styles.card}>
            <View style={styles.riskRow}>
              <RiskBadge level={approval.context.risk_level} />
              <Text style={styles.riskDescription}>
                {RISK_DESCRIPTIONS[approval.context.risk_level]}
              </Text>
            </View>
            {approval.context.risk_level === "high" && (
              <View style={styles.highRiskWarning}>
                <Text style={styles.highRiskWarningText}>
                  This is a high-risk action. Review the details carefully.
                </Text>
              </View>
            )}
          </View>
        </View>
      )}

      {/* Expiry Countdown */}
      {isPending && (
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Expiry</Text>
          <View style={styles.card}>
            <View style={styles.expiryRow}>
              <CountdownBadge expiresAt={approval.expires_at} />
              <Text style={styles.expiryLabel}>
                {isExpired ? "This request has expired" : "remaining"}
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
            {paramEntries.map(([key, value]) => (
              <View key={key} style={styles.paramRow}>
                <Text style={styles.paramKey}>{key}</Text>
                <Text style={styles.paramValue} selectable>
                  {formatParamValue(value)}
                </Text>
              </View>
            ))}
          </View>
        </View>
      )}

      {/* Context Details */}
      {contextDetailEntries.length > 0 && (
        <View style={styles.section}>
          <Text style={styles.sectionLabel}>Additional Context</Text>
          <View style={styles.card}>
            {contextDetailEntries.map(([key, value]) => (
              <View key={key} style={styles.paramRow}>
                <Text style={styles.paramKey}>{key}</Text>
                <Text style={styles.paramValue} selectable>
                  {formatParamValue(value)}
                </Text>
              </View>
            ))}
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

const RISK_DESCRIPTIONS: Record<string, string> = {
  low: "Minimal impact, easily reversible",
  medium: "Moderate impact, some consequences",
  high: "Significant impact, hard to reverse",
};

function formatParamValue(value: unknown): string {
  if (value == null) return "null";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) {
    return value.map((v) => formatParamValue(v)).join(", ");
  }
  return JSON.stringify(value, null, 2);
}

function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

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
  contextDescription: {
    fontSize: 15,
    color: colors.gray700,
    lineHeight: 22,
  },
  riskRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  riskDescription: {
    fontSize: 14,
    color: colors.gray500,
    flex: 1,
  },
  highRiskWarning: {
    marginTop: 12,
    backgroundColor: colors.riskHighBg,
    borderRadius: 8,
    padding: 12,
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
  footerLabel: {
    fontSize: 11,
    color: colors.gray400,
    textAlign: "center",
    fontFamily: "monospace",
  },
});
