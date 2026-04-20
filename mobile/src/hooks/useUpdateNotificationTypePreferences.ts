/**
 * Updates per-notification-type preferences; invalidates the type-preferences cache.
 */
import { useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";
import { NOTIFICATION_TYPE_PREFS_KEY } from "./useNotificationTypePreferences";

type NotificationTypePreferenceUpdate =
  components["schemas"]["NotificationTypePreferenceUpdate"];

export function useUpdateNotificationTypePreferences() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id;

  const tokenRef = useRef(session?.access_token);
  tokenRef.current = session?.access_token;

  const mutation = useMutation({
    mutationFn: async (preferences: NotificationTypePreferenceUpdate[]) => {
      const token = tokenRef.current;
      if (!token) {
        throw new Error("Not authenticated");
      }
      const { data, error } = await client.PUT(
        "/v1/profile/notification-type-preferences",
        {
          headers: { Authorization: `Bearer ${token}` },
          body: { preferences },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            "Failed to update notification type preferences.",
          ),
        );
      }
      return data;
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: [NOTIFICATION_TYPE_PREFS_KEY],
      });
    },
  });

  return {
    updatePreferences: mutation.mutateAsync,
    isUpdating: mutation.isPending,
    error: mutation.error,
  };
}
