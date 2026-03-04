/**
 * Hook for biometric authentication (FaceID / TouchID / Fingerprint).
 *
 * Provides:
 * - Hardware capability detection
 * - Enrolled biometrics check
 * - User preference toggle (persisted in secure storage)
 * - Authentication prompt
 *
 * Biometric auth is an optional per-user setting — when enabled, it gates
 * access to the app on resume from background.
 */
import { useCallback, useEffect, useState } from "react";
import * as LocalAuthentication from "expo-local-authentication";
import * as SecureStore from "expo-secure-store";

const BIOMETRIC_ENABLED_KEY = "biometric_auth_enabled";

export type BiometricStatus =
  | "checking"
  | "unavailable"
  | "available"
  | "enrolled";

export function useBiometricAuth() {
  const [status, setStatus] = useState<BiometricStatus>("checking");
  const [isEnabled, setIsEnabled] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  // Check hardware and enrollment on mount
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

      // Load user preference
      const stored = await SecureStore.getItemAsync(BIOMETRIC_ENABLED_KEY);
      if (stored === "true") {
        setIsEnabled(true);
      }
    }

    check();
  }, []);

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

    await SecureStore.setItemAsync(BIOMETRIC_ENABLED_KEY, String(enabled));
    setIsEnabled(enabled);
    if (!enabled) {
      setIsAuthenticated(true);
    }
    return true;
  }, []);

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
