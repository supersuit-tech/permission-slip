import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

interface DeleteActionConfigParams {
  configId: string;
  agentId: number;
}

export function useDeleteActionConfig() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ configId }: DeleteActionConfigParams) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/action-configurations/{config_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { config_id: configId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to delete action configuration"),
        );
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["action-configs", variables.agentId],
      });
    },
  });

  return {
    deleteActionConfig: (params: DeleteActionConfigParams) =>
      mutation.mutateAsync(params),
    isPending: mutation.isPending,
  };
}
