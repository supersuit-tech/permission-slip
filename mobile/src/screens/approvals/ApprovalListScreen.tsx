/**
 * Approval list screen — the primary screen after login. Displays a
 * tabbed list of approval requests (Pending / Approved / Denied) with
 * pull-to-refresh, loading/error/empty states, and navigation to the
 * detail screen.
 */
import { memo, useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  FlatList,
  RefreshControl,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useIsFocused } from "@react-navigation/native";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useApprovals, type ApprovalSummary } from "../../hooks/useApprovals";
import { useDenyApproval } from "../../hooks/useDenyApproval";
import { useDenyAllApprovals } from "../../hooks/useDenyAllApprovals";
import {
  useStandingApprovalRequests,
  type StandingApprovalRequestSummary,
} from "../../hooks/useStandingApprovalRequests";
import { useDenyStandingApprovalRequest } from "../../hooks/useDenyStandingApprovalRequest";
import { useStandingApprovalInstanceScope } from "../../hooks/useStandingApprovalInstanceScope";
import { useAgents, getAgentDisplayName } from "../../hooks/useAgents";
import { useActionSchema } from "../../hooks/useActionSchema";
import { useStandingApprovalConnectorLabel } from "../../hooks/useStandingApprovalConnectorLabel";
import { colors } from "../../theme/colors";
import { buildActionSummary, humanizeActionType, safeParams, isExpired as checkExpired, formatRelativeTime, formatLastUpdated } from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";
import { BrandBadge } from "../../components/BrandBadge";
import { StandingApprovalInstanceScopeLine } from "./StandingApprovalInstanceScopeLine";
import { DeclineRequestButton } from "./DeclineRequestButton";

type StatusTab = "pending" | "approved" | "denied";

const TABS: { key: StatusTab; label: string }[] = [
  { key: "pending", label: "Pending" },
  { key: "approved", label: "Approved" },
  { key: "denied", label: "Denied" },
];

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalList">;

