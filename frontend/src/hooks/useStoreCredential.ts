import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type StoreRequest = components["schemas"]["StoreCredentialRequest"];

export function useStoreCredential() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (body: StoreRequest) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST("/v1/credentials", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to store credential"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["credentials"] });
    },
  });

  return {
    storeCredential: (body: StoreRequest) => mutation.mutateAsync(body),
    isLoading: mutation.isPending,
  };
}
