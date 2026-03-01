import { useEffect } from "react";
import { Alert } from "react-native";
import { NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import LoginScreen from "../screens/LoginScreen";
import HomeScreen from "../screens/HomeScreen";

export type RootStackParamList = {
  Login: undefined;
  Home: undefined;
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
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {authStatus === "authenticated" ? (
          <Stack.Screen name="Home" component={HomeScreen} />
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
