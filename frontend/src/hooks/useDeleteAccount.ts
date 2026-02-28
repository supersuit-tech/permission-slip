import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDeleteAccount() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { error } = await client.DELETE("/v1/profile", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body: { confirmation: "DELETE" },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to delete account"),
        );
      }
    },
    onSuccess: () => {
      // Clear all cached queries since the account no longer exists.
      queryClient.clear();
    },
  });

  return {
    deleteAccount: () => mutation.mutateAsync(),
    isDeleting: mutation.isPending,
  };
}
