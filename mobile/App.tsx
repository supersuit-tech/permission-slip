import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
import { MockAuthProvider } from "./src/auth/MockAuthProvider";
import RootNavigator from "./src/navigation/RootNavigator";
import { ErrorBoundary } from "./src/components/ErrorBoundary";
import { usePushSetup } from "./src/hooks/usePushSetup";
import { useBiometricAuth } from "./src/hooks/useBiometricAuth";
import { BiometricLockScreen } from "./src/screens/BiometricLockScreen";
import { colors } from "./src/theme/colors";
import {
  getCustomHost,
  getGatewaySecret,
  hasConfiguredApiBase,
  loadCustomHostConfig,
} from "./src/lib/customHostConfig";
import ServerUrlSetupScreen from "./src/screens/ServerUrlSetupScreen";
import { ServerSetupContext } from "./src/lib/serverSetupContext";

const useMockAuth = __DEV__ && process.env.EXPO_PUBLIC_MOCK_AUTH === "true";
const ActiveAuthProvider = useMockAuth ? MockAuthProvider : AuthProvider;

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

function AppContent({
  onRetry,
  onOpenServerSetup,
}: {
  onRetry: () => void;
  onOpenServerSetup: () => void;
}) {
  const { authStatus, session } = useAuth();
  const qc = useQueryClient();
  const [timedOut, setTimedOut] = useState(false);
  const prevAuthStatus = useRef(authStatus);

  // Register/unregister Expo push token as auth status changes
  usePushSetup();

  // Biometric auth gate — locks the app on resume from background.
  // Pass userId so the preference is scoped per user on shared devices.
  const biometric = useBiometricAuth({ userId: session?.user?.id });
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
            <TouchableOpacity
              testID="loading-change-server"
              accessibilityLabel="Change server URL"
              accessibilityRole="button"
              style={styles.secondaryButton}
              onPress={onOpenServerSetup}
            >
              <Text style={styles.secondaryButtonText}>Change server URL</Text>
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
  // Hydrate custom host config from SecureStore BEFORE mounting any subtree
  // that can issue API calls. Without this gate, the first one or more
  // requests after cold start would bypass the custom host and gateway
  // secret, causing surprising 403s against a gateway-locked server.
  const [hostHydrated, setHostHydrated] = useState(false);
  const [serverSetupBump, setServerSetupBump] = useState(0);
  useEffect(() => {
    loadCustomHostConfig().finally(() => setHostHydrated(true));
  }, []);

  // Subscribe to AppState changes so React Query knows when the app is focused.
  useEffect(() => {
    const sub = AppState.addEventListener("change", onAppStateChange);
    return () => sub.remove();
  }, []);

  // Incrementing the key re-mounts AuthProvider, which re-runs bootstrap
  // from secure-stored refresh token (useful after a connection timeout).
  const [authKey, setAuthKey] = useState(0);
  const handleRetry = useCallback(() => setAuthKey((k) => k + 1), []);

  // Overlay state for the in-app "Change server URL" affordance. Reachable
  // from the Connection issue screen and the Login screen so a user stuck
  // against a dead/unreachable host can fix it without reinstalling. (On
  // iOS, expo-secure-store uses Keychain, which persists across uninstalls,
  // so even a fresh install does not always reset the saved URL.)
  const [serverSetupOpen, setServerSetupOpen] = useState(false);
  const openServerSetup = useCallback(() => setServerSetupOpen(true), []);
  const closeServerSetup = useCallback(() => setServerSetupOpen(false), []);
  const serverSetupContextValue = useMemo(
    () => ({ openServerSetup }),
    [openServerSetup],
  );

  if (!hostHydrated) {
    return (
      <SafeAreaProvider>
        <View style={styles.loading}>
          <ActivityIndicator size="large" color={colors.gray900} />
        </View>
        <StatusBar style="auto" />
      </SafeAreaProvider>
    );
  }

  // First-launch setup: no env URL and no saved host. Render the setup
  // screen unconditionally; it owns the entire UI until a host is saved.
  if (!useMockAuth && !hasConfiguredApiBase()) {
    return (
      <SafeAreaProvider>
        <ServerUrlSetupScreen
          onComplete={() => {
            setServerSetupBump((n) => n + 1);
          }}
        />
        <StatusBar style="auto" />
      </SafeAreaProvider>
    );
  }

  // "Change server URL" overlay. Pre-fills the current saved values so the
  // user can see and edit them. Saving re-mounts AuthProvider so the next
  // request goes to the new host with a fresh bootstrap; Cancel just
  // dismisses without touching anything.
  if (serverSetupOpen) {
    return (
      <SafeAreaProvider>
        <ServerUrlSetupScreen
          initialHostUrl={getCustomHost() ?? ""}
          initialSecret={getGatewaySecret() ?? ""}
          onCancel={closeServerSetup}
          onComplete={() => {
            closeServerSetup();
            setServerSetupBump((n) => n + 1);
            setAuthKey((k) => k + 1);
          }}
        />
        <StatusBar style="auto" />
      </SafeAreaProvider>
    );
  }

  // serverSetupBump is read here so changes to the saved host trigger a
  // re-render of the tree below — useful when the user saves a new URL or
  // resets, since the new value must be picked up immediately.
  void serverSetupBump;

  return (
    <QueryClientProvider client={queryClient}>
      <SafeAreaProvider>
        <ServerSetupContext.Provider value={serverSetupContextValue}>
          <ActiveAuthProvider key={authKey}>
            <AppContent
              onRetry={handleRetry}
              onOpenServerSetup={openServerSetup}
            />
            <StatusBar style="auto" />
          </ActiveAuthProvider>
        </ServerSetupContext.Provider>
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
  secondaryButton: {
    marginTop: 12,
    borderRadius: 8,
    paddingVertical: 12,
    paddingHorizontal: 32,
    borderWidth: 1,
    borderColor: colors.gray300,
  },
  secondaryButtonText: {
    color: colors.gray900,
    fontSize: 15,
    fontWeight: "600",
  },
});
