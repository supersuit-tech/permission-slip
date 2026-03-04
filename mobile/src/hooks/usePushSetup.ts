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
import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNotifications } from "./useNotifications";
import { useRegisterPushToken, unregisterPushTokenDirect } from "./useRegisterPushToken";
import { useAuth } from "../auth/AuthContext";

/** Maximum number of retry attempts for token registration. */
const MAX_RETRIES = 3;

/** Returns the delay in ms for the given retry attempt (exponential backoff). */
function retryDelay(attempt: number): number {
  return Math.min(1000 * Math.pow(2, attempt), 8000);
}

export function usePushSetup() {
  const { authStatus, session } = useAuth();
  const queryClient = useQueryClient();
  const [isTokenRegistered, setIsTokenRegistered] = useState(false);

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

  // Track active retry timeout so we can cancel on unmount or auth change
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
      if (__DEV__) {
        console.log("[push] Authenticated — requesting push permissions");
      }
      registerForPushNotifications();
    }
  }, [authStatus, registerForPushNotifications]);

  // When we have a token and are authenticated, register it with the backend.
  // Retries with exponential backoff on failure (up to MAX_RETRIES attempts).
  useEffect(() => {
    if (authStatus !== "authenticated" || !expoPushToken) return;
    if (registeredTokenRef.current === expoPushToken) return; // already registered

    let cancelled = false;

    async function attemptRegistration(attempt: number) {
      if (cancelled) return;

      try {
        await registerToken(expoPushToken!);
        if (cancelled) return;
        registeredTokenRef.current = expoPushToken;
        setIsTokenRegistered(true);
        if (__DEV__) {
          console.log("[push] Token registered with backend");
        }
      } catch (err) {
        if (cancelled) return;

        if (__DEV__) {
          const msg = err instanceof Error ? err.message : String(err);
          console.warn(
            `[push] Registration attempt ${attempt + 1}/${MAX_RETRIES + 1} failed: ${msg}`,
          );
        }

        if (attempt < MAX_RETRIES) {
          const delay = retryDelay(attempt);
          if (__DEV__) {
            console.log(`[push] Retrying in ${delay}ms...`);
          }
          retryTimerRef.current = setTimeout(
            () => attemptRegistration(attempt + 1),
            delay,
          );
        } else if (__DEV__) {
          console.warn(
            "[push] All registration attempts exhausted. " +
              "Will retry on next app launch or auth change.",
          );
        }
      }
    }

    attemptRegistration(0);

    return () => {
      cancelled = true;
      if (retryTimerRef.current) {
        clearTimeout(retryTimerRef.current);
        retryTimerRef.current = null;
      }
    };
  }, [authStatus, expoPushToken, registerToken]);

  // On sign-out, unregister the token from the backend using the captured
  // access token (since session is already null at this point).
  useEffect(() => {
    if (authStatus === "unauthenticated" && registeredTokenRef.current) {
      const pushToken = registeredTokenRef.current;
      const accessToken = lastAccessTokenRef.current;
      registeredTokenRef.current = null;
      lastAccessTokenRef.current = null;
      setIsTokenRegistered(false);

      // Cancel any in-flight retry
      if (retryTimerRef.current) {
        clearTimeout(retryTimerRef.current);
        retryTimerRef.current = null;
      }

      if (accessToken) {
        if (__DEV__) {
          console.log("[push] Signed out — unregistering token from backend");
        }
        unregisterPushTokenDirect(pushToken, accessToken);
      }
    }
  }, [authStatus]);

  return {
    lastNotificationResponse,
    expoPushToken,
    permissionGranted,
    notificationError,
    /** Whether the push token has been successfully registered with the backend. */
    isTokenRegistered,
  };
}
