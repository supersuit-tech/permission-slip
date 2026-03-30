import { ActivityIndicator, StyleSheet, Text, View } from "react-native";
import { NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import LoginScreen from "../screens/LoginScreen";
import MfaChallengeScreen from "../auth/MfaChallengeScreen";
import ApprovalListScreen from "../screens/approvals/ApprovalListScreen";
import ApprovalDetailScreen from "../screens/approvals/ApprovalDetailScreen";
import DeepLinkDetailScreen from "../screens/approvals/DeepLinkDetailScreen";
import SettingsScreen from "../screens/settings/SettingsScreen";
import type { ApprovalSummary } from "../hooks/useApprovals";
import { linking } from "./linking";
import { navigationRef } from "./navigationRef";
import { colors } from "../theme/colors";

export type RootStackParamList = {
  Login: undefined;
  MfaChallenge: undefined;
  ApprovalList: undefined;
  ApprovalDetail: {
    approvalId: string;
    approval: ApprovalSummary;
  };
  DeepLinkDetail: {
    approvalId: string;
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
          <ActivityIndicator size="large" color={colors.gray900} />
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
              name="Settings"
              component={SettingsScreen}
              options={{
                headerShown: true,
                headerTitle: "Settings",
                headerBackTitle: "Back",
              }}
            />
          </>
        ) : authStatus === "mfa_required" ? (
          <Stack.Screen name="MfaChallenge" component={MfaChallengeScreen} />
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
