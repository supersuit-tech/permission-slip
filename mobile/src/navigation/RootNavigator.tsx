import { useEffect } from "react";
import { ActivityIndicator, Alert, StyleSheet, Text, View } from "react-native";
import { NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import LoginScreen from "../screens/LoginScreen";
import ApprovalListScreen from "../screens/approvals/ApprovalListScreen";
import ApprovalDetailScreen from "../screens/approvals/ApprovalDetailScreen";
import DeepLinkDetailScreen from "../screens/approvals/DeepLinkDetailScreen";
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
};

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
