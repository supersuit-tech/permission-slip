import { useCallback, useState } from "react";
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
import { useAuth } from "../../auth/AuthContext";
import { useApprovals, type ApprovalStatus, type ApprovalSummary } from "../../hooks/useApprovals";
import { colors } from "../../theme/colors";
import ApprovalListItem from "./ApprovalListItem";
import StatusFilterTabs from "./StatusFilterTabs";

export default function ApprovalListScreen() {
  const { signOut, user } = useAuth();
  const insets = useSafeAreaInsets();
  const [status, setStatus] = useState<ApprovalStatus>("pending");
  const { approvals, isLoading, isRefetching, error, refetch } = useApprovals(status);

  const handleSignOut = useCallback(() => {
    Alert.alert("Sign out", "Are you sure you want to sign out?", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign Out", style: "destructive", onPress: () => signOut() },
    ]);
  }, [signOut]);

  const renderItem = useCallback(
    ({ item }: { item: ApprovalSummary }) => (
      <ApprovalListItem
        approval={item}
        onPress={() => {
          // Detail screen navigation will be added in the next phase
        }}
      />
    ),
    [],
  );

  const keyExtractor = useCallback(
    (item: ApprovalSummary) => item.approval_id,
    [],
  );

  return (
    <View style={[styles.container, { paddingTop: insets.top }]}>
      <View style={styles.header}>
        <View>
          <Text style={styles.title}>Approvals</Text>
          <Text style={styles.subtitle}>{user?.email ?? ""}</Text>
        </View>
        <TouchableOpacity
          testID="sign-out"
          accessibilityLabel="Sign out"
          accessibilityRole="button"
          style={styles.signOutButton}
          onPress={handleSignOut}
        >
          <Text style={styles.signOutText}>Sign Out</Text>
        </TouchableOpacity>
      </View>

      <StatusFilterTabs selected={status} onSelect={setStatus} />

      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator size="large" color={colors.gray900} />
        </View>
      ) : error ? (
        <View style={styles.centered}>
          <Text style={styles.errorText}>{error}</Text>
          <TouchableOpacity
            testID="retry-button"
            accessibilityLabel="Retry loading approvals"
            accessibilityRole="button"
            style={styles.retryButton}
            onPress={() => refetch()}
          >
            <Text style={styles.retryText}>Retry</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <FlatList
          testID="approval-list"
          data={approvals}
          renderItem={renderItem}
          keyExtractor={keyExtractor}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor={colors.gray500}
            />
          }
          contentContainerStyle={
            approvals.length === 0 ? styles.emptyContainer : styles.listContent
          }
          ListEmptyComponent={
            <View style={styles.centered}>
              <Text style={styles.emptyTitle}>
                {status === "pending" ? "All clear" : `No ${status} approvals`}
              </Text>
              <Text style={styles.emptyBody}>
                {status === "pending"
                  ? "You have no pending approval requests."
                  : `There are no ${status} approvals to show.`}
              </Text>
            </View>
          }
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.gray50,
  },
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 4,
    backgroundColor: colors.white,
  },
  title: {
    fontSize: 24,
    fontWeight: "700",
    color: colors.gray900,
  },
  subtitle: {
    fontSize: 13,
    color: colors.gray400,
    marginTop: 2,
  },
  signOutButton: {
    borderWidth: 1,
    borderColor: colors.gray300,
    borderRadius: 8,
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  signOutText: {
    color: colors.gray700,
    fontSize: 13,
    fontWeight: "500",
  },
  listContent: {
    paddingTop: 8,
    paddingBottom: 24,
  },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 32,
  },
  emptyContainer: {
    flexGrow: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 32,
  },
  emptyTitle: {
    fontSize: 17,
    fontWeight: "600",
    color: colors.gray700,
    marginBottom: 6,
  },
  emptyBody: {
    fontSize: 14,
    color: colors.gray400,
    textAlign: "center",
    lineHeight: 20,
  },
  errorText: {
    fontSize: 15,
    color: colors.error,
    textAlign: "center",
    marginBottom: 16,
  },
  retryButton: {
    backgroundColor: colors.gray900,
    borderRadius: 8,
    paddingVertical: 10,
    paddingHorizontal: 24,
  },
  retryText: {
    color: colors.white,
    fontSize: 15,
    fontWeight: "600",
  },
});
