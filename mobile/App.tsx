import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  AppState,
  type AppStateStatus,
  Platform,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { focusManager, QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./src/auth/AuthContext";
import RootNavigator from "./src/navigation/RootNavigator";
import { ErrorBoundary } from "./src/components/ErrorBoundary";
import { usePushSetup } from "./src/hooks/usePushSetup";
import { useBiometricAuth } from "./src/hooks/useBiometricAuth";
import { BiometricLockScreen } from "./src/screens/BiometricLockScreen";
import { colors } from "./src/theme/colors";

// Tell React Query when the app returns to the foreground so queries with
// refetchOnWindowFocus automatically re-fetch (AppState is the RN equivalent
// of the browser's visibilitychange event).
function onAppStateChange(status: AppStateStatus) {
  if (Platform.OS !== "web") {
    focusManager.setFocused(status === "active");
  }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 30_000,
    },
  },
});

const LOADING_TIMEOUT_MS = 10_000;

function AppContent({ onRetry }: { onRetry: () => void }) {
  const { authStatus } = useAuth();
  const qc = useQueryClient();
  const [timedOut, setTimedOut] = useState(false);
  const prevAuthStatus = useRef(authStatus);

  // Register/unregister Expo push token as auth status changes
  usePushSetup();

  // Biometric auth gate — locks the app on resume from background
  const biometric = useBiometricAuth();
  const appStateRef = useRef(AppState.currentState);

  useEffect(() => {
    const sub = AppState.addEventListener("change", (nextState) => {
      // Re-lock when app goes to background and comes back
      if (
        appStateRef.current.match(/inactive|background/) &&
        nextState === "active" &&
        biometric.isEnabled
      ) {
        biometric.setIsAuthenticated(false);
        biometric.authenticate();
      }
      appStateRef.current = nextState;
    });
    return () => sub.remove();
  }, [biometric.isEnabled, biometric.authenticate, biometric.setIsAuthenticated]);

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
              onPress={onRetry}
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

  // Show biometric lock screen when authenticated but biometric hasn't passed
  if (
    authStatus === "authenticated" &&
    biometric.isEnabled &&
    !biometric.isAuthenticated
  ) {
    return <BiometricLockScreen onUnlock={biometric.authenticate} />;
  }

  return (
    <ErrorBoundary>
      <RootNavigator />
    </ErrorBoundary>
  );
}

export default function App() {
  // Subscribe to AppState changes so React Query knows when the app is focused.
  useEffect(() => {
    const sub = AppState.addEventListener("change", onAppStateChange);
    return () => sub.remove();
  }, []);

  // Incrementing the key re-mounts AuthProvider, which re-triggers
  // Supabase's onAuthStateChange and retries the initial session check.
  const [authKey, setAuthKey] = useState(0);
  const handleRetry = useCallback(() => setAuthKey((k) => k + 1), []);

  return (
    <QueryClientProvider client={queryClient}>
      <SafeAreaProvider>
        <AuthProvider key={authKey}>
          <AppContent onRetry={handleRetry} />
          <StatusBar style="auto" />
        </AuthProvider>
      </SafeAreaProvider>
    </QueryClientProvider>
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
