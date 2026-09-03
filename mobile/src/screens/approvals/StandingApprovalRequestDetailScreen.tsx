import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useAgents, getAgentDisplayName, type AgentSummary } from "../../hooks/useAgents";
import { useApproveStandingApprovalRequest } from "../../hooks/useApproveStandingApprovalRequest";
import { useDenyStandingApprovalRequest } from "../../hooks/useDenyStandingApprovalRequest";
import { useStandingApprovalConnectorLabel } from "../../hooks/useStandingApprovalConnectorLabel";
import { useStandingApprovalInstanceScope } from "../../hooks/useStandingApprovalInstanceScope";
import { colors } from "../../theme/colors";
import { humanizeActionType } from "./approvalUtils";
import { formatStandingApprovalConstraintsText } from "./formatStandingApprovalConstraints";
import { StandingApprovalInstanceScopeLine } from "./StandingApprovalInstanceScopeLine";
import { constraintsAreUnrestricted } from "./unrestrictedConstraints";

type Props = NativeStackScreenProps<
  RootStackParamList,
  "StandingApprovalRequestDetail"
>;

export default function StandingApprovalRequestDetailScreen({
  route,
  navigation,
}: Props) {
  const { request } = route.params;
  const { agents } = useAgents();
  const approve = useApproveStandingApprovalRequest();
  const deny = useDenyStandingApprovalRequest();
  const [busy, setBusy] = useState(false);

  const agent = agents.find((a: AgentSummary) => a.agent_id === request.agent_id);
  const agentName = agent ? getAgentDisplayName(agent) : `Agent ${request.agent_id}`;
  const { connectorLabel } = useStandingApprovalConnectorLabel(request);
  const { scopeLabel } = useStandingApprovalInstanceScope(request);

  const constraintsObject =
    request.constraints && typeof request.constraints === "object"
      ? (request.constraints as Record<string, unknown>)
      : null;
  const unrestricted = constraintsAreUnrestricted(constraintsObject);
  const constraintsText = unrestricted
    ? "Unrestricted — any parameters for this action"
    : constraintsObject
      ? formatStandingApprovalConstraintsText(constraintsObject)
      : "No constraints";

  const runApprove = useCallback(async () => {
    setBusy(true);
    try {
      await approve.mutateAsync(request.request_id);
      navigation.goBack();
    } catch (e) {
      Alert.alert("Error", e instanceof Error ? e.message : "Approve failed");
    } finally {
      setBusy(false);
    }
  }, [approve, navigation, request.request_id]);

  const handleApprove = useCallback(async () => {
    if (unrestricted) {
      Alert.alert(
        "Unrestricted rule",
        "This standing approval authorizes any parameters for this action. No approval prompts will be sent.",
        [
          { text: "Cancel", style: "cancel" },
          { text: "Approve", onPress: () => { void runApprove(); } },
        ],
      );
      return;
    }
    await runApprove();
  }, [runApprove, unrestricted]);

  const handleDeny = useCallback(async () => {
    setBusy(true);
    try {
      await deny.mutateAsync(request.request_id);
      navigation.goBack();
    } catch (e) {
      Alert.alert("Error", e instanceof Error ? e.message : "Deny failed");
    } finally {
      setBusy(false);
    }
  }, [deny, navigation, request.request_id]);

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <View style={styles.badge}>
        <Text style={styles.badgeText}>Rule proposal</Text>
      </View>
      <Text style={styles.title}>{humanizeActionType(request.action_type)}</Text>
      <Text style={styles.subtitle}>
        {connectorLabel} · From {agentName}
      </Text>

      {scopeLabel && <StandingApprovalInstanceScopeLine label={scopeLabel} />}

      <Text style={styles.sectionLabel}>Constraints</Text>
      <Text style={styles.mono}>{constraintsText}</Text>
      <Text style={styles.metaNote}>
        Verified fields ($meta) match server-fetched envelope data, not
        agent-supplied parameters.
      </Text>

      <View style={styles.actions}>
        <TouchableOpacity
          style={[styles.button, styles.denyButton]}
          onPress={handleDeny}
          disabled={busy}
        >
          {busy ? (
            <ActivityIndicator color={colors.gray700} />
          ) : (
            <Text style={styles.denyText}>Deny</Text>
          )}
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.button, styles.approveButton]}
          onPress={handleApprove}
          disabled={busy}
        >
          {busy ? (
            <ActivityIndicator color={colors.white} />
          ) : (
            <Text style={styles.approveText}>Approve rule</Text>
          )}
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { padding: 16, paddingBottom: 40, backgroundColor: colors.white },
  badge: {
    alignSelf: "flex-start",
    backgroundColor: colors.gray100,
    borderRadius: 6,
    paddingHorizontal: 8,
    paddingVertical: 4,
    marginBottom: 12,
  },
  badgeText: { fontSize: 12, fontWeight: "600", color: colors.gray700 },
  title: { fontSize: 22, fontWeight: "700", color: colors.gray900 },
  subtitle: { fontSize: 14, color: colors.gray500, marginTop: 4, marginBottom: 16 },
  sectionLabel: {
    fontSize: 12,
    fontWeight: "600",
    textTransform: "uppercase",
    color: colors.gray500,
    marginBottom: 8,
  },
  mono: {
    fontFamily: "monospace",
    fontSize: 12,
    backgroundColor: colors.gray100,
    padding: 12,
    borderRadius: 8,
    color: colors.gray900,
  },
  metaNote: { marginTop: 8, fontSize: 12, color: colors.gray500, lineHeight: 18 },
  actions: { flexDirection: "row", gap: 12, marginTop: 24 },
  button: {
    flex: 1,
    paddingVertical: 14,
    borderRadius: 8,
    alignItems: "center",
  },
  denyButton: { borderWidth: 1, borderColor: colors.gray300 },
  approveButton: { backgroundColor: colors.primary },
  denyText: { color: colors.gray900, fontWeight: "600" },
  approveText: { color: colors.white, fontWeight: "600" },
});
