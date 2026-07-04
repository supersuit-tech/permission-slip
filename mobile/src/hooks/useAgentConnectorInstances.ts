import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import type { components } from "../api/schema";

export type AgentConnectorInstance = components["schemas"]["AgentConnectorInstance"];

export function useAgentConnectorInstances(agentId: number, connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: ["agent-connector-instances", agentId, connectorId],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
        },
      );
      if (error) throw new Error("Failed to load connector instances");
      return data?.data ?? [];
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId,
  });

  return {
    instances: query.data ?? [],
    isLoading: query.isLoading,
    error: query.isError ? "Unable to load connector instances." : null,
    refetch: query.refetch,
  };
}
