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
import { setCustomHostConfig } from "../lib/customHostConfig";

type Props = {
  onComplete: () => void;
};

/**
 * First-launch screen when neither EXPO_PUBLIC_API_BASE_URL nor a saved
 * custom server URL is present. Self-hosted Permission Slip has no default host.
 */
export default function ServerUrlSetupScreen({ onComplete }: Props) {
  const insets = useSafeAreaInsets();
  const [hostUrl, setHostUrl] = useState("");
  const [secret, setSecret] = useState("");
  const [saving, setSaving] = useState(false);

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
      onComplete();
    } catch {
      Alert.alert("Error", "Could not save server URL. Please try again.");
    } finally {
      setSaving(false);
    }
  }, [hostUrl, secret, onComplete]);

  return (
    <KeyboardAvoidingView
      style={[styles.root, { paddingTop: insets.top + 24, paddingBottom: insets.bottom + 24 }]}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <View style={styles.inner}>
        <Text style={styles.title}>Connect to your server</Text>
        <Text style={styles.subtitle}>
          Permission Slip is self-hosted. Enter the API base URL for your instance
          (the same URL you use in the web app, usually ending in /api).
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
        />

        <TouchableOpacity
          testID="server-url-setup-continue"
          style={[styles.button, saving && styles.buttonDisabled]}
          onPress={() => {
            void handleSave();
          }}
          disabled={saving}
          accessibilityRole="button"
          accessibilityLabel="Continue with this server URL"
        >
          {saving ? (
            <ActivityIndicator size="small" color={colors.white} />
          ) : (
            <Text style={styles.buttonText}>Continue</Text>
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
});
