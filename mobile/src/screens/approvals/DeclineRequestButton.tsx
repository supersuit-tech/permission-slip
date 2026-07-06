import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  StyleSheet,
  Text,
} from "react-native";
import { colors } from "../../theme/colors";

type DeclineRequestButtonProps = {
  testID?: string;
  accessibilityLabel?: string;
  onDecline: () => Promise<void>;
  disabled?: boolean;
};

export function DeclineRequestButton({
  testID,
  accessibilityLabel = "Decline request",
  onDecline,
  disabled = false,
}: DeclineRequestButtonProps) {
  const [isDeclining, setIsDeclining] = useState(false);

  const handlePress = useCallback(async () => {
    if (disabled || isDeclining) return;

    setIsDeclining(true);
    try {
      await onDecline();
    } catch {
      Alert.alert("Error", "Failed to decline request. Please try again.");
    } finally {
      setIsDeclining(false);
    }
  }, [disabled, isDeclining, onDecline]);

  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      disabled={disabled || isDeclining}
      onPress={() => {
        void handlePress();
      }}
      style={({ pressed }) => [
        styles.button,
        (disabled || isDeclining) && styles.buttonDisabled,
        pressed && !disabled && !isDeclining && styles.buttonPressed,
      ]}
    >
      {isDeclining ? (
        <ActivityIndicator size="small" color={colors.error} testID="decline-loading" />
      ) : (
        <Text style={styles.icon}>{"\u2715"}</Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    width: 32,
    height: 32,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: 16,
  },
  buttonDisabled: {
    opacity: 0.5,
  },
  buttonPressed: {
    backgroundColor: colors.gray100,
  },
  icon: {
    fontSize: 18,
    lineHeight: 20,
    color: colors.gray500,
  },
});
