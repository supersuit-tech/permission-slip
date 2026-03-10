import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

/**
 * Converts a base64url-encoded string to a Uint8Array.
 * Used to convert VAPID public key for the PushManager.subscribe() call.
 */
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, "+")
    .replace(/_/g, "/");

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/**
 * Translates browser PushManager.subscribe() errors into user-friendly messages.
 * The browser throws DOMExceptions with technical names that mean nothing to users.
 */
function getPushSubscribeErrorMessage(err: unknown): string {
  if (!(err instanceof DOMException)) {
    return "Something went wrong enabling notifications. Please try again.";
  }

  if (err.name === "NotAllowedError") {
    return "Notification permission was denied. Please allow notifications in your browser settings and try again.";
  }

  if (err.name === "InvalidStateError") {
    return "There is a conflicting notification subscription. Try clearing your browser's site data and re-enabling notifications.";
  }

  if (err.name === "AbortError" || err.message.includes("push service")) {
    return "Could not reach the push notification service. Check your internet connection and try again.";
  }

  return "Something went wrong enabling notifications. Please try again.";
}

export function useSubscribePush() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (vapidKey: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      // Request notification permission
      const permission = await Notification.requestPermission();
      if (permission !== "granted") {
        throw new Error("Notification permission denied");
      }

      // Register service worker
      const registration = await navigator.serviceWorker.register("/sw.js");
      await navigator.serviceWorker.ready;

      // Subscribe to push
      const serverKey = urlBase64ToUint8Array(vapidKey);
      let pushSubscription: PushSubscription;
      try {
        pushSubscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          // Safe: urlBase64ToUint8Array creates a fresh Uint8Array (always backed by ArrayBuffer, not SharedArrayBuffer)
          applicationServerKey: serverKey.buffer as ArrayBuffer,
        });
      } catch (err) {
        throw new Error(getPushSubscribeErrorMessage(err));
      }

      const subJSON = pushSubscription.toJSON();
      const endpoint = subJSON.endpoint;
      const p256dh = subJSON.keys?.p256dh;
      const auth = subJSON.keys?.auth;

      if (!endpoint || !p256dh || !auth) {
        throw new Error("Invalid push subscription from browser");
      }

      // Send subscription to server
      const { data, error } = await client.POST("/v1/push-subscriptions", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body: { type: "web-push", endpoint, p256dh, auth },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to register push subscription"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["push-subscriptions"] });
    },
  });

  return {
    subscribePush: (vapidKey: string) => mutation.mutateAsync(vapidKey),
    isLoading: mutation.isPending,
    error: mutation.error?.message ?? null,
  };
}
