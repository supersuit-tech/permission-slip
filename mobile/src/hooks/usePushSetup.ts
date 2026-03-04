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
import client from "../api/client";

export function usePushSetup() {
  const { authStatus, session } = useAuth();
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
  const { registerToken } = useRegisterPushToken();

  // Track the token we last sent to the backend so we can unregister on logout
  const registeredTokenRef = useRef<string | null>(null);

  // Capture the access token while authenticated so we can use it for
  // unregistration after sign-out. By the time authStatus transitions to
  // "unauthenticated", the session is already null — the mutation hook's
  // session reference would be stale and fail with "Not authenticated".
  const lastAccessTokenRef = useRef<string | null>(null);
  useEffect(() => {
    if (session?.access_token) {
      lastAccessTokenRef.current = session.access_token;
    }
  }, [session?.access_token]);

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

  // On sign-out, unregister the token from the backend using the captured
  // access token (since session is already null at this point).
  useEffect(() => {
    if (authStatus === "unauthenticated" && registeredTokenRef.current) {
      const pushToken = registeredTokenRef.current;
      const accessToken = lastAccessTokenRef.current;
      registeredTokenRef.current = null;
      lastAccessTokenRef.current = null;

      if (accessToken) {
        client
          .POST("/v1/push-subscriptions/unregister", {
            headers: { Authorization: `Bearer ${accessToken}` },
            body: { expo_token: pushToken },
          })
          .catch(() => {
            // Best-effort cleanup. The backend will eventually prune stale tokens.
          });
      }
    }
  }, [authStatus]);

  return {
    lastNotificationResponse,
    expoPushToken,
    permissionGranted,
    notificationError,
  };
}
