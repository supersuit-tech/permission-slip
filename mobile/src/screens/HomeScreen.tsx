import { Alert, StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { useAuth } from "../auth/AuthContext";
import { colors } from "../theme/colors";

/** Placeholder home screen shown after successful authentication. */
export default function HomeScreen() {
  const { signOut, user } = useAuth();

  const handleSignOut = async () => {
    const { error } = await signOut();
    if (error) {
      Alert.alert("Sign out failed", "Please try again.");
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Permission Slip</Text>
      <Text testID="home-email" style={styles.subtitle}>
        Signed in as {user?.email ?? "unknown"}
      </Text>
      <Text style={styles.placeholder}>
        Approval list coming in Phase 2.
      </Text>
      <TouchableOpacity
        testID="sign-out"
        accessibilityLabel="Sign out"
        accessibilityRole="button"
        style={styles.signOutButton}
        onPress={handleSignOut}
      >
        <Text style={styles.signOutText}>Sign Out</Text>
      </TouchableOpacity>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.white,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 24,
  },
  title: {
    fontSize: 24,
    fontWeight: "700",
    color: colors.gray900,
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 14,
    color: colors.gray500,
    marginBottom: 24,
  },
  placeholder: {
    fontSize: 15,
    color: colors.gray400,
    marginBottom: 32,
  },
  signOutButton: {
    borderWidth: 1,
    borderColor: colors.gray300,
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 24,
  },
  signOutText: {
    color: colors.gray700,
    fontSize: 15,
    fontWeight: "500",
  },
});
