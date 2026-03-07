import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ConfigResponse = components["schemas"]["ConfigResponse"];

export function useConfig() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["config"],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/config", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load server config");
      return data;
    },
    enabled: !!accessToken,
    staleTime: 5 * 60 * 1000, // config changes infrequently
  });

  return {
    config: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load server configuration. Please try again later."
      : null,
  };
}
