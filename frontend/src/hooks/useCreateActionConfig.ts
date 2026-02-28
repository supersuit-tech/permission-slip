import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type CreateActionConfigRequest =
  components["schemas"]["CreateActionConfigRequest"];

export function useCreateActionConfig() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (req: CreateActionConfigRequest) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST("/v1/action-configurations", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body: req,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to create action configuration"),
        );
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["action-configs", variables.agent_id],
      });
    },
  });

  return {
    createActionConfig: (req: CreateActionConfigRequest) =>
      mutation.mutateAsync(req),
    isPending: mutation.isPending,
  };
}
