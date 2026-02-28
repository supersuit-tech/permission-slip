import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type Agent = components["schemas"]["AgentSummary"];

/** Polling interval when at least one agent is in "pending" status (5 seconds). */
const PENDING_POLL_INTERVAL = 5_000;

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
    // Poll while any agent is pending so the dashboard updates
    // when the agent completes verification.
    refetchInterval: (query) => {
      const agents = query.state.data?.data;
      const hasPending = agents?.some((a) => a.status === "pending");
      return hasPending ? PENDING_POLL_INTERVAL : false;
    },
  });

  return {
    agents: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load agents. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
