/**
 * Fetches per-notification-type preferences (shared with web via the same API).
 */
import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type NotificationTypePreference =
  components["schemas"]["NotificationTypePreference"];

export const NOTIFICATION_TYPE_PREFS_KEY =
  "notification-type-preferences" as const;

/** Matches OpenAPI enum and backend `db.NotificationTypeStandingExecution`. */
export const NOTIFICATION_TYPE_STANDING_EXECUTION = "standing_execution" as const;

export function useNotificationTypePreferences() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: [NOTIFICATION_TYPE_PREFS_KEY, userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Not authenticated");
      const { data, error } = await client.GET(
        "/v1/profile/notification-type-preferences",
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            "Unable to load notification type preferences.",
          ),
        );
      }
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    preferences: query.data?.preferences ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? (query.error?.message ??
          "Unable to load notification type preferences.")
      : null,
    refetch: query.refetch,
  };
}
