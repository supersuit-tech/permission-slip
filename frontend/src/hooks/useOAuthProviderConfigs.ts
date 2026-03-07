import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type OAuthProviderConfig =
  components["schemas"]["OAuthProviderConfig"];

export function useOAuthProviderConfigs() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["oauth-provider-configs"],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/oauth/provider-configs",
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (error) throw new Error("Failed to load BYOA provider configs");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    configs: query.data?.configs ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load OAuth provider configurations. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
