/**
 * Wrapper screen for deep link navigation to approval details.
 *
 * Deep links (permissionslip://permission-slip/approve/{approvalId}) only
 * provide the approval ID, not the full ApprovalSummary object. This screen
 * fetches the approval by ID then renders the standard ApprovalDetailScreen.
 *
 * Shows a loading spinner while fetching, and an error state with retry
 * and fallback navigation if the approval can't be found.
 */
import { useEffect } from "react";
import {
  ActivityIndicator,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useApproval } from "../../hooks/useApproval";
import { colors } from "../../theme/colors";

type Props = NativeStackScreenProps<RootStackParamList, "DeepLinkDetail">;

export default function DeepLinkDetailScreen({ route, navigation }: Props) {
  const { approvalId } = route.params;
  const { approval, isLoading, error, refetch } = useApproval(approvalId);

  // Once we have the approval, replace this screen with the real detail screen
  useEffect(() => {
    if (approval) {
      navigation.replace("ApprovalDetail", {
        approvalId: approval.approval_id,
        approval,
      });
    }
  }, [approval, navigation]);

  if (isLoading || approval) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={styles.text}>Loading approval...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.container}>
        <Text style={styles.errorTitle}>Approval not found</Text>
        <Text style={styles.errorBody}>{error}</Text>
        <View style={styles.actions}>
          <TouchableOpacity
            style={styles.retryButton}
            onPress={() => refetch()}
            accessibilityLabel="Retry loading approval"
            accessibilityRole="button"
          >
            <Text style={styles.retryButtonText}>Retry</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={styles.button}
            onPress={() => navigation.replace("ApprovalList")}
            accessibilityLabel="Go to approval list"
            accessibilityRole="button"
          >
            <Text style={styles.buttonText}>Go to Approvals</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  }

  return null;
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.white,
    paddingHorizontal: 32,
  },
  text: {
    marginTop: 12,
    fontSize: 14,
    color: colors.gray500,
  },
  errorTitle: {
    fontSize: 18,
    fontWeight: "600",
    color: colors.gray900,
    marginBottom: 8,
  },
  errorBody: {
    fontSize: 14,
    color: colors.gray500,
    textAlign: "center",
    marginBottom: 24,
  },
  actions: {
    flexDirection: "row",
    gap: 12,
  },
  retryButton: {
    borderWidth: 1,
    borderColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 24,
  },
  retryButtonText: {
    color: colors.primary,
    fontSize: 16,
    fontWeight: "600",
  },
  button: {
    backgroundColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 24,
  },
  buttonText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
});
