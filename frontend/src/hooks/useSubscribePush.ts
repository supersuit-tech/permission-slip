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
      const pushSubscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        // Cast is safe: Uint8Array.buffer is always an ArrayBuffer (never SharedArrayBuffer)
        applicationServerKey: serverKey.buffer as ArrayBuffer,
      });

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
        body: { endpoint, p256dh, auth },
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
