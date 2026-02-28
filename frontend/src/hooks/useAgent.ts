import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentDetail = components["schemas"]["AgentSummary"];

export function useAgent(agentId: number) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent", agentId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/agents/{agent_id}", {
        headers: { Authorization: `Bearer ${accessToken}` },
        params: { path: { agent_id: agentId } },
      });
      if (error) throw new Error("Failed to load agent");
      return data;
    },
    enabled: !!accessToken && agentId > 0,
  });

  return {
    agent: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load agent. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
