/**
 * Settings screen — allows the user to manage notification preferences
 * and sign out. Currently shows a toggle for the mobile-push notification
 * channel; other channels are managed via the web app.
 */
import { useCallback, useMemo } from "react";
import {
  ActivityIndicator,
  Alert,
  Linking,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useAuth } from "../../auth/AuthContext";
import { useNotificationPreferences } from "../../hooks/useNotificationPreferences";
import { useUpdateNotificationPreferences } from "../../hooks/useUpdateNotificationPreferences";
import { useNotificationTypePreferences } from "../../hooks/useNotificationTypePreferences";
import { useUpdateNotificationTypePreferences } from "../../hooks/useUpdateNotificationTypePreferences";
import Constants from "expo-constants";
import { useDeleteAccount } from "../../hooks/useDeleteAccount";
import { colors } from "../../theme/colors";
import CustomServerSettings from "./CustomServerSettings";

const GIT_COMMIT_HASH: string =
  (Constants.expoConfig?.extra?.gitCommitHash as string) ?? "unknown";

const GIT_COMMIT_TIMESTAMP: string =
  (Constants.expoConfig?.extra?.gitCommitTimestamp as string) ?? "unknown";

function formatCommitTimestamp(iso: string): string {
  if (iso === "unknown") return "";
  const date = new Date(iso);
  if (isNaN(date.getTime())) return "";
  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

const PRIVACY_POLICY_URL = "https://app.permissionslip.dev/policy/privacy";
const TERMS_OF_SERVICE_URL = "https://app.permissionslip.dev/policy/terms";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

export default function SettingsScreen(_props: Props) {
  const insets = useSafeAreaInsets();
  const { signOut, user } = useAuth();
  const { preferences, isLoading, error, refetch } =
    useNotificationPreferences();
  const {
    preferences: typePreferences,
    isLoading: isLoadingTypePrefs,
    error: typePrefsError,
  } = useNotificationTypePreferences();
  const { updatePreferences, isUpdating } =
    useUpdateNotificationPreferences();
  const { updatePreferences: updateTypePreferences, isUpdating: isUpdatingType } =
    useUpdateNotificationTypePreferences();
  const { deleteAccount, isDeleting } = useDeleteAccount();

  const contentContainerStyle = useMemo(
    () => ({ paddingBottom: insets.bottom + 24 }),
    [insets.bottom],
  );

  const mobilePushPref = preferences.find((p) => p.channel === "mobile-push");
  const mobilePushEnabled = mobilePushPref?.enabled ?? true;

  const standingPref = typePreferences.find(
    (p) => p.notification_type === "standing_execution",
  );
  const standingExecutionEnabled = standingPref?.enabled ?? true;

  const handleToggleMobilePush = useCallback(async () => {
    try {
      await updatePreferences([
        { channel: "mobile-push", enabled: !mobilePushEnabled },
      ]);
    } catch {
      Alert.alert(
        "Error",
        "Failed to update notification preference. Please try again.",
      );
    }
  }, [mobilePushEnabled, updatePreferences]);

  const handleToggleStandingExecution = useCallback(async () => {
    try {
      await updateTypePreferences([
        {
          notification_type: "standing_execution",
          enabled: !standingExecutionEnabled,
        },
      ]);
    } catch {
      Alert.alert(
        "Error",
        "Failed to update notification preference. Please try again.",
      );
    }
  }, [standingExecutionEnabled, updateTypePreferences]);

  const handleSignOut = useCallback(() => {
    Alert.alert("Sign out", "Are you sure you want to sign out?", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign Out", style: "destructive", onPress: () => signOut() },
    ]);
  }, [signOut]);

  const handleDeleteAccount = useCallback(() => {
    Alert.alert(
      "Delete Account",
      "This will permanently delete your account and all associated data including agents, approvals, and credentials. This action cannot be undone.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Delete Account",
          style: "destructive",
          onPress: () => {
            deleteAccount().catch(() => {
              Alert.alert(
                "Error",
                "Failed to delete account. Please try again.",
              );
            });
          },
        },
      ],
    );
  }, [deleteAccount]);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={contentContainerStyle}
    >
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Notifications</Text>
        <Text style={styles.sectionDescription}>
          Control how you receive approval request notifications on this device.
        </Text>

        {isLoading || isLoadingTypePrefs ? (
          <View style={styles.loadingContainer}>
            <ActivityIndicator
              size="small"
              color={colors.gray500}
              testID="prefs-loading"
            />
          </View>
        ) : error || typePrefsError ? (
          <View style={styles.errorContainer}>
            <Text style={styles.errorText}>{error ?? typePrefsError}</Text>
            <TouchableOpacity
              style={styles.retryButton}
              accessibilityRole="button"
              accessibilityLabel="Retry loading notification preferences"
              onPress={() => refetch()}
            >
              <Text style={styles.retryText}>Retry</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <View style={styles.card}>
            <View style={styles.toggleRow}>
              <View style={styles.toggleLabel}>
                <Text style={styles.toggleTitle}>Push Notifications</Text>
                <Text style={styles.toggleDescription}>
                  Receive push notifications when actions need your approval.
                </Text>
              </View>
              <Switch
                testID="mobile-push-toggle"
                value={mobilePushEnabled}
                onValueChange={handleToggleMobilePush}
                disabled={isUpdating}
                trackColor={{
                  false: colors.gray300,
                  true: colors.primary,
                }}
                accessibilityLabel="Push Notifications"
                accessibilityRole="switch"
                accessibilityState={{ checked: mobilePushEnabled }}
              />
            </View>
            <Text style={styles.subsectionTitle}>Notify me about</Text>
            <View style={[styles.card, styles.cardSpaced]}>
              <View style={styles.toggleRow}>
                <View style={styles.toggleLabel}>
                  <Text style={styles.toggleTitle}>Auto-approval executions</Text>
                  <Text style={styles.toggleDescription}>
                    When an action runs automatically because it matched a
                    standing approval.
                  </Text>
                </View>
                <Switch
                  testID="standing-execution-toggle"
                  value={standingExecutionEnabled}
                  onValueChange={handleToggleStandingExecution}
                  disabled={isUpdating || isUpdatingType}
                  trackColor={{
                    false: colors.gray300,
                    true: colors.primary,
                  }}
                  accessibilityLabel="Auto-approval execution notifications"
                  accessibilityRole="switch"
                  accessibilityState={{ checked: standingExecutionEnabled }}
                />
              </View>
            </View>
          </View>
        )}
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Account</Text>
        {user?.email ? (
          <View style={styles.card}>
            <View style={styles.accountRow}>
              <Text style={styles.accountLabel}>Signed in as</Text>
              <Text
                style={styles.accountEmail}
                numberOfLines={1}
                ellipsizeMode="middle"
              >
                {user.email}
              </Text>
            </View>
          </View>
        ) : null}
        <TouchableOpacity
          testID="sign-out-button"
          style={[styles.actionButton, user?.email ? styles.actionButtonSpaced : null]}
          accessibilityRole="button"
          accessibilityLabel="Sign out of your account"
          onPress={handleSignOut}
        >
          <Text style={styles.destructiveActionText}>Sign Out</Text>
        </TouchableOpacity>
        <TouchableOpacity
          testID="delete-account-button"
          style={[styles.actionButton, styles.actionButtonSpaced]}
          accessibilityRole="button"
          accessibilityLabel="Permanently delete your account"
          onPress={handleDeleteAccount}
          disabled={isDeleting}
        >
          {isDeleting ? (
            <ActivityIndicator size="small" color={colors.error} />
          ) : (
            <Text style={styles.destructiveActionText}>Delete Account</Text>
          )}
        </TouchableOpacity>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Server</Text>
        <Text style={styles.sectionDescription}>
          Connect to a self-hosted Permission Slip instance.
        </Text>
        <CustomServerSettings />
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Legal</Text>
        <View style={styles.card}>
          <TouchableOpacity
            testID="privacy-policy-link"
            style={styles.linkRow}
            accessibilityRole="link"
            accessibilityLabel="Privacy Policy"
            onPress={() => {
              Linking.openURL(PRIVACY_POLICY_URL).catch(() => {
                Alert.alert("Error", "Could not open Privacy Policy.");
              });
            }}
          >
            <Text style={styles.linkText}>Privacy Policy</Text>
            <Text style={styles.linkChevron}>›</Text>
          </TouchableOpacity>
          <View style={styles.linkSeparator} />
          <TouchableOpacity
            testID="terms-link"
            style={styles.linkRow}
            accessibilityRole="link"
            accessibilityLabel="Terms of Service"
            onPress={() => {
              Linking.openURL(TERMS_OF_SERVICE_URL).catch(() => {
                Alert.alert("Error", "Could not open Terms of Service.");
              });
            }}
          >
            <Text style={styles.linkText}>Terms of Service</Text>
            <Text style={styles.linkChevron}>›</Text>
          </TouchableOpacity>
        </View>
      </View>

      <View style={styles.buildInfo}>
        <Text style={styles.buildInfoText} testID="git-commit-hash">
          Build {GIT_COMMIT_HASH.slice(0, 7)}
          {formatCommitTimestamp(GIT_COMMIT_TIMESTAMP)
            ? ` · ${formatCommitTimestamp(GIT_COMMIT_TIMESTAMP)}`
            : ""}
        </Text>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.primaryBg,
  },
  section: {
    paddingHorizontal: 20,
    paddingTop: 24,
  },
  sectionTitle: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.gray500,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginBottom: 4,
  },
  sectionDescription: {
    fontSize: 13,
    color: colors.gray400,
    marginBottom: 12,
  },
  loadingContainer: {
    paddingVertical: 24,
    alignItems: "center",
  },
  errorContainer: {
    paddingVertical: 16,
    alignItems: "center",
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
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  retryText: {
    color: colors.gray700,
    fontSize: 14,
    fontWeight: "500",
  },
  card: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
  },
  cardSpaced: {
    marginTop: 12,
  },
  subsectionTitle: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.gray500,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    marginTop: 16,
    marginBottom: 8,
  },
  toggleRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  toggleLabel: {
    flex: 1,
    marginRight: 12,
  },
  toggleTitle: {
    fontSize: 15,
    fontWeight: "600",
    color: colors.gray900,
    marginBottom: 2,
  },
  toggleDescription: {
    fontSize: 13,
    color: colors.gray500,
    lineHeight: 18,
  },
  accountRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  accountLabel: {
    fontSize: 15,
    color: colors.gray500,
  },
  accountEmail: {
    fontSize: 15,
    fontWeight: "500",
    color: colors.gray900,
    flexShrink: 1,
    marginLeft: 12,
  },
  actionButton: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    paddingVertical: 14,
    alignItems: "center",
  },
  actionButtonSpaced: {
    marginTop: 12,
  },
  destructiveActionText: {
    color: colors.error,
    fontSize: 15,
    fontWeight: "600",
  },
  linkRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  linkSeparator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: colors.gray200,
    marginLeft: 16,
  },
  linkText: {
    fontSize: 15,
    color: colors.gray900,
  },
  linkChevron: {
    fontSize: 18,
    color: colors.gray400,
  },
  buildInfo: {
    paddingTop: 24,
    paddingBottom: 8,
    alignItems: "center",
  },
  buildInfoText: {
    fontSize: 12,
    color: colors.gray400,
  },
});
