import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type UpdateRequest = components["schemas"]["UpdateCredentialRequest"];

export function useUpdateCredential() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      credentialId,
      body,
    }: {
      credentialId: string;
      body: UpdateRequest;
    }) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PATCH(
        "/v1/credentials/{credential_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { credential_id: credentialId } },
          body,
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to update credential"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["credentials"] });
    },
  });

  return {
    updateCredential: (credentialId: string, body: UpdateRequest) =>
      mutation.mutateAsync({ credentialId, body }),
    isLoading: mutation.isPending,
  };
}
