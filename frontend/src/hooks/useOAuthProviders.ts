import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type OAuthProvider = components["schemas"]["OAuthProvider"];

export function useOAuthProviders(options?: { enabled?: boolean }) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const callerEnabled = options?.enabled ?? true;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["oauth-providers"],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/oauth/providers", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load OAuth providers");
      return data;
    },
    enabled: !!accessToken && callerEnabled,
    staleTime: 5 * 60 * 1000,
  });

  return {
    providers: query.data?.providers ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load OAuth providers. Please try again later."
      : null,
  };
}