export default function ApprovalListScreen({ navigation }: Props) {
  const [activeTab, setActiveTab] = useState<StatusTab>("pending");
  const [dismissedApprovalIds, setDismissedApprovalIds] = useState<Set<string>>(
    () => new Set(),
  );
  const [dismissedRuleIds, setDismissedRuleIds] = useState<Set<string>>(
    () => new Set(),
  );
  const { approvals, isLoading, isRefetching, error, refetch, dataUpdatedAt } =
    useApprovals(activeTab);
  const { denyApproval } = useDenyApproval();
  const { denyAllApprovals, isPending: isDenyingAll } = useDenyAllApprovals(activeTab);
  const {
    requests: ruleProposals,
    refetch: refetchRules,
    isRefetching: rulesRefetching,
  } = useStandingApprovalRequests();
  const { mutateAsync: denyRuleRequest } = useDenyStandingApprovalRequest();
  const { agents } = useAgents();
  const insets = useSafeAreaInsets();

  // Re-render the "Updated X ago" label every 15 seconds so it stays current.
  // Only tick when the screen is focused to avoid background re-renders when
  // the user is on the detail screen.
  const isFocused = useIsFocused();
  const [, setTick] = useState(0);
  useEffect(() => {
    if (!isFocused || dataUpdatedAt === 0) return;
    const id = setInterval(() => setTick((t) => t + 1), 15_000);
    return () => clearInterval(id);
  }, [isFocused, dataUpdatedAt]);
  const lastUpdatedText = formatLastUpdated(dataUpdatedAt);

  const agentMap = useMemo(() => {
    const map = new Map<number, { agent_id: number; metadata?: unknown }>();
    for (const agent of agents) {
      map.set(agent.agent_id, agent);
    }
    return map;
  }, [agents]);

  const resolveAgentName = useCallback(
    (agentId: number) => {
      const agent = agentMap.get(agentId);
      if (agent) return getAgentDisplayName(agent);
      return `Agent ${agentId}`;
    },
    [agentMap],
  );

  const handlePress = useCallback(
    (approval: ApprovalSummary) => {
      navigation.navigate("ApprovalDetail", {
        approvalId: approval.approval_id,
        approval,
      });
    },
    [navigation],
  );

  const handleBulkPress = useCallback(
    (bulkGroupId: string) => {
      navigation.navigate("BulkApprovalGroup", { bulkGroupId });
    },
    [navigation],
  );

  const { standaloneApprovals, bulkGroups } = useMemo(() => {
    const groups = new Map<
      string,
      { actionType: string; itemCount: number; agentId: number }
    >();
    const standalone: ApprovalSummary[] = [];
    for (const approval of approvals) {
      const gid = approval.bulk_group_id;
      if (gid) {
        const existing = groups.get(gid);
        if (!existing) {
          groups.set(gid, {
            actionType: approval.action.type,
            itemCount: 1,
            agentId: approval.agent_id,
          });
        } else {
          existing.itemCount += 1;
        }
      } else {
        standalone.push(approval);
      }
    }
    return {
      standaloneApprovals: standalone,
      bulkGroups: [...groups.entries()].map(([id, meta]) => ({ id, ...meta })),
    };
  }, [approvals]);

  const visibleStandaloneApprovals = useMemo(
    () =>
      standaloneApprovals.filter(
        (approval) => !dismissedApprovalIds.has(approval.approval_id),
      ),
    [standaloneApprovals, dismissedApprovalIds],
  );

  const visibleRuleProposals = useMemo(
    () =>
      ruleProposals.filter(
        (request: StandingApprovalRequestSummary) =>
          !dismissedRuleIds.has(request.request_id),
      ),
    [ruleProposals, dismissedRuleIds],
  );

  const handleRulePress = useCallback(
    (request: StandingApprovalRequestSummary) => {
      navigation.navigate("StandingApprovalRequestDetail", {
        requestId: request.request_id,
        request,
      });
    },
    [navigation],
  );

  const handleOpenSettings = useCallback(() => {
    navigation.navigate("Settings");
  }, [navigation]);

  const handleDeclineApproval = useCallback(
    async (approvalId: string) => {
      setDismissedApprovalIds((prev) => new Set(prev).add(approvalId));
      try {
        await denyApproval(approvalId);
      } catch (err) {
        setDismissedApprovalIds((prev) => {
          const next = new Set(prev);
          next.delete(approvalId);
          return next;
        });
        throw err;
      }
    },
    [denyApproval],
  );

  const handleDeclineRule = useCallback(
    async (requestId: string) => {
      setDismissedRuleIds((prev) => new Set(prev).add(requestId));
      try {
        await denyRuleRequest(requestId);
      } catch (err) {
        setDismissedRuleIds((prev) => {
          const next = new Set(prev);
          next.delete(requestId);
          return next;
        });
        throw err;
      }
    },
    [denyRuleRequest],
  );

  const handleDeclineAllPress = useCallback(() => {
    const count = visibleStandaloneApprovals.length;
    Alert.alert(
      "Decline all pending requests?",
      `Decline all ${count} pending request${count !== 1 ? "s" : ""}? Agents will be notified that these requests were denied.`,
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Decline all",
          style: "destructive",
          onPress: () => {
            void denyAllApprovals().catch(() => {
              Alert.alert("Error", "Failed to decline pending requests. Please try again.");
            });
          },
        },
      ],
    );
  }, [denyAllApprovals, visibleStandaloneApprovals.length]);

  const renderItem = useCallback(
    ({ item }: { item: ApprovalSummary }) => (
      <ApprovalRow
        approval={item}
        agentName={resolveAgentName(item.agent_id)}
        onPress={() => handlePress(item)}
        onDecline={() => handleDeclineApproval(item.approval_id)}
      />
    ),
    [resolveAgentName, handlePress, handleDeclineApproval],
  );

  const keyExtractor = useCallback(
    (item: ApprovalSummary) => item.approval_id,
    [],
  );

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <BrandBadge size={28} />
          <Text style={styles.title}>Permission Slip</Text>
        </View>
        <TouchableOpacity
          testID="settings-button"
          accessibilityLabel="Settings"
          accessibilityRole="button"
          onPress={handleOpenSettings}
          style={styles.settingsButton}
        >
          <Text style={styles.settingsText}>Settings</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.tabBar}>
        {TABS.map((tab) => {
          const isActive = activeTab === tab.key;
          const count =
            isActive && (visibleStandaloneApprovals.length + bulkGroups.length) > 0
              ? visibleStandaloneApprovals.length + bulkGroups.length
              : null;
          return (
            <TouchableOpacity
              key={tab.key}
              testID={`tab-${tab.key}`}
              accessibilityRole="tab"
              accessibilityState={{ selected: isActive }}
              accessibilityLabel={
                count
                  ? `${tab.label}, ${count} item${count !== 1 ? "s" : ""}`
                  : tab.label
              }
              style={[styles.tab, isActive && styles.tabActive]}
              onPress={() => setActiveTab(tab.key)}
            >
              <View style={styles.tabContent}>
                <Text
                  style={[styles.tabText, isActive && styles.tabTextActive]}
                >
                  {tab.label}
                </Text>
                {count != null && (
                  <View style={styles.tabBadge}>
                    <Text style={styles.tabBadgeText}>{count}</Text>
                  </View>
                )}
              </View>
            </TouchableOpacity>
          );
        })}
      </View>

      {lastUpdatedText != null && !isLoading && (
        <View style={styles.lastUpdatedBar}>
          <Text style={styles.lastUpdatedText} testID="last-updated">
            {lastUpdatedText}
          </Text>
        </View>
      )}

      {isLoading && !isRefetching ? (
        <View style={styles.center}>
          <ActivityIndicator
            size="large"
            color={colors.primary}
            testID="loading-indicator"
          />
        </View>
      ) : error ? (
        <View style={styles.center}>
          <Text style={styles.errorText}>{error}</Text>
          <TouchableOpacity
            style={styles.retryButton}
            onPress={() => refetch()}
          >
            <Text style={styles.retryText}>Retry</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <FlatList
          data={visibleStandaloneApprovals}
          renderItem={renderItem}
          keyExtractor={keyExtractor}
          contentContainerStyle={
            visibleStandaloneApprovals.length === 0 &&
            bulkGroups.length === 0 &&
            (activeTab !== "pending" || visibleRuleProposals.length === 0)
              ? styles.emptyContainer
              : styles.list
          }
          ListHeaderComponent={
            activeTab === "pending" &&
            (visibleRuleProposals.length > 0 ||
              bulkGroups.length > 0 ||
              visibleStandaloneApprovals.length > 0) ? (
              <View style={styles.ruleSection}>
                {visibleStandaloneApprovals.length > 0 && (
                  <TouchableOpacity
                    testID="decline-all-button"
                    accessibilityRole="button"
                    accessibilityLabel={`Decline all ${visibleStandaloneApprovals.length} pending requests`}
                    style={[
                      styles.declineAllButton,
                      isDenyingAll && styles.declineAllButtonDisabled,
                    ]}
                    disabled={isDenyingAll}
                    onPress={handleDeclineAllPress}
                  >
                    <Text style={styles.declineAllText}>
                      Decline all ({visibleStandaloneApprovals.length})
                    </Text>
                  </TouchableOpacity>
                )}
                {bulkGroups.map((group) => (
                  <TouchableOpacity
                    key={group.id}
                    style={styles.bulkRow}
                    onPress={() => handleBulkPress(group.id)}
                  >
                    <Text style={styles.bulkTitle}>
                      {humanizeActionType(group.actionType)} ({group.itemCount}{" "}
                      items)
                    </Text>
                    <Text style={styles.bulkAgent}>
                      {resolveAgentName(group.agentId)}
                    </Text>
                  </TouchableOpacity>
                ))}
                {visibleRuleProposals.map((request: StandingApprovalRequestSummary) => (
                  <RuleProposalRow
                    key={request.request_id}
                    request={request}
                    agentName={resolveAgentName(request.agent_id)}
                    onPress={() => handleRulePress(request)}
                    onDecline={() => handleDeclineRule(request.request_id)}
                  />
                ))}
              </View>
            ) : null
          }
          ListEmptyComponent={
            activeTab === "pending" && visibleRuleProposals.length > 0 ? null : (
              <EmptyState tab={activeTab} />
            )
          }
          refreshControl={
            <RefreshControl
              refreshing={isRefetching || rulesRefetching}
              onRefresh={() => {
                refetch();
                if (activeTab === "pending") refetchRules();
              }}
              tintColor={colors.gray500}
            />
          }
        />
      )}
    </View>
  );
}

