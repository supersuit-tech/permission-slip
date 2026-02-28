import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type NotificationPreference = components["schemas"]["NotificationPreference"];

/**
 * Fetches the current user's notification preferences for all channels
 * (email, web-push, sms). Channels without explicit preferences default
 * to enabled on the server side.
 */
export function useNotificationPreferences() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const { data, isLoading, error } = useQuery({
    queryKey: ["notification-preferences"],
    queryFn: async () => {
      if (!accessToken) throw new Error("Not authenticated");
      const { data, error } = await client.GET(
        "/v1/profile/notification-preferences",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
        },
      );
      if (error) throw new Error("Failed to load notification preferences");
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
