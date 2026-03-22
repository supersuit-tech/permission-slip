import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ActionConfiguration = components["schemas"]["ActionConfiguration"];

export function useActionConfigs(agentId: number) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["action-configs", agentId],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/action-configurations", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { agent_id: agentId } },
      });
      if (error) throw new Error("Failed to load action configurations");
      return data;
    },
    enabled: !!accessToken && agentId > 0,
  });

  return {
    configs: query.data?.data ?? [],
    isLoading: query.isLoading,
    /** True after this query has finished a fetch (success or error), including empty results. */
    isFetched: query.isFetched,
    error: query.isError
      ? "Unable to load action configurations. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
