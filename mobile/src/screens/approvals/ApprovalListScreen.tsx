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
import { useAgents, getAgentDisplayName } from "../../hooks/useAgents";
import { colors } from "../../theme/colors";
import { buildActionSummary, humanizeActionType, safeParams, isExpired as checkExpired, formatRelativeTime, formatLastUpdated } from "./approvalUtils";
import { RiskBadge } from "./RiskBadge";
import { CountdownBadge } from "./CountdownBadge";
import { useAuth } from "../../auth/AuthContext";

type StatusTab = "pending" | "approved" | "denied";

const TABS: { key: StatusTab; label: string }[] = [
  { key: "pending", label: "Pending" },
  { key: "approved", label: "Approved" },
  { key: "denied", label: "Denied" },
];

type Props = NativeStackScreenProps<RootStackParamList, "ApprovalList">;

export default function ApprovalListScreen({ navigation }: Props) {
  const [activeTab, setActiveTab] = useState<StatusTab>("pending");
  const { approvals, isLoading, isRefetching, error, refetch, dataUpdatedAt } =
    useApprovals(activeTab);
  const { agents } = useAgents();
  const { signOut } = useAuth();
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

  const handleSignOut = useCallback(() => {
    Alert.alert("Sign out", "Are you sure you want to sign out?", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign Out", style: "destructive", onPress: () => signOut() },
    ]);
  }, [signOut]);

  const renderItem = useCallback(
    ({ item }: { item: ApprovalSummary }) => (
      <ApprovalRow
        approval={item}
        agentName={resolveAgentName(item.agent_id)}
        onPress={() => handlePress(item)}
      />
    ),
    [resolveAgentName, handlePress],
  );

  const keyExtractor = useCallback(
    (item: ApprovalSummary) => item.approval_id,
    [],
  );

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      <View style={styles.header}>
        <Text style={styles.title}>Approvals</Text>
        <TouchableOpacity
          testID="sign-out"
          accessibilityLabel="Sign out"
          accessibilityRole="button"
          onPress={handleSignOut}
          style={styles.signOutButton}
        >
          <Text style={styles.signOutText}>Sign Out</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.tabBar}>
        {TABS.map((tab) => {
          const isActive = activeTab === tab.key;
          const count =
            isActive && approvals.length > 0 ? approvals.length : null;
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
            color={colors.gray900}
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
          data={approvals}
          renderItem={renderItem}
          keyExtractor={keyExtractor}
          contentContainerStyle={
            approvals.length === 0 ? styles.emptyContainer : styles.list
          }
          ListEmptyComponent={<EmptyState tab={activeTab} />}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={() => refetch()}
              tintColor={colors.gray500}
            />
          }
        />
      )}
    </View>
  );
}

/** A single row in the approval list showing action type, agent, risk, and countdown. */
const ApprovalRow = memo(function ApprovalRow({
  approval,
  agentName,
  onPress,
}: {
  approval: ApprovalSummary;
  agentName: string;
  onPress: () => void;
}) {
  const summary = buildActionSummary(
    approval.action.type,
    safeParams(approval.action.parameters),
  );
  const expired = checkExpired(approval.status, approval.expires_at);

  return (
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
  );
});

/** Tab-specific empty state shown when there are no approvals for the selected status. */
function EmptyState({ tab }: { tab: StatusTab }) {
  const messages: Record<StatusTab, { title: string; body: string }> = {
    pending: {
      title: "No pending requests",
      body: "New approval requests from your agents will appear here.",
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
  title: {
    fontSize: 28,
    fontWeight: "700",
    color: colors.gray900,
  },
  signOutButton: {
    paddingVertical: 6,
    paddingHorizontal: 12,
  },
  signOutText: {
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
    borderBottomColor: colors.gray900,
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
    backgroundColor: colors.gray900,
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
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: 1,
    borderBottomColor: colors.gray100,
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
});
