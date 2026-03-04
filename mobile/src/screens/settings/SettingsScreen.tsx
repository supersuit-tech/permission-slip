/**
 * Settings screen — allows the user to manage notification preferences
 * and sign out. Currently shows a toggle for the mobile-push notification
 * channel; other channels are managed via the web app.
 */
import { useCallback } from "react";
import {
  ActivityIndicator,
  Alert,
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
import { colors } from "../../theme/colors";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

export default function SettingsScreen(_props: Props) {
  const insets = useSafeAreaInsets();
  const { signOut, user } = useAuth();
  const { preferences, isLoading, error, refetch } =
    useNotificationPreferences();
  const { updatePreferences, isUpdating } =
    useUpdateNotificationPreferences();

  const mobilePushPref = preferences.find((p) => p.channel === "mobile-push");
  const mobilePushEnabled = mobilePushPref?.enabled ?? true;

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

  const handleSignOut = useCallback(() => {
    Alert.alert("Sign out", "Are you sure you want to sign out?", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign Out", style: "destructive", onPress: () => signOut() },
    ]);
  }, [signOut]);

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={{ paddingBottom: insets.bottom + 24 }}
    >
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Notifications</Text>
        <Text style={styles.sectionDescription}>
          Control how you receive approval request notifications on this device.
        </Text>

        {isLoading ? (
          <View style={styles.loadingContainer}>
            <ActivityIndicator
              size="small"
              color={colors.gray500}
              testID="prefs-loading"
            />
          </View>
        ) : error ? (
          <View style={styles.errorContainer}>
            <Text style={styles.errorText}>{error}</Text>
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
          style={[styles.signOutButton, user?.email ? styles.signOutButtonSpaced : null]}
          accessibilityRole="button"
          accessibilityLabel="Sign out of your account"
          onPress={handleSignOut}
        >
          <Text style={styles.signOutText}>Sign Out</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.gray50,
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
  signOutButton: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    paddingVertical: 14,
    alignItems: "center",
  },
  signOutButtonSpaced: {
    marginTop: 12,
  },
  signOutText: {
    color: colors.error,
    fontSize: 15,
    fontWeight: "600",
  },
});
