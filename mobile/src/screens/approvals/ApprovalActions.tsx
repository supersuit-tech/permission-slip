/**
 * Approve/Deny action buttons shown at the bottom of the approval detail
 * screen for pending, non-expired approvals.
 *
 * Provides haptic feedback on press: heavy impact for approve, warning
 * notification for deny.
 */
import { useCallback } from "react";
import { ActivityIndicator, StyleSheet, Text, TouchableOpacity, View } from "react-native";
import * as Haptics from "expo-haptics";
import { colors } from "../../theme/colors";

interface ApprovalActionsProps {
  onApprove: () => void;
  onDeny: () => void;
  isApproving: boolean;
  isDenying: boolean;
  disabled: boolean;
}

export function ApprovalActions({
  onApprove,
  onDeny,
  isApproving,
  isDenying,
  disabled,
}: ApprovalActionsProps) {
  const isBusy = isApproving || isDenying;

  const handleApprove = useCallback(() => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
    onApprove();
  }, [onApprove]);

  const handleDeny = useCallback(() => {
    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
    onDeny();
  }, [onDeny]);

  return (
    <View style={styles.container}>
      <TouchableOpacity
        style={[styles.denyButton, (disabled || isBusy) && styles.buttonDisabled]}
        onPress={handleDeny}
        disabled={disabled || isBusy}
        accessibilityLabel="Deny request"
        accessibilityRole="button"
        testID="deny-button"
      >
        {isDenying ? (
          <ActivityIndicator size="small" color={colors.error} />
        ) : (
          <Text style={[styles.denyText, (disabled || isBusy) && styles.textDisabled]}>
            Deny
          </Text>
        )}
      </TouchableOpacity>

      <TouchableOpacity
        style={[styles.approveButton, (disabled || isBusy) && styles.approveButtonDisabled]}
        onPress={handleApprove}
        disabled={disabled || isBusy}
        accessibilityLabel="Approve request"
        accessibilityRole="button"
        testID="approve-button"
      >
        {isApproving ? (
          <ActivityIndicator size="small" color={colors.white} />
        ) : (
          <Text style={[styles.approveText, (disabled || isBusy) && styles.approveTextDisabled]}>
            Approve
          </Text>
        )}
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: "row",
    gap: 12,
    paddingHorizontal: 20,
    paddingVertical: 16,
  },
  denyButton: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.error,
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  denyText: {
    color: colors.error,
    fontSize: 16,
    fontWeight: "600",
  },
  approveButton: {
    flex: 1,
    backgroundColor: colors.success,
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  approveText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
  buttonDisabled: {
    borderColor: colors.gray300,
    opacity: 0.5,
  },
  textDisabled: {
    color: colors.gray400,
  },
  approveButtonDisabled: {
    backgroundColor: colors.gray300,
    opacity: 0.5,
  },
  approveTextDisabled: {
    color: colors.gray500,
  },
});
