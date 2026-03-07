import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

interface SaveProviderConfigParams {
  provider: string;
  clientId: string;
  clientSecret: string;
}

export function useSaveOAuthProviderConfig() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      provider,
      clientId,
      clientSecret,
    }: SaveProviderConfigParams) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/oauth/provider-configs",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          body: {
            provider,
            client_id: clientId,
            client_secret: clientSecret,
          },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to save OAuth provider config"),
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
    save: (params: SaveProviderConfigParams) => mutation.mutateAsync(params),
    isLoading: mutation.isPending,
  };
}