const RuleProposalRow = memo(function RuleProposalRow({
  request,
  agentName,
  onPress,
  onDecline,
}: {
  request: StandingApprovalRequestSummary;
  agentName: string;
  onPress: () => void;
  onDecline: () => Promise<void>;
}) {
  const { connectorLabel } = useStandingApprovalConnectorLabel(request);
  const { scopeLabel } = useStandingApprovalInstanceScope(request);

  return (
    <View style={styles.ruleRowContainer}>
      <TouchableOpacity style={styles.ruleRow} onPress={onPress} accessibilityRole="button">
        <View style={styles.ruleBadge}>
          <Text style={styles.ruleBadgeText}>Rule proposal</Text>
        </View>
        <Text style={styles.rowTitle}>{humanizeActionType(request.action_type)}</Text>
        <Text style={styles.rowSubtitle}>
          {connectorLabel} · {agentName}
        </Text>
        {scopeLabel && (
          <StandingApprovalInstanceScopeLine label={scopeLabel} compact />
        )}
      </TouchableOpacity>
      <DeclineRequestButton
        testID={`decline-rule-${request.request_id}`}
        onDecline={onDecline}
      />
    </View>
  );
});

/** A single row in the approval list showing action type, agent, risk, and countdown. */
const ApprovalRow = memo(function ApprovalRow({
  approval,
  agentName,
  onPress,
  onDecline,
}: {
  approval: ApprovalSummary;
  agentName: string;
  onPress: () => void;
  onDecline: () => Promise<void>;
}) {
  const { displayTemplate } = useActionSchema(approval.action.type);
  const summary = buildActionSummary(
    approval.action.type,
    safeParams(approval.action.parameters),
    displayTemplate,
    approval.resource_details as Record<string, unknown> | undefined,
  );
  const expired = checkExpired(approval.status, approval.expires_at);

  return (
    <View style={styles.rowContainer}>
      <TouchableOpacity
        testID={`approval-row-${approval.approval_id}`}
        accessibilityLabel={`${humanizeActionType(approval.action.type)} from ${agentName}`}
        style={[styles.row, expired && styles.rowExpired]}
        onPress={onPress}
      >
        <View style={styles.rowContent}>
          <View style={styles.rowTop}>
            <Text style={styles.actionType} numberOfLines={1}>
              {humanizeActionType(approval.action.type)}
            </Text>
            <RiskBadge level={approval.context.risk_level} />
          </View>
          <Text style={styles.summary} numberOfLines={1}>
            {summary}
          </Text>
          <View style={styles.rowBottom}>
            <Text style={styles.agentName} numberOfLines={1}>
              {agentName}
            </Text>
            {approval.status === "pending" && (
              <>
                <Text style={styles.dot}>{"\u00B7"}</Text>
                <CountdownBadge expiresAt={approval.expires_at} />
              </>
            )}
            {approval.status === "approved" && (
              <>
                <Text style={styles.dot}>{"\u00B7"}</Text>
                <Text style={styles.statusApproved}>Approved</Text>
              </>
            )}
            {approval.status === "denied" && (
              <>
                <Text style={styles.dot}>{"\u00B7"}</Text>
                <Text style={styles.statusDenied}>Denied</Text>
              </>
            )}
            <Text style={styles.dot}>{"\u00B7"}</Text>
            <Text style={styles.relativeTime}>
              {formatRelativeTime(approval.created_at)}
            </Text>
          </View>
        </View>
        <Text style={styles.chevron}>{"\u203A"}</Text>
      </TouchableOpacity>
      {approval.status === "pending" && (
        <DeclineRequestButton
          testID={`decline-approval-${approval.approval_id}`}
          onDecline={onDecline}
        />
      )}
    </View>
  );
});

