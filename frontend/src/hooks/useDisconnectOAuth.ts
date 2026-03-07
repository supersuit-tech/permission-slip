import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDisconnectOAuth() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (provider: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/oauth/connections/{provider}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { provider } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to disconnect provider"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["oauth-connections"] });
    },
  });

  return {
    disconnect: (provider: string) => mutation.mutateAsync(provider),
    isLoading: mutation.isPending,
  };
}
