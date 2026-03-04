/**
 * React Query mutation hook for updating notification channel preferences.
 * Accepts an array of channel/enabled pairs — channels not included are
 * left unchanged. Invalidates the preferences cache on success.
 */
import { useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

type NotificationPreference = components["schemas"]["NotificationPreference"];
type NotificationPreferenceUpdate =
  components["schemas"]["NotificationPreferenceUpdate"];
type PreferencesResponse =
  components["schemas"]["NotificationPreferencesResponse"];

export function useUpdateNotificationPreferences() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id;

  // Use a ref so the mutation always reads the latest token, even if the
  // session refreshed between render and mutation execution.
  const tokenRef = useRef(session?.access_token);
  tokenRef.current = session?.access_token;

  const mutation = useMutation({
    mutationFn: async (preferences: NotificationPreferenceUpdate[]) => {
      const token = tokenRef.current;
      if (!token) {
        throw new Error("Not authenticated");
      }
      const { data, error } = await client.PUT(
        "/v1/profile/notification-preferences",
        {
          headers: { Authorization: `Bearer ${token}` },
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
    onMutate: async (updates) => {
      const queryKey = ["notification-preferences", userId ?? ""];
      await queryClient.cancelQueries({ queryKey });
      const previous =
        queryClient.getQueryData<PreferencesResponse>(queryKey);

      queryClient.setQueryData<PreferencesResponse>(queryKey, (old) => {
        if (!old) return old;
        const updated = old.preferences.map(
          (p: NotificationPreference): NotificationPreference => {
            const match = updates.find((u) => u.channel === p.channel);
            return match ? { ...p, enabled: match.enabled } : p;
          },
        );
        return { ...old, preferences: updated };
      });

      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        const queryKey = ["notification-preferences", userId ?? ""];
        queryClient.setQueryData(queryKey, context.previous);
      }
    },
    onSettled: () => {
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