/** Tab-specific empty state shown when there are no approvals for the selected status. */
function EmptyState({ tab }: { tab: StatusTab }) {
  const messages: Record<StatusTab, { title: string; body: string }> = {
    pending: {
      title: "No pending requests",
      body: "New approval requests from Openclaw will appear here.",
    },
    approved: {
      title: "No approved requests",
      body: "Approved requests will appear here.",
    },
    denied: {
      title: "No denied requests",
      body: "Denied requests will appear here.",
    },
  };
  const msg = messages[tab];

  return (
    <View style={styles.empty}>
      <Text style={styles.emptyTitle}>{msg.title}</Text>
      <Text style={styles.emptyBody}>{msg.body}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.white,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 20,
    paddingVertical: 12,
  },
  headerLeft: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  title: {
    fontSize: 28,
    fontWeight: "700",
    color: colors.gray900,
  },
  settingsButton: {
    paddingVertical: 6,
    paddingHorizontal: 12,
  },
  settingsText: {
    color: colors.gray500,
    fontSize: 14,
    fontWeight: "500",
  },
  tabBar: {
    flexDirection: "row",
    paddingHorizontal: 20,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray200,
  },
  tab: {
    paddingVertical: 10,
    paddingHorizontal: 16,
    marginRight: 4,
    borderBottomWidth: 2,
    borderBottomColor: "transparent",
  },
  tabActive: {
    borderBottomColor: colors.secondary,
  },
  tabText: {
    fontSize: 14,
    fontWeight: "500",
    color: colors.gray400,
  },
  tabTextActive: {
    color: colors.gray900,
  },
  tabContent: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  tabBadge: {
    backgroundColor: colors.primary,
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6,
  },
  tabBadgeText: {
    color: colors.white,
    fontSize: 11,
    fontWeight: "700",
  },
  lastUpdatedBar: {
    paddingHorizontal: 20,
    paddingVertical: 6,
    backgroundColor: colors.gray50,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  lastUpdatedText: {
    fontSize: 11,
    color: colors.gray400,
    textAlign: "right",
  },
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 32,
  },
  errorText: {
    color: colors.error,
    fontSize: 14,
    textAlign: "center",
    marginBottom: 12,
  },
  retryButton: {
    borderWidth: 1,
    borderColor: colors.gray300,
    borderRadius: 8,
    paddingVertical: 10,
    paddingHorizontal: 20,
  },
  retryText: {
    color: colors.gray700,
    fontSize: 14,
    fontWeight: "500",
  },
  list: {
    paddingVertical: 4,
  },
  emptyContainer: {
    flexGrow: 1,
    justifyContent: "center",
  },
  rowContainer: {
    flexDirection: "row",
    alignItems: "center",
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
  },
  row: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  rowExpired: {
    opacity: 0.5,
  },
  rowContent: {
    flex: 1,
    marginRight: 8,
  },
  rowTop: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    marginBottom: 2,
  },
  actionType: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
    flexShrink: 1,
  },
  summary: {
    fontSize: 13,
    color: colors.gray500,
    marginBottom: 4,
  },
  rowBottom: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  agentName: {
    fontSize: 12,
    color: colors.gray400,
    flexShrink: 1,
  },
  dot: {
    fontSize: 12,
    color: colors.gray400,
  },
  statusApproved: {
    fontSize: 12,
    fontWeight: "500",
    color: colors.success,
  },
  statusDenied: {
    fontSize: 12,
    fontWeight: "500",
    color: colors.error,
  },
  relativeTime: {
    fontSize: 12,
    color: colors.gray400,
  },
  chevron: {
    fontSize: 22,
    color: colors.gray400,
  },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 40,
    paddingVertical: 60,
  },
  emptyTitle: {
    fontSize: 16,
    fontWeight: "600",
    color: colors.gray500,
    marginBottom: 8,
    textAlign: "center",
  },
  emptyBody: {
    fontSize: 14,
    color: colors.gray400,
    textAlign: "center",
    lineHeight: 20,
  },
  ruleSection: {
    borderBottomWidth: 1,
    borderBottomColor: colors.gray200,
  },
  declineAllButton: {
    alignSelf: "flex-end",
    marginHorizontal: 20,
    marginVertical: 10,
    paddingVertical: 8,
    paddingHorizontal: 12,
    borderWidth: 1,
    borderColor: colors.gray300,
    borderRadius: 8,
  },
  declineAllButtonDisabled: {
    opacity: 0.5,
  },
  declineAllText: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.error,
  },
  bulkRow: {
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
    backgroundColor: "#eff6ff",
  },
  bulkTitle: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
  },
  bulkAgent: {
    fontSize: 12,
    color: colors.gray500,
    marginTop: 4,
  },
  ruleRowContainer: {
    flexDirection: "row",
    alignItems: "center",
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
    backgroundColor: "#f5f3ff",
  },
  ruleRow: {
    flex: 1,
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  ruleBadge: {
    alignSelf: "flex-start",
    backgroundColor: colors.gray200,
    borderRadius: 4,
    paddingHorizontal: 6,
    paddingVertical: 2,
    marginBottom: 6,
  },
  ruleBadgeText: {
    fontSize: 11,
    fontWeight: "600",
    color: colors.gray700,
  },
  rowTitle: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
  },
  rowSubtitle: {
    fontSize: 12,
    color: colors.gray500,
    marginTop: 2,
  },
});
