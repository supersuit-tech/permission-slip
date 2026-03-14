import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

type DisableResponse =
  components["schemas"]["DisableAgentConnectorResponse"];

interface DisableArgs {
  agentId: number;
  connectorId: string;
  deleteCredentials?: boolean;
}

export function useDisableAgentConnector() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ agentId, connectorId, deleteCredentials }: DisableArgs) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.DELETE(
        "/v1/agents/{agent_id}/connectors/{connector_id}",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: {
            path: { agent_id: agentId, connector_id: connectorId },
            query: deleteCredentials ? { delete_credentials: true } : {},
          },
        },
      );
      if (error) throw new Error("Failed to disable connector");
      return data as DisableResponse;
    },
    onSuccess: (_data, { agentId, connectorId, deleteCredentials }) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-connectors", agentId],
      });
      if (deleteCredentials) {
        queryClient.invalidateQueries({ queryKey: ["credentials"] });
        queryClient.invalidateQueries({
          queryKey: ["agent-connector-credential", agentId, connectorId],
        });
      }
    },
  });

  return {
    disableConnector: (args: DisableArgs) => mutation.mutateAsync(args),
    isLoading: mutation.isPending,
  };
}
