import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";
import { getApiErrorMessage } from "@/api/errors";
import { NOTIFICATION_TYPE_PREFS_QUERY_KEY } from "./useNotificationTypePreferences";

type NotificationTypePreferenceUpdate =
  components["schemas"]["NotificationTypePreferenceUpdate"];

/**
 * Updates per-notification-type preferences. Types not included are unchanged.
 */
export function useUpdateNotificationTypePreferences() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (preferences: NotificationTypePreferenceUpdate[]) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PUT(
        "/v1/profile/notification-type-preferences",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          body: { preferences },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            "Failed to update notification type preferences",
          ),
        );
      }
      return data;
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: [NOTIFICATION_TYPE_PREFS_QUERY_KEY],
      });
    },
  });

  return {
    updatePreferences: mutation.mutateAsync,
    isUpdating: mutation.isPending,
    error: mutation.error,
  };
}
