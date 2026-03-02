/**
 * React Query hook for fetching the user's registered agents, plus a
 * utility to derive a human-readable display name from agent metadata.
 */
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import type { components } from "../api/schema";

export type AgentSummary = components["schemas"]["AgentSummary"];

interface AgentMetadata {
  name?: string;
  [key: string]: unknown;
}

/** Safely narrows unknown agent metadata to an object with an optional `name` field. */
function parseAgentMetadata(raw: unknown): AgentMetadata | null {
  if (raw != null && typeof raw === "object" && !Array.isArray(raw)) {
    return raw as AgentMetadata;
  }
  return null;
}

/**
 * Returns a human-friendly name for an agent. Prefers `metadata.name`
 * if present; falls back to "Agent {id}".
 */
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

/**
 * Fetches all agents belonging to the current user. Data is cached by
 * userId and does not auto-refetch (agents change infrequently).
 */
export function useAgents() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const query = useQuery({
    queryKey: ["agents", userId ?? ""],
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
