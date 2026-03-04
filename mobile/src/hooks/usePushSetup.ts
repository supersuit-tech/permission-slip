/**
 * Orchestrates push notification setup: requests permissions, retrieves the
 * Expo push token, and registers/unregisters it with the backend as the
 * user's auth status changes.
 *
 * Also refreshes the approvals cache when a push notification is received
 * while the app is in the foreground, so the user immediately sees new
 * approval requests without waiting for the next poll.
 *
 * Should be called once in the authenticated app shell (e.g. AppContent).
 */
import { useCallback, useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNotifications } from "./useNotifications";
import { useRegisterPushToken } from "./useRegisterPushToken";
import { useAuth } from "../auth/AuthContext";

export function usePushSetup() {
  const { authStatus } = useAuth();
  const queryClient = useQueryClient();

  // When a notification arrives in the foreground, invalidate the approvals
  // cache so the list updates immediately without waiting for the next poll.
  const onNotificationReceived = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["approvals"] });
  }, [queryClient]);

  const {
    expoPushToken,
    permissionGranted,
    error: notificationError,
    registerForPushNotifications,
    lastNotificationResponse,
  } = useNotifications({ onNotificationReceived });
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

  return {
    lastNotificationResponse,
    expoPushToken,
    permissionGranted,
    notificationError,
  };
}
