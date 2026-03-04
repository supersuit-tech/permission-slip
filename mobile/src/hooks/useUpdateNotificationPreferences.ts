/**
 * React Query mutation hook for updating notification channel preferences.
 * Accepts an array of channel/enabled pairs — channels not included are
 * left unchanged. Invalidates the preferences cache on success.
 */
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

type NotificationPreferenceUpdate =
  components["schemas"]["NotificationPreferenceUpdate"];

export function useUpdateNotificationPreferences() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (preferences: NotificationPreferenceUpdate[]) => {
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
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to update notification preferences."),
        );
      }
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
    isUpdating: mutation.isPending,
    error: mutation.error,
  };
}
