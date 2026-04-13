/**
 * Custom Server settings — allows the user to point the mobile app at a
 * private Permission Slip deployment instead of the default production host.
 *
 * When enabled, all API calls are routed to the custom host URL and the
 * gateway secret is sent as the X-Gateway-Secret header on every request.
 *
 * Changes take effect on app restart.
 */
import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from "react-native";
import { colors } from "../../theme/colors";
import {
  clearCustomHostConfig,
  getCustomHost,
  getGatewaySecret,
  isCustomHostEnabled,
  loadCustomHostConfig,
  setCustomHostConfig,
} from "../../lib/customHostConfig";

export default function CustomServerSettings() {
  const [enabled, setEnabled] = useState(false);
  const [hostUrl, setHostUrl] = useState("");
  const [secret, setSecret] = useState("");
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState(false);

  // Hydrate from SecureStore on mount.
  useEffect(() => {
    loadCustomHostConfig().then(() => {
      setEnabled(isCustomHostEnabled());
      setHostUrl(getCustomHost() ?? "");
      setSecret(getGatewaySecret() ?? "");
      setLoaded(true);
    });
  }, []);

  const handleToggle = useCallback(
    (value: boolean) => {
      if (!value) {
        // Turning off — clear config.
        setSaving(true);
        clearCustomHostConfig()
          .then(() => {
            setEnabled(false);
            setHostUrl("");
            setSecret("");
            Alert.alert(
              "Custom Server Disabled",
              "The app will use the default server on next restart.",
            );
          })
          .catch(() => {
            Alert.alert("Error", "Failed to clear custom server settings.");
          })
          .finally(() => setSaving(false));
      } else {
        setEnabled(true);
      }
    },
    [],
  );

  const handleSave = useCallback(async () => {
    const trimmedHost = hostUrl.trim();
    if (!trimmedHost) {
      Alert.alert("Missing URL", "Please enter a server URL.");
      return;
    }

    // Basic URL validation.
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
      Alert.alert(
        "Saved",
        "Custom server settings saved. Restart the app for changes to take effect.",
      );
    } catch {
      Alert.alert("Error", "Failed to save custom server settings.");
    } finally {
      setSaving(false);
    }
  }, [hostUrl, secret]);

  if (!loaded) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="small" color={colors.gray500} />
      </View>
    );
  }

  return (
    <View>
      <View style={styles.card}>
        <View style={styles.toggleRow}>
          <View style={styles.toggleLabel}>
            <Text style={styles.toggleTitle}>Custom Server</Text>
            <Text style={styles.toggleDescription}>
              Connect to a private Permission Slip deployment instead of the
              default server.
            </Text>
          </View>
          <Switch
            testID="custom-server-toggle"
            value={enabled}
            onValueChange={handleToggle}
            disabled={saving}
            trackColor={{
              false: colors.gray300,
              true: colors.primary,
            }}
            accessibilityLabel="Custom Server"
            accessibilityRole="switch"
            accessibilityState={{ checked: enabled }}
          />
        </View>
      </View>

      {enabled ? (
        <View style={styles.formCard}>
          <Text style={styles.inputLabel}>Server URL</Text>
          <TextInput
            testID="custom-host-input"
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

          <Text style={[styles.inputLabel, styles.inputLabelSpaced]}>
            Gateway Secret
          </Text>
          <TextInput
            testID="gateway-secret-input"
            style={styles.input}
            placeholder="Optional — leave blank if not required"
            placeholderTextColor={colors.gray400}
            value={secret}
            onChangeText={setSecret}
            autoCapitalize="none"
            autoCorrect={false}
            secureTextEntry
          />

          <TouchableOpacity
            testID="save-custom-server-button"
            style={[styles.saveButton, saving && styles.saveButtonDisabled]}
            onPress={handleSave}
            disabled={saving}
            accessibilityRole="button"
            accessibilityLabel="Save custom server settings"
          >
            {saving ? (
              <ActivityIndicator size="small" color={colors.white} />
            ) : (
              <Text style={styles.saveButtonText}>Save</Text>
            )}
          </TouchableOpacity>

          <Text style={styles.hint}>
            Changes take effect on app restart.
          </Text>
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  loadingContainer: {
    paddingVertical: 24,
    alignItems: "center",
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
  formCard: {
    marginTop: 12,
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.gray200,
    padding: 16,
  },
  inputLabel: {
    fontSize: 13,
    fontWeight: "600",
    color: colors.gray700,
    marginBottom: 6,
  },
  inputLabelSpaced: {
    marginTop: 16,
  },
  input: {
    backgroundColor: colors.primaryBg,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.gray200,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
    color: colors.gray900,
  },
  saveButton: {
    marginTop: 20,
    backgroundColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 12,
    alignItems: "center",
  },
  saveButtonDisabled: {
    opacity: 0.6,
  },
  saveButtonText: {
    color: colors.white,
    fontSize: 15,
    fontWeight: "600",
  },
  hint: {
    marginTop: 10,
    fontSize: 12,
    color: colors.gray400,
    textAlign: "center",
  },
});
