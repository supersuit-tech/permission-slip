import { useEffect } from "react";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useStandingApprovalRequest } from "../../hooks/useStandingApprovalRequest";
import { colors } from "../../theme/colors";

type Props = NativeStackScreenProps<RootStackParamList, "DeepLinkRuleDetail">;

export default function DeepLinkRuleDetailScreen({ route, navigation }: Props) {
  const { requestId } = route.params;
  const { request, isLoading, error } = useStandingApprovalRequest(requestId);

  useEffect(() => {
    if (request) {
      navigation.replace("StandingApprovalRequestDetail", {
        requestId: request.request_id,
        request,
      });
    }
  }, [request, navigation]);

  if (isLoading || request) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={styles.text}>Loading rule proposal...</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.container}>
        <Text style={styles.errorTitle}>Rule proposal not found</Text>
        <Text style={styles.errorBody}>{error}</Text>
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
  text: { marginTop: 12, fontSize: 14, color: colors.gray500 },
  errorTitle: { fontSize: 18, fontWeight: "600", color: colors.gray900, marginBottom: 8 },
  errorBody: { fontSize: 14, color: colors.gray500, textAlign: "center" },
});
