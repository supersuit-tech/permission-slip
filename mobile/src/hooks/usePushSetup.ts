/**
 * Orchestrates push notification setup: requests permissions, retrieves the
 * Expo push token, and registers/unregisters it with the backend as the
 * user's auth status changes.
 *
 * Should be called once in the authenticated app shell (e.g. AppContent).
 */
import { useEffect, useRef } from "react";
import { useNotifications } from "./useNotifications";
import { useRegisterPushToken } from "./useRegisterPushToken";
import { useAuth } from "../auth/AuthContext";

export function usePushSetup() {
  const { authStatus } = useAuth();
  const { expoPushToken, registerForPushNotifications, lastNotificationResponse } =
    useNotifications();
  const { registerToken, unregisterToken } = useRegisterPushToken();

  // Track the token we last sent to the backend so we can unregister on logout
  const registeredTokenRef = useRef<string | null>(null);

  // When authenticated, request permissions and get the push token
  useEffect(() => {
    if (authStatus === "authenticated") {
      registerForPushNotifications();
    }
  }, [authStatus, registerForPushNotifications]);

  // When we have a token and are authenticated, register it with the backend
  useEffect(() => {
    if (authStatus !== "authenticated" || !expoPushToken) return;
    if (registeredTokenRef.current === expoPushToken) return; // already registered

    registerToken(expoPushToken)
      .then(() => {
        registeredTokenRef.current = expoPushToken;
      })
      .catch(() => {
        // Registration is best-effort; the user can still use the app.
        // The hook will retry on next mount or auth change.
      });
  }, [authStatus, expoPushToken, registerToken]);

  // On sign-out, unregister the token from the backend
  useEffect(() => {
    if (authStatus === "unauthenticated" && registeredTokenRef.current) {
      const token = registeredTokenRef.current;
      registeredTokenRef.current = null;
      unregisterToken(token).catch(() => {
        // Best-effort cleanup. The backend will eventually prune stale tokens.
      });
    }
  }, [authStatus, unregisterToken]);

  return { lastNotificationResponse };
}
