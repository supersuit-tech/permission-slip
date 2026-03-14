import { useMemo, useRef } from "react";
import { useQueries } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { ActionConfiguration } from "./useActionConfigs";

/**
 * Fetches action configurations for multiple agents and returns
 * a Map keyed by config ID for quick lookups.
 */
export function useActionConfigMap(agentIds: number[]) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const uniqueIds = useMemo(
    () => [...new Set(agentIds.filter((id) => id > 0))],
    [agentIds],
  );

  const results = useQueries({
    queries: uniqueIds.map((agentId) => ({
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
      enabled: !!accessToken,
    })),
  });

  const configMap = useMemo(() => {
    const map = new Map<string, ActionConfiguration>();
    for (const result of results) {
      const configs = result.data?.data;
      if (configs) {
        for (const config of configs) {
          map.set(config.id, config);
        }
      }
    }
    return map;
  }, [results]);

  return configMap;
}
