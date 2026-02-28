import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDeleteCredential() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (credentialId: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/credentials/{credential_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { credential_id: credentialId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to delete credential"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["credentials"] });
    },
  });

  return {
    deleteCredential: (credentialId: string) =>
      mutation.mutateAsync(credentialId),
    isLoading: mutation.isPending,
  };
}
