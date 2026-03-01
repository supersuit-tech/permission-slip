import { Bell, Loader2, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { useProfile } from "@/hooks/useProfile";
import { useUpdateProfile } from "@/hooks/useUpdateProfile";
import { trackEvent, PostHogEvents } from "@/lib/posthog";
import { useNotificationPreferences } from "@/hooks/useNotificationPreferences";
import { useUpdateNotificationPreferences } from "@/hooks/useUpdateNotificationPreferences";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const CHANNEL_LABELS: Record<string, { name: string; description: string }> = {
  email: {
    name: "Email",
    description: "Receive notifications via email when actions need approval.",
  },
  "web-push": {
    name: "Web Push",
    description:
      "Browser push notifications for real-time approval alerts.",
  },
  sms: {
    name: "SMS",
    description: "Text message notifications for urgent approval requests.",
  },
};

export function NotificationSection() {
  const { profile } = useProfile();
  const { updateProfile, isLoading: isUpdatingProfile } = useUpdateProfile();
  const { preferences, isLoading, error } = useNotificationPreferences();
  const { updatePreferences, isLoading: isUpdating } =
    useUpdateNotificationPreferences();

  // Determine which channels are missing required contact info.
  const missingContact: Record<string, string> = {};
  if (!profile?.email) {
    missingContact["email"] = "Add a contact email above to receive email notifications.";
  }
  if (!profile?.phone) {
    missingContact["sms"] = "Add a phone number above to receive SMS notifications.";
  }

  async function handleToggleProductUpdates() {
    const newValue = !profile?.marketing_opt_in;
    try {
      await updateProfile({ marketing_opt_in: newValue });
      trackEvent(PostHogEvents.MARKETING_OPT_IN_UPDATED, { enabled: newValue });
      toast.success(
        `Product updates ${newValue ? "enabled" : "disabled"}.`,
      );
    } catch {
      toast.error("Failed to update product updates preference.");
    }
  }

  async function handleToggle(channel: string, currentEnabled: boolean) {
    try {
      await updatePreferences([
        { channel: channel as "email" | "web-push" | "sms", enabled: !currentEnabled },
      ]);
      toast.success(
        `${CHANNEL_LABELS[channel]?.name ?? channel} notifications ${!currentEnabled ? "enabled" : "disabled"}.`,
      );
    } catch {
      toast.error("Failed to update notification preference.");
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Bell className="text-muted-foreground size-5" />
          <CardTitle>Notifications</CardTitle>
        </div>
        <CardDescription>
          Choose how you want to be notified about approval requests and agent
          activity.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading notification preferences"
          >
            <Loader2 className="text-muted-foreground size-5 animate-spin" />
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">{error}</p>
        ) : (
          <div className="space-y-4">
            {preferences.map((pref) => {
              const label = CHANNEL_LABELS[pref.channel];
              const warning = missingContact[pref.channel];
              return (
                <div
                  key={pref.channel}
                  className="rounded-lg border p-4"
                >
                  <div className="flex items-center justify-between">
                    <div className="space-y-0.5">
                      <p className="text-sm font-medium">
                        {label?.name ?? pref.channel}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {label?.description ?? ""}
                      </p>
                    </div>
                    <Button
                      variant={pref.enabled ? "default" : "outline"}
                      size="sm"
                      disabled={isUpdating}
                      onClick={() => handleToggle(pref.channel, pref.enabled)}
                    >
                      {pref.enabled ? "Enabled" : "Disabled"}
                    </Button>
                  </div>
                  {warning && pref.enabled && (
                    <div className="mt-2 flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
                      <AlertTriangle className="size-3.5 shrink-0" />
                      <span>{warning}</span>
                    </div>
                  )}
                </div>
              );
            })}

            <hr className="border-border" />

            <div className="rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-sm font-medium">Product updates</p>
                  <p className="text-xs text-muted-foreground">
                    Occasional emails about new features, platform improvements,
                    and tips.
                  </p>
                </div>
                <Button
                  variant={profile?.marketing_opt_in ? "default" : "outline"}
                  size="sm"
                  disabled={isUpdatingProfile}
                  onClick={handleToggleProductUpdates}
                >
                  {profile?.marketing_opt_in ? "Enabled" : "Disabled"}
                </Button>
              </div>
              {!profile?.email && profile?.marketing_opt_in && (
                <div className="mt-2 flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
                  <AlertTriangle className="size-3.5 shrink-0" />
                  <span>Add a contact email above to receive product update emails.</span>
                </div>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
