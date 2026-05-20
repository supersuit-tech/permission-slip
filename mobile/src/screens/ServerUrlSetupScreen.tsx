import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { colors } from "../theme/colors";
import {
  clearCustomHostConfig,
  setCustomHostConfig,
} from "../lib/customHostConfig";
import { clearStoredRefreshToken } from "../lib/authStorage";

type Props = {
  onComplete: () => void;
  /**
   * Initial value for the server URL field. Pre-fill with the currently
   * saved host when this screen is used to *change* the URL (vs. first
   * launch), so the user can see and edit what's there.
   */
  initialHostUrl?: string;
  /** Initial value for the gateway secret field. */
  initialSecret?: string;
  /**
   * When provided, a Cancel button is shown that calls this callback.
   * Used when the screen is opened as an overlay (e.g. from the
   * connection-error retry screen or the login screen), so the user can
   * back out without changing anything.
   */
  onCancel?: () => void;
};

/**
 * Server URL setup. Shown:
 *   - On first launch when neither EXPO_PUBLIC_API_BASE_URL nor a saved
 *     custom host exists (App.tsx gates rendering on this).
 *   - As an overlay from unauthenticated screens (Connection issue, Login)
 *     so a user pointing at a dead server can fix it without reinstalling.
 *
 * Self-hosted Permission Slip has no default host, so this is the only
 * way to configure connectivity from the device.
 */
export default function ServerUrlSetupScreen({
  onComplete,
  initialHostUrl = "",
  initialSecret = "",
  onCancel,
}: Props) {
  const insets = useSafeAreaInsets();
  const [hostUrl, setHostUrl] = useState(initialHostUrl);
  const [secret, setSecret] = useState(initialSecret);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);

  const isChangeMode = onCancel != null;
  const busy = saving || resetting;

  const handleSave = useCallback(async () => {
    const trimmedHost = hostUrl.trim();
    if (!trimmedHost) {
      Alert.alert("Missing URL", "Enter your Permission Slip server URL (e.g. https://your-pi:8080/api).");
      return;
    }
    try {
      const parsed = new URL(trimmedHost);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        Alert.alert("Invalid URL", "The URL must start with http:// or https://.");
        return;
      }
    } catch {
      Alert.alert("Invalid URL", "Please enter a valid server URL.");
      return;
    }

    setSaving(true);
    try {
      await setCustomHostConfig(trimmedHost, secret.trim() || null);
      // Always drop any cached refresh token when the server URL is saved
      // from this screen — tokens issued by the old host won't work against
      // the new one. Safe to call even when no token is stored.
      await clearStoredRefreshToken();
      onComplete();
    } catch {
      Alert.alert("Error", "Could not save server URL. Please try again.");
    } finally {
      setSaving(false);
    }
  }, [hostUrl, secret, onComplete]);

  const handleReset = useCallback(() => {
    Alert.alert(
      "Reset connection?",
      "This clears the saved server URL, gateway secret, and signed-in session on this device. You'll need to enter the server URL again.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Reset",
          style: "destructive",
          onPress: () => {
            setResetting(true);
            void (async () => {
              try {
                await clearCustomHostConfig();
                await clearStoredRefreshToken();
                setHostUrl("");
                setSecret("");
                onComplete();
              } catch {
                Alert.alert("Error", "Could not reset connection. Please try again.");
              } finally {
                setResetting(false);
              }
            })();
          },
        },
      ],
    );
  }, [onComplete]);

  return (
    <KeyboardAvoidingView
      style={[styles.root, { paddingTop: insets.top + 24, paddingBottom: insets.bottom + 24 }]}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <View style={styles.inner}>
        <Text style={styles.title}>
          {isChangeMode ? "Change server URL" : "Connect to your server"}
        </Text>
        <Text style={styles.subtitle}>
          {isChangeMode
            ? "Update the URL the app talks to. Saving signs you out so the new server takes effect."
            : "Permission Slip is self-hosted. Enter the API base URL for your instance (the same URL you use in the web app, usually ending in /api)."}
        </Text>

        <Text style={styles.label}>Server URL</Text>
        <TextInput
          testID="server-url-setup-input"
          style={styles.input}
          placeholder="https://your-server.example.com/api"
          placeholderTextColor={colors.gray400}
          value={hostUrl}
          onChangeText={setHostUrl}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="url"
          textContentType="URL"
          editable={!busy}
        />

        <Text style={[styles.label, styles.labelSpaced]}>Gateway secret</Text>
        <TextInput
          testID="server-url-setup-secret"
          style={styles.input}
          placeholder="Optional — only if your server requires X-Gateway-Secret"
          placeholderTextColor={colors.gray400}
          value={secret}
          onChangeText={setSecret}
          autoCapitalize="none"
          autoCorrect={false}
          secureTextEntry
          editable={!busy}
        />

        <TouchableOpacity
          testID="server-url-setup-continue"
          style={[styles.button, busy && styles.buttonDisabled]}
          onPress={() => {
            void handleSave();
          }}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel={isChangeMode ? "Save server URL" : "Continue with this server URL"}
        >
          {saving ? (
            <ActivityIndicator size="small" color={colors.white} />
          ) : (
            <Text style={styles.buttonText}>{isChangeMode ? "Save" : "Continue"}</Text>
          )}
        </TouchableOpacity>

        {isChangeMode ? (
          <TouchableOpacity
            testID="server-url-setup-cancel"
            style={[styles.secondaryButton, busy && styles.buttonDisabled]}
            onPress={onCancel}
            disabled={busy}
            accessibilityRole="button"
            accessibilityLabel="Cancel"
          >
            <Text style={styles.secondaryButtonText}>Cancel</Text>
          </TouchableOpacity>
        ) : null}

        <TouchableOpacity
          testID="server-url-setup-reset"
          style={[styles.resetButton, busy && styles.buttonDisabled]}
          onPress={handleReset}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel="Reset saved connection"
        >
          {resetting ? (
            <ActivityIndicator size="small" color={colors.gray700} />
          ) : (
            <Text style={styles.resetButtonText}>Reset saved connection</Text>
          )}
        </TouchableOpacity>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: colors.white,
    paddingHorizontal: 24,
  },
  inner: {
    flex: 1,
    justifyContent: "center",
  },
  title: {
    fontSize: 22,
    fontWeight: "700",
    color: colors.gray900,
    marginBottom: 12,
  },
  subtitle: {
    fontSize: 15,
    color: colors.gray500,
    lineHeight: 22,
    marginBottom: 28,
  },
  label: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.gray700,
    marginBottom: 8,
  },
  labelSpaced: {
    marginTop: 16,
  },
  input: {
    backgroundColor: colors.primaryBg,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.gray200,
    paddingHorizontal: 12,
    paddingVertical: 12,
    fontSize: 16,
    color: colors.gray900,
  },
  button: {
    marginTop: 28,
    backgroundColor: colors.gray900,
    borderRadius: 10,
    paddingVertical: 14,
    alignItems: "center",
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
  secondaryButton: {
    marginTop: 12,
    borderRadius: 10,
    paddingVertical: 14,
    alignItems: "center",
    borderWidth: 1,
    borderColor: colors.gray300,
    backgroundColor: colors.white,
  },
  secondaryButtonText: {
    color: colors.gray900,
    fontSize: 16,
    fontWeight: "600",
  },
  resetButton: {
    marginTop: 16,
    paddingVertical: 12,
    alignItems: "center",
  },
  resetButtonText: {
    color: colors.gray500,
    fontSize: 14,
    fontWeight: "500",
    textDecorationLine: "underline",
  },
});
