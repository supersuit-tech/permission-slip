/**
 * Full-screen biometric lock overlay shown when biometric auth is enabled
 * and the user has not yet authenticated (e.g. after app resume).
 *
 * Automatically triggers the biometric prompt on mount so the user doesn't
 * have to tap — they just see the FaceID/TouchID dialog immediately.
 * The manual "Unlock" button remains as a fallback if the auto-prompt
 * is dismissed or fails.
 */
import { useCallback, useEffect, useRef } from "react";
import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import * as Haptics from "expo-haptics";
import { colors } from "../theme/colors";

interface BiometricLockScreenProps {
  onUnlock: () => void;
}

export function BiometricLockScreen({ onUnlock }: BiometricLockScreenProps) {
  const hasPrompted = useRef(false);

  // Auto-prompt biometric on mount for a seamless unlock experience
  useEffect(() => {
    if (!hasPrompted.current) {
      hasPrompted.current = true;
      onUnlock();
    }
  }, [onUnlock]);

  const handleUnlock = useCallback(() => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    onUnlock();
  }, [onUnlock]);

  return (
    <View style={styles.container}>
      <View style={styles.lockIcon}>
        <Text style={styles.lockIconText}>P</Text>
      </View>
      <Text style={styles.title}>Permission Slip</Text>
      <Text style={styles.subtitle}>
        Use biometrics to unlock
      </Text>
      <TouchableOpacity
        style={styles.button}
        onPress={handleUnlock}
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
  lockIcon: {
    width: 72,
    height: 72,
    borderRadius: 16,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: 16,
  },
  lockIconText: {
    fontSize: 28,
    fontWeight: "700",
    color: colors.white,
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
    backgroundColor: colors.primary,
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
