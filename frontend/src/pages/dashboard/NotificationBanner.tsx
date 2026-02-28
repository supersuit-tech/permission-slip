import { useState } from "react";
import { Bell, X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useVAPIDKey } from "@/hooks/useVAPIDKey";
import { usePushSubscriptions } from "@/hooks/usePushSubscriptions";
import { useSubscribePush } from "@/hooks/useSubscribePush";
import {
  isPushSupported,
  getPermissionState,
} from "@/hooks/usePushSubscription";

/**
 * Non-intrusive banner shown on the dashboard when push notifications
 * are available but not yet enabled. Explains the benefit before
 * triggering the browser permission prompt.
 */
export function NotificationBanner() {
  const { vapidKey, isLoading: vapidLoading } = useVAPIDKey();
  const { subscriptions, isLoading: subsLoading } = usePushSubscriptions();
  const {
    subscribePush,
    isLoading: isSubscribing,
    error: subscribeError,
  } = useSubscribePush();
  const [dismissed, setDismissed] = useState(false);

  const isSupported = isPushSupported();
  const permissionState = getPermissionState();
  const isLoading = vapidLoading || subsLoading;

  // Don't show if:
  // - Push not supported in this browser
  // - Already has subscriptions
  // - Permission was denied (can't ask again)
  // - Banner was dismissed this session
  // - Still loading
  if (
    !isSupported ||
    isLoading ||
    dismissed ||
    permissionState === "denied" ||
    subscriptions.length > 0
  ) {
    return null;
  }

  const handleEnable = async () => {
    if (!vapidKey) return;
    try {
      await subscribePush(vapidKey);
    } catch {
      // Error is displayed via subscribeError
    }
  };

  return (
    <div className="bg-muted/50 relative flex items-start gap-3 rounded-lg border p-4">
      <div className="bg-primary/10 shrink-0 rounded-full p-2">
        <Bell className="text-primary size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">Enable push notifications</p>
        <p className="text-muted-foreground mt-0.5 text-xs">
          Get instant browser notifications when agents request approval — even
          when this tab is closed.
        </p>
        {subscribeError && (
          <p className="mt-1 text-xs text-red-600">{subscribeError}</p>
        )}
        <Button
          size="sm"
          className="mt-2"
          onClick={handleEnable}
          disabled={isSubscribing || !vapidKey}
        >
          {isSubscribing && <Loader2 className="mr-1.5 size-3 animate-spin" />}
          Enable notifications
        </Button>
      </div>
      <button
        onClick={() => setDismissed(true)}
        className="text-muted-foreground hover:text-foreground shrink-0 p-1"
        aria-label="Dismiss notification banner"
      >
        <X className="size-4" />
      </button>
    </div>
  );
}
