/**
 * Full-screen biometric lock overlay shown when biometric auth is enabled
 * and the user has not yet authenticated (e.g. after app resume).
 */
import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { colors } from "../theme/colors";

interface BiometricLockScreenProps {
  onUnlock: () => void;
}

export function BiometricLockScreen({ onUnlock }: BiometricLockScreenProps) {
  return (
    <View style={styles.container}>
      <Text style={styles.icon}>🔒</Text>
      <Text style={styles.title}>Permission Slip</Text>
      <Text style={styles.subtitle}>
        Tap to unlock with biometrics
      </Text>
      <TouchableOpacity
        style={styles.button}
        onPress={onUnlock}
        accessibilityLabel="Unlock with biometrics"
        accessibilityRole="button"
        testID="biometric-unlock-button"
      >
        <Text style={styles.buttonText}>Unlock</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.white,
    paddingHorizontal: 32,
  },
  icon: {
    fontSize: 48,
    marginBottom: 16,
  },
  title: {
    fontSize: 22,
    fontWeight: "700",
    color: colors.gray900,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 15,
    color: colors.gray500,
    textAlign: "center",
    marginBottom: 32,
  },
  button: {
    backgroundColor: colors.gray900,
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 48,
  },
  buttonText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
});
