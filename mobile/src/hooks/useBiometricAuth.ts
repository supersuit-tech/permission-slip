/**
 * Hook for biometric authentication (FaceID / TouchID / Fingerprint).
 *
 * Provides:
 * - Hardware capability detection
 * - Enrolled biometrics check
 * - User preference toggle (persisted in secure storage, scoped per user)
 * - Authentication prompt
 *
 * Biometric auth is an optional per-user setting — when enabled, it gates
 * access to the app on resume from background. The preference is stored
 * with a user-specific key so multiple accounts on a shared device don't
 * interfere with each other.
 */
import { useCallback, useEffect, useState } from "react";
import * as LocalAuthentication from "expo-local-authentication";
import * as SecureStore from "expo-secure-store";

const BIOMETRIC_KEY_PREFIX = "biometric_auth_enabled";

/** Returns the SecureStore key for the given user's biometric preference. */
function biometricKey(userId: string | undefined): string {
  // Include user ID to prevent cross-account preference leakage on shared
  // devices. Falls back to a global key if user ID is unavailable (shouldn't
  // happen in practice since the hook is only used when authenticated).
  return userId ? `${BIOMETRIC_KEY_PREFIX}_${userId}` : BIOMETRIC_KEY_PREFIX;
}

export type BiometricStatus =
  | "checking"
  | "unavailable"
  | "available"
  | "enrolled";

interface UseBiometricAuthOptions {
  /** The current user's ID, used to scope biometric preferences per user. */
  userId?: string;
}

export function useBiometricAuth(options: UseBiometricAuthOptions = {}) {
  const { userId } = options;
  const [status, setStatus] = useState<BiometricStatus>("checking");
  const [isEnabled, setIsEnabled] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Check hardware and enrollment on mount (and when userId changes)
  useEffect(() => {
    async function check() {
      const hasHardware = await LocalAuthentication.hasHardwareAsync();
      if (!hasHardware) {
        setStatus("unavailable");
        return;
      }

      const isEnrolled = await LocalAuthentication.isEnrolledAsync();
      if (!isEnrolled) {
        setStatus("available");
        return;
      }

      setStatus("enrolled");

      // Load user-specific preference
      const key = biometricKey(userId);
      const stored = await SecureStore.getItemAsync(key);
      if (stored === "true") {
        setIsEnabled(true);
      } else {
        setIsEnabled(false);
      }
    }

    check();
  }, [userId]);

  const toggleBiometric = useCallback(async (enabled: boolean) => {
    if (enabled) {
      // Verify the user can actually authenticate before enabling
      const result = await LocalAuthentication.authenticateAsync({
        promptMessage: "Verify your identity to enable biometric lock",
        cancelLabel: "Cancel",
        disableDeviceFallback: false,
      });

      if (!result.success) return false;
    }

    const key = biometricKey(userId);
    await SecureStore.setItemAsync(key, String(enabled));
    setIsEnabled(enabled);
    if (!enabled) {
      setIsAuthenticated(true);
    }
    return true;
  }, [userId]);

  const authenticate = useCallback(async () => {
    if (!isEnabled) {
      setIsAuthenticated(true);
      return true;
    }

    const result = await LocalAuthentication.authenticateAsync({
      promptMessage: "Unlock Permission Slip",
      cancelLabel: "Cancel",
      disableDeviceFallback: false,
    });

    setIsAuthenticated(result.success);
    return result.success;
  }, [isEnabled]);

  // Skip biometric gate if not enabled
  useEffect(() => {
    if (!isEnabled) {
      setIsAuthenticated(true);
    }
  }, [isEnabled]);

  return {
    /** Whether biometric hardware is available and enrolled. */
    status,
    /** Whether the user has opted in to biometric auth. */
    isEnabled,
    /** Whether the user has passed biometric auth this session. */
    isAuthenticated,
    /** Mark as authenticated (e.g. after returning from background). */
    setIsAuthenticated,
    /** Toggle biometric auth on/off. Returns false if verification failed. */
    toggleBiometric,
    /** Prompt the user for biometric authentication. */
    authenticate,
  };
}
