import { useEffect } from "react";
import { Alert } from "react-native";
import {
  NavigationContainer,
  createNavigationContainerRef,
} from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import LoginScreen from "../screens/LoginScreen";
import ApprovalListScreen from "../screens/approvals/ApprovalListScreen";
import ApprovalDetailScreen from "../screens/approvals/ApprovalDetailScreen";
import type { ApprovalSummary } from "../hooks/useApprovals";

export type RootStackParamList = {
  Login: undefined;
  ApprovalList: undefined;
  ApprovalDetail: {
    approvalId: string;
    approval: ApprovalSummary;
  };
};

export const navigationRef = createNavigationContainerRef<RootStackParamList>();

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function RootNavigator() {
  const { authStatus, signOut } = useAuth();

  // MFA challenge is not yet supported on mobile. If the user's account
  // requires MFA, sign them out and show a message. A proper MFA challenge
  // screen will be added in a future phase.
  useEffect(() => {
    if (authStatus === "mfa_required") {
      Alert.alert(
        "MFA required",
        "Multi-factor authentication is not yet supported in the mobile app. Please sign in via the web app.",
        [{ text: "OK", onPress: () => signOut() }]
      );
    }
  }, [authStatus, signOut]);

  return (
    <NavigationContainer ref={navigationRef}>
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
