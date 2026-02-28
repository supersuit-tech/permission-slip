import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type DataRetention = components["schemas"]["DataRetentionResponse"];

export function useDataRetention() {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["data-retention"],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/profile/data-retention", {
        headers: { Authorization: `Bearer ${accessToken}` },
      });
      if (error) throw new Error("Failed to load data retention policy");
      // Cast is safe: openapi-fetch returns typed response matching the schema
      return data as DataRetention;
    },
    enabled: !!accessToken,
    staleTime: 60_000,
  });

  return { dataRetention: data ?? null, isLoading, error, refetch };
}
