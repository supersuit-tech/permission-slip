import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

type NotificationPreference = components["schemas"]["NotificationPreference"];

/**
 * Hook for toggling notification channel preferences. Accepts an array
 * of channel/enabled pairs — channels not included are left unchanged.
 * Invalidates the preferences cache on success.
 */
export function useUpdateNotificationPreferences() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (preferences: NotificationPreference[]) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PUT(
        "/v1/profile/notification-preferences",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          body: { preferences },
        },
      );
      if (error) throw new Error("Failed to update notification preferences");
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["notification-preferences"],
      });
    },
  });

  return {
    updatePreferences: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error,
  };
}
