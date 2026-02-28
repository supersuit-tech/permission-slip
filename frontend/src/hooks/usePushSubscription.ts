/**
 * Returns the current Notification.permission state, handling the case
 * where Notification API is not available (e.g. insecure context).
 */
export function getPermissionState(): NotificationPermission | "unsupported" {
  if (typeof Notification === "undefined") return "unsupported";
  return Notification.permission;
}

/**
 * Returns true if the browser supports Web Push notifications.
 */
export function isPushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    typeof Notification !== "undefined"
  );
}
