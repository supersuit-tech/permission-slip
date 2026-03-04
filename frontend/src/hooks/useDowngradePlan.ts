import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDowngradePlan() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST("/v1/billing/downgrade", {
        headers: { Authorization: `Bearer ${session.access_token}` },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to downgrade plan"),
        );
      }
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing"] });
    },
  });

  return {
    downgrade: () => mutation.mutateAsync(),
    isDowngrading: mutation.isPending,
    error: mutation.error?.message ?? null,
  };
}
