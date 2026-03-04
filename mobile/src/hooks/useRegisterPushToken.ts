/**
 * React Query mutation hook for registering and unregistering Expo push tokens
 * with the backend. Used by the notification setup flow and logout.
 */
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";

/**
 * Registers an Expo push token with the backend. If the token is already
 * registered for this user, it is refreshed (upsert).
 *
 * Usage:
 * ```ts
 * const { registerToken, unregisterToken, isRegistering } = useRegisterPushToken();
 * await registerToken("ExponentPushToken[abc123]");
 * ```
 */
export function useRegisterPushToken() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const register = useMutation({
    mutationFn: async (expoToken: string) => {
      const accessToken = session?.access_token;
      if (!accessToken) throw new Error("Not authenticated");

      const { data, error } = await client.POST("/v1/push-subscriptions", {
        headers: { Authorization: `Bearer ${accessToken}` },
        body: { type: "expo", expo_token: expoToken },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to register push token"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["push-subscriptions"] });
    },
  });

  const unregister = useMutation({
    mutationFn: async (expoToken: string) => {
      const accessToken = session?.access_token;
      if (!accessToken) throw new Error("Not authenticated");

      const { data, error } = await client.POST(
        "/v1/push-subscriptions/unregister",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          body: { expo_token: expoToken },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to unregister push token"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["push-subscriptions"] });
    },
  });

  return {
    registerToken: register.mutateAsync,
    unregisterToken: unregister.mutateAsync,
    isRegistering: register.isPending,
    isUnregistering: unregister.isPending,
    registerError: register.error?.message ?? null,
    unregisterError: unregister.error?.message ?? null,
  };
}
