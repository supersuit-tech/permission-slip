import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDeleteOAuthProviderConfig() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (provider: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/oauth/provider-configs/{provider}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { provider } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(
            error,
            "Failed to delete OAuth provider config",
          ),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["oauth-provider-configs"] });
      queryClient.invalidateQueries({ queryKey: ["oauth-providers"] });
    },
  });

  return {
    deleteConfig: (provider: string) => mutation.mutateAsync(provider),
    isLoading: mutation.isPending,
  };
}
