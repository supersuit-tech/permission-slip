import { Bell, Loader2, AlertTriangle, Lock } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { useProfile } from "@/hooks/useProfile";
import { useUpdateProfile } from "@/hooks/useUpdateProfile";
import { trackEvent, PostHogEvents } from "@/lib/posthog";
import { useNotificationPreferences } from "@/hooks/useNotificationPreferences";
import { useUpdateNotificationPreferences } from "@/hooks/useUpdateNotificationPreferences";
import {
  NOTIFICATION_TYPE_STANDING_EXECUTION,
  useNotificationTypePreferences,
} from "@/hooks/useNotificationTypePreferences";
import { useUpdateNotificationTypePreferences } from "@/hooks/useUpdateNotificationTypePreferences";
import type { components } from "@/api/schema";
import { Switch } from "@/components/ui/switch";
import { NotifyAboutAutoApprovalsRow } from "./NotifyAboutAutoApprovalsRow";
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
  sms: {
    name: "SMS",
    description: "Text message notifications for urgent approval requests.",
  },
  "mobile-push": {
    name: "Mobile Push",
    description:
      "Push notifications to your mobile device for real-time approval alerts.",
  },
};

export function NotificationSection() {
  const { profile } = useProfile();
  const { updateProfile, isLoading: isUpdatingProfile } = useUpdateProfile();
  const { preferences, isLoading, error } = useNotificationPreferences();
  const {
    preferences: typePreferences,
    isLoading: isLoadingTypePrefs,
    error: typePrefsError,
  } = useNotificationTypePreferences();
  const { updatePreferences, isLoading: isUpdating } =
    useUpdateNotificationPreferences();
  const {
    updatePreferences: updateTypePreferences,
    isUpdating: isUpdatingTypePrefs,
  } = useUpdateNotificationTypePreferences();

  const standingExecutionPref = typePreferences.find(
    (p) => p.notification_type === NOTIFICATION_TYPE_STANDING_EXECUTION,
  );
  const standingExecutionEnabled = standingExecutionPref?.enabled ?? true;

  // Determine which channels are missing required contact info.
  const missingContact: Record<string, string> = {};
  if (!profile?.email) {
    missingContact["email"] = "Add a contact email above to receive email notifications.";
  }
  if (!profile?.phone) {
    missingContact["sms"] = "Add a phone number above to receive SMS notifications.";
  }

  async function handleToggleProductUpdates() {
    if (!profile) return;
    const newValue = !profile.marketing_opt_in;
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

  async function handleToggleStandingExecution(enabled: boolean) {
    try {
      await updateTypePreferences([
        {
          notification_type: NOTIFICATION_TYPE_STANDING_EXECUTION,
          enabled,
        },
      ]);
      toast.success(
        `Auto-approval execution notifications ${enabled ? "enabled" : "silenced"}.`,
      );
    } catch {
      toast.error("Failed to update notification preference.");
    }
  }

  async function handleToggle(channel: string, currentEnabled: boolean) {
    try {
      await updatePreferences([
        { channel: channel as components["schemas"]["NotificationPreferenceUpdate"]["channel"], enabled: !currentEnabled },
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
        {isLoading || isLoadingTypePrefs ? (
          <div
            className="flex items-center justify-center py-8"
            role="status"
            aria-label="Loading notification preferences"
          >
            <Loader2 className="text-muted-foreground size-5 animate-spin" />
          </div>
        ) : error || typePrefsError ? (
          <p className="text-destructive text-sm">
            {error ?? typePrefsError}
          </p>
        ) : (
          <div className="space-y-4">
            {preferences
              .filter((pref) => pref.channel !== "web-push")
              .map((pref) => {
              const label = CHANNEL_LABELS[pref.channel];
              const warning = missingContact[pref.channel];
              const planGated = pref.available === false;
              return (
                <div
                  key={pref.channel}
                  className={cn(
                    "rounded-lg border p-4",
                    planGated && "border-dashed opacity-75",
                  )}
                >
                  <div className="flex items-center justify-between">
                    <div className="space-y-0.5">
                      <p className="text-sm font-medium">
                        {planGated && (
                          <Lock className="mr-1.5 inline size-3.5 text-muted-foreground" />
                        )}
                        {label?.name ?? pref.channel}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {label?.description ?? ""}
                      </p>
                    </div>
                    {planGated ? (
                      <span className="inline-flex items-center whitespace-nowrap rounded-md border border-muted bg-muted/50 px-2.5 py-1.5 text-xs font-medium text-muted-foreground">
                        Coming soon
                      </span>
                    ) : (
                      <Switch
                        checked={pref.enabled}
                        disabled={isUpdating}
                        aria-label={`${label?.name ?? pref.channel} notifications`}
                        onCheckedChange={(checked) => handleToggle(pref.channel, !checked)}
                      />
                    )}
                  </div>
                  {!planGated && warning && pref.enabled && (
                    <div className="mt-2 flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400">
                      <AlertTriangle className="size-3.5 shrink-0" />
                      <span>{warning}</span>
                    </div>
                  )}
                </div>
              );
            })}

            <div className="space-y-2">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Notify me about
              </p>
              <NotifyAboutAutoApprovalsRow
                enabled={standingExecutionEnabled}
                disabled={isUpdating || isUpdatingTypePrefs}
                onCheckedChange={(checked) =>
                  void handleToggleStandingExecution(checked)
                }
              />
            </div>

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
                <Switch
                  checked={profile?.marketing_opt_in ?? false}
                  disabled={isUpdatingProfile || !profile}
                  aria-label="Product updates notifications"
                  onCheckedChange={handleToggleProductUpdates}
                />
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
