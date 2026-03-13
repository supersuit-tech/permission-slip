import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDisconnectOAuth() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (connectionId: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/oauth/connections/{connection_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { connection_id: connectionId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to disconnect connection"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["oauth-connections"] });
    },
  });

  return {
    disconnect: (connectionId: string) => mutation.mutateAsync(connectionId),
    isLoading: mutation.isPending,
  };
}
