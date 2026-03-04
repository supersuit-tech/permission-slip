/**
 * DenyAction — self-contained deny interaction for a pending approval.
 *
 * Renders one of two states:
 * 1. **Button state** — a red outlined "Deny" button. Pressing it shows a
 *    native `Alert.alert` confirmation dialog. Confirming fires the deny
 *    API call via `useDenyApproval`.
 * 2. **Confirmation state** — after a successful deny, shows a "Request
 *    Denied" banner with a "Back to List" button. Auto-calls `onDenied`
 *    after 1.5 s so the parent can navigate away.
 *
 * The parent (ApprovalDetailScreen) controls *when* DenyAction is rendered
 * (only for pending, non-expired approvals) and provides the `onDenied`
 * callback for navigation.
 */
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useDenyApproval } from "../../hooks/useDenyApproval";
import { colors } from "../../theme/colors";

interface DenyActionProps {
  approvalId: string;
  /** Called after the deny succeeds — typically used to navigate away. */
  onDenied: () => void;
}

export function DenyAction({ approvalId, onDenied }: DenyActionProps) {
  const { denyApproval, isPending: isDenying } = useDenyApproval();
  const [denied, setDenied] = useState(false);
  const autoNavTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (denied) {
      autoNavTimer.current = setTimeout(() => {
        onDenied();
      }, 1500);
    }
    return () => {
      if (autoNavTimer.current) clearTimeout(autoNavTimer.current);
    };
  }, [denied, onDenied]);

  const handleDeny = useCallback(() => {
    Alert.alert(
      "Deny Request",
      "Are you sure you want to deny this request?",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Deny",
          style: "destructive",
          onPress: async () => {
            try {
              await denyApproval(approvalId);
              setDenied(true);
            } catch {
              Alert.alert("Error", "Failed to deny request. Please try again.");
            }
          },
        },
      ],
    );
  }, [denyApproval, approvalId]);

  if (denied) {
    return (
      <View style={styles.deniedConfirmation} accessibilityRole="alert" testID="denied-confirmation">
        <Text style={styles.deniedConfirmationTitle}>Request Denied</Text>
        <Text style={styles.deniedConfirmationSubtitle}>
          Returning to list...
        </Text>
        <Pressable
          testID="back-to-list-button"
          style={styles.backToListButton}
          onPress={() => {
            // Clear the auto-nav timer so onDenied isn't called twice.
            if (autoNavTimer.current) clearTimeout(autoNavTimer.current);
            onDenied();
          }}
          accessibilityRole="button"
          accessibilityLabel="Go back to approval list"
        >
          <Text style={styles.backToListButtonText}>Back to List</Text>
        </Pressable>
      </View>
    );
  }

  return (
    <View style={styles.section}>
      <Pressable
        testID="deny-button"
        style={({ pressed }) => [
          styles.denyButton,
          pressed && styles.denyButtonPressed,
          isDenying && styles.denyButtonDisabled,
        ]}
        onPress={handleDeny}
        disabled={isDenying}
        accessibilityRole="button"
        accessibilityLabel="Deny request"
        accessibilityHint="Double-tap to deny this approval request"
        accessibilityState={{ disabled: isDenying, busy: isDenying }}
      >
        {isDenying ? (
          <ActivityIndicator
            testID="deny-loading"
            accessibilityLabel="Denying request"
            color={colors.error}
            size="small"
          />
        ) : (
          <Text style={styles.denyButtonText}>Deny</Text>
        )}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  section: {
    paddingHorizontal: 20,
    marginTop: 20,
  },
  deniedConfirmation: {
    backgroundColor: colors.riskHighBg,
    paddingVertical: 24,
    paddingHorizontal: 20,
    alignItems: "center",
    gap: 6,
  },
  deniedConfirmationTitle: {
    fontSize: 18,
    fontWeight: "700",
    color: colors.error,
  },
  deniedConfirmationSubtitle: {
    fontSize: 14,
    color: colors.gray500,
    marginBottom: 8,
  },
  backToListButton: {
    paddingVertical: 8,
    paddingHorizontal: 20,
    borderRadius: 8,
    backgroundColor: colors.white,
    borderWidth: 1,
    borderColor: colors.gray200,
  },
  backToListButtonText: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.gray700,
  },
  denyButton: {
    backgroundColor: colors.white,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: colors.error,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  denyButtonPressed: {
    backgroundColor: colors.riskHighBg,
  },
  denyButtonDisabled: {
    opacity: 0.6,
  },
  denyButtonText: {
    fontSize: 16,
    fontWeight: "600",
    color: colors.error,
  },
});
