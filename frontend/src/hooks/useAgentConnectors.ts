import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentConnector = components["schemas"]["AgentConnectorSummary"];

export function useAgentConnectors(agentId: number) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent-connectors", agentId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/connectors",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
        },
      );
      if (error) throw new Error("Failed to load agent connectors");
      return data;
    },
    enabled: !!accessToken && agentId > 0,
  });

  return {
    connectors: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load connectors. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
