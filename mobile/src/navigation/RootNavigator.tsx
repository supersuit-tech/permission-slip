import { ActivityIndicator, StyleSheet, Text, View } from "react-native";
import { NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import LoginScreen from "../screens/LoginScreen";
import ApprovalListScreen from "../screens/approvals/ApprovalListScreen";
import ApprovalDetailScreen from "../screens/approvals/ApprovalDetailScreen";
import DeepLinkDetailScreen from "../screens/approvals/DeepLinkDetailScreen";
import StandingApprovalRequestDetailScreen from "../screens/approvals/StandingApprovalRequestDetailScreen";
import DeepLinkRuleDetailScreen from "../screens/approvals/DeepLinkRuleDetailScreen";
import BulkApprovalGroupScreen from "../screens/approvals/BulkApprovalGroupScreen";
import DeepLinkBulkGroupScreen from "../screens/approvals/DeepLinkBulkGroupScreen";
import type { StandingApprovalRequestSummary } from "../hooks/useStandingApprovalRequests";
import SettingsScreen from "../screens/settings/SettingsScreen";
import type { ApprovalSummary } from "../hooks/useApprovals";
import { linking } from "./linking";
import { navigationRef } from "./navigationRef";
import { colors } from "../theme/colors";

export type RootStackParamList = {
  Login: undefined;
  ApprovalList: undefined;
  ApprovalDetail: {
    approvalId: string;
    approval: ApprovalSummary;
  };
  DeepLinkDetail: {
    approvalId: string;
  };
  StandingApprovalRequestDetail: {
    requestId: string;
    request: StandingApprovalRequestSummary;
  };
  DeepLinkRuleDetail: {
    requestId: string;
  };
  BulkApprovalGroup: {
    bulkGroupId: string;
  };
  DeepLinkBulkGroup: {
    bulkGroupId: string;
  };
  Settings: undefined;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function RootNavigator() {
  const { authStatus } = useAuth();

  return (
    <NavigationContainer
      ref={navigationRef}
      linking={linking}
      fallback={
        <View style={styles.fallback}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.fallbackText}>Loading...</Text>
        </View>
      }
    >
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {authStatus === "authenticated" ? (
          <>
            <Stack.Screen name="ApprovalList" component={ApprovalListScreen} />
            <Stack.Screen
              name="ApprovalDetail"
              component={ApprovalDetailScreen}
              options={{
                headerShown: true,
                headerTitle: "Approval Details",
                headerBackTitle: "Back",
              }}
            />
            <Stack.Screen
              name="DeepLinkDetail"
              component={DeepLinkDetailScreen}
              options={{
                headerShown: true,
                headerTitle: "Approval Details",
                headerBackTitle: "Back",
              }}
            />
            <Stack.Screen
              name="StandingApprovalRequestDetail"
              component={StandingApprovalRequestDetailScreen}
              options={{
                headerShown: true,
                headerTitle: "Rule Proposal",
                headerBackTitle: "Back",
              }}
            />
            <Stack.Screen
              name="DeepLinkRuleDetail"
              component={DeepLinkRuleDetailScreen}
              options={{
                headerShown: true,
                headerTitle: "Rule Proposal",
                headerBackTitle: "Back",
              }}
            />
            <Stack.Screen
              name="BulkApprovalGroup"
              component={BulkApprovalGroupScreen}
              options={{
                headerShown: true,
                headerTitle: "Bulk Approval",
                headerBackTitle: "Back",
              }}
            />
            <Stack.Screen
              name="DeepLinkBulkGroup"
              component={DeepLinkBulkGroupScreen}
              options={{ headerShown: false }}
            />
            <Stack.Screen
              name="Settings"
              component={SettingsScreen}
              options={{
                headerShown: true,
                headerTitle: "Settings",
                headerBackTitle: "Back",
              }}
            />
          </>
        ) : (
          <Stack.Screen
            name="Login"
            component={LoginScreen}
            options={{ animationTypeForReplace: "pop" }}
          />
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
}

const styles = StyleSheet.create({
  fallback: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.white,
  },
  fallbackText: {
    marginTop: 12,
    fontSize: 14,
    color: colors.gray500,
  },
});
