import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

interface EnableArgs {
  agentId: number;
  connectorId: string;
}

export function useEnableAgentConnector() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ agentId, connectorId }: EnableArgs) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PUT(
        "/v1/agents/{agent_id}/connectors/{connector_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: {
            path: { agent_id: agentId, connector_id: connectorId },
          },
        },
      );
      if (error) throw new Error("Failed to enable connector");
      return data;
    },
    onSuccess: (_data, { agentId }) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-connectors", agentId],
      });
    },
  });

  return {
    enableConnector: (args: EnableArgs) => mutation.mutateAsync(args),
    isLoading: mutation.isPending,
  };
}
