// Service Worker for Web Push notifications.
// Handles incoming push events and notification click actions.

self.addEventListener("push", (event) => {
  if (!event.data) return;

  let data;
  try {
    data = event.data.json();
  } catch {
    // Not JSON — show raw text
    data = { title: "Permission Slip", body: event.data.text() };
  }

  const title = data.title || "Permission Slip";
  const options = {
    body: data.body || "Action requires your approval",
    icon: "/favicon.ico",
    badge: "/favicon.ico",
    tag: data.approval_id || "ps-notification",
    data: {
      url: data.url || "/",
      approval_id: data.approval_id,
    },
    // Require user interaction — don't auto-dismiss
    requireInteraction: true,
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  // Only navigate to same-origin paths to prevent open-redirect via crafted payloads.
  const rawUrl = event.notification.data?.url || "/";
  let url = "/";
  if (rawUrl.startsWith("/") && !rawUrl.startsWith("//")) {
    // Relative path — safe to use directly.
    url = rawUrl;
  } else {
    // Absolute URL — verify it's same-origin before using the path.
    try {
      const parsed = new URL(rawUrl);
      if (parsed.origin === self.location.origin) {
        url = parsed.pathname + parsed.search + parsed.hash;
      }
    } catch {
      // Malformed URL — fall back to dashboard.
    }
  }

  event.waitUntil(
    clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      // Focus an existing tab if one is open on our origin
      for (const client of windowClients) {
        if (new URL(client.url).origin === self.location.origin && "focus" in client) {
          client.focus();
          client.navigate(url);
          return;
        }
      }
      // Otherwise open a new window
      return clients.openWindow(url);
    })
  );
});
