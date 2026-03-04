/**
 * React Query hook for fetching the authenticated user's notification
 * preferences. Mirrors the web frontend's hook but adapted for mobile
 * auth patterns (ref-based token to avoid query key churn).
 */
import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type NotificationPreference =
  components["schemas"]["NotificationPreference"];

/** Shared query key prefix — used by the update hook for cache invalidation. */
export const NOTIFICATION_PREFS_KEY = "notification-preferences" as const;

export function useNotificationPreferences() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: [NOTIFICATION_PREFS_KEY, userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Not authenticated");
      const { data, error } = await client.GET(
        "/v1/profile/notification-preferences",
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            "Unable to load notification preferences.",
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
          "Unable to load notification preferences.")
      : null,
    refetch: query.refetch,
  };
}
