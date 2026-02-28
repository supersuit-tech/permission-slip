import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useUnsubscribePush() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (subscriptionId: number) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/push-subscriptions/{subscription_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { subscription_id: subscriptionId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to delete push subscription"),
        );
      }

      // Also unsubscribe the browser
      try {
        const registration = await navigator.serviceWorker.ready;
        const pushSub = await registration.pushManager.getSubscription();
        if (pushSub) {
          await pushSub.unsubscribe();
        }
      } catch {
        // Browser unsubscribe failure is non-critical
      }

      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["push-subscriptions"] });
    },
  });

  return {
    unsubscribePush: (subscriptionId: number) =>
      mutation.mutateAsync(subscriptionId),
    isLoading: mutation.isPending,
  };
}
