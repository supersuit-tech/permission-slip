import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type OAuthConnection = components["schemas"]["OAuthConnection"];

export function useOAuthConnections() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["oauth-connections", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/oauth/connections", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load OAuth connections");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    connections: query.data?.connections ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load connected accounts. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
