import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type UpdateActionConfigRequest =
  components["schemas"]["UpdateActionConfigRequest"];

interface UpdateActionConfigParams {
  configId: string;
  agentId: number;
  body: UpdateActionConfigRequest;
}

export function useUpdateActionConfig() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ configId, body }: UpdateActionConfigParams) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PUT(
        "/v1/action-configurations/{config_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { config_id: configId } },
          body,
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to update action configuration"),
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
    updateActionConfig: (params: UpdateActionConfigParams) =>
      mutation.mutateAsync(params),
    isPending: mutation.isPending,
  };
}
