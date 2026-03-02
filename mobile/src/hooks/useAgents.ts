import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import type { components } from "../api/schema";

export type AgentSummary = components["schemas"]["AgentSummary"];

interface AgentMetadata {
  name?: string;
  [key: string]: unknown;
}

function parseAgentMetadata(raw: unknown): AgentMetadata | null {
  if (raw != null && typeof raw === "object" && !Array.isArray(raw)) {
    return raw as AgentMetadata;
  }
  return null;
}

export function getAgentDisplayName(agent: {
  agent_id: number;
  metadata?: unknown;
}): string {
  const meta = parseAgentMetadata(agent.metadata);
  if (meta?.name && meta.name.length > 0) {
    return meta.name;
  }
  return `Agent ${agent.agent_id}`;
}

export function useAgents() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agents"],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/agents", {
        headers: { Authorization: `Bearer ${accessToken}` },
      });
      if (error) throw new Error("Failed to load agents");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    agents: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load agents. Please try again later."
      : null,
  };
}
