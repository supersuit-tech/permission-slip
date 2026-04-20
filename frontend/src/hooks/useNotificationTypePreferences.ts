import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type NotificationTypePreference =
  components["schemas"]["NotificationTypePreference"];

export const NOTIFICATION_TYPE_PREFS_QUERY_KEY = "notification-type-preferences" as const;

/**
 * Fetches per-notification-type preferences (e.g. auto-approval execution notifications).
 * Missing types default to enabled on the server.
 */
export function useNotificationTypePreferences() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const { data, isLoading, error } = useQuery({
    queryKey: [NOTIFICATION_TYPE_PREFS_QUERY_KEY],
    queryFn: async () => {
      if (!accessToken) throw new Error("Not authenticated");
      const { data, error } = await client.GET(
        "/v1/profile/notification-type-preferences",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
        },
      );
      if (error) throw new Error("Failed to load notification type preferences");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    preferences: data?.preferences ?? [],
    isLoading,
    error: error?.message ?? null,
  };
}
