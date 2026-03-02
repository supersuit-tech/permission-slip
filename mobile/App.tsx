import { useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./src/auth/AuthContext";
import RootNavigator from "./src/navigation/RootNavigator";
import { colors } from "./src/theme/colors";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Don't keep cached data from a previous session visible during
      // loading of a new session (important after sign-out/sign-in).
      gcTime: 5 * 60 * 1000,
    },
  },
});

const LOADING_TIMEOUT_MS = 10_000;

function AppContent() {
  const { authStatus } = useAuth();
  const qc = useQueryClient();
  const [timedOut, setTimedOut] = useState(false);
  const prevAuthStatus = useRef(authStatus);

  // Clear the React Query cache when the user signs out so the next user
  // on a shared device never sees stale approval data from a prior session.
  useEffect(() => {
    if (
      prevAuthStatus.current === "authenticated" &&
      authStatus === "unauthenticated"
    ) {
      qc.clear();
    }
    prevAuthStatus.current = authStatus;
  }, [authStatus, qc]);

  // If loading takes longer than 10 seconds, show an error with a retry option.
  useEffect(() => {
    if (authStatus !== "loading") {
      setTimedOut(false);
      return;
    }
    const timer = setTimeout(() => setTimedOut(true), LOADING_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [authStatus]);

  if (authStatus === "loading") {
    return (
      <View style={styles.loading}>
        {timedOut ? (
          <>
            <Text style={styles.errorTitle}>Connection issue</Text>
            <Text style={styles.errorBody}>
              Unable to reach the server. Check your connection and try again.
            </Text>
            <TouchableOpacity
              testID="loading-retry"
              accessibilityLabel="Retry connection"
              accessibilityRole="button"
              style={styles.retryButton}
              onPress={() => setTimedOut(false)}
            >
              <Text style={styles.retryText}>Retry</Text>
            </TouchableOpacity>
          </>
        ) : (
          <ActivityIndicator size="large" color={colors.gray900} />
        )}
      </View>
    );
  }

  return <RootNavigator />;
}

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <AppContent />
          <StatusBar style="auto" />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    backgroundColor: colors.white,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 32,
  },
  errorTitle: {
    fontSize: 18,
    fontWeight: "600",
    color: colors.gray900,
    marginBottom: 8,
    textAlign: "center",
  },
  errorBody: {
    fontSize: 15,
    color: colors.gray500,
    textAlign: "center",
    marginBottom: 24,
    lineHeight: 22,
  },
  retryButton: {
    backgroundColor: colors.gray900,
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 32,
  },
  retryText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "600",
  },
});
