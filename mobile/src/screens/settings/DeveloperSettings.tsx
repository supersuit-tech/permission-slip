import { useCallback, useSyncExternalStore } from "react";
import { StyleSheet, Switch, Text, View } from "react-native";
import {
  isDevModeEnabled,
  setDevModeEnabled,
  subscribeDevMode,
} from "../../lib/devModeConfig";
import { colors } from "../../theme/colors";

/**
 * Settings section that exposes the Developer Mode toggle. When on, every
 * API request the app makes is captured into an in-memory ring buffer and
 * surfaced via {@link DevLogsOverlay} pinned to the bottom of the screen.
 * Useful for diagnosing self-hosted server issues without server log access.
 */
export default function DeveloperSettings() {
  const enabled = useSyncExternalStore(
    subscribeDevMode,
    isDevModeEnabled,
    isDevModeEnabled,
  );
  const onToggle = useCallback(() => {
    void setDevModeEnabled(!enabled);
  }, [enabled]);

  return (
    <View style={styles.section}>
      <Text style={styles.sectionTitle}>Developer</Text>
      <Text style={styles.sectionDescription}>
        Show a live log of every API request and response at the bottom of
        the screen. Useful for diagnosing connectivity or server errors.
      </Text>
      <View style={styles.card}>
        <View style={styles.toggleRow}>
          <View style={styles.toggleLabel}>
            <Text style={styles.toggleTitle}>Developer Mode</Text>
            <Text style={styles.toggleDescription}>
              Captures recent requests in memory only. Turn off to hide.
            </Text>
          </View>
          <Switch
            testID="developer-mode-toggle"
            value={enabled}
            onValueChange={onToggle}
            trackColor={{ false: colors.gray300, true: colors.primary }}
            accessibilityLabel="Developer Mode"
            accessibilityRole="switch"
            accessibilityState={{ checked: enabled }}
          />
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
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
});
