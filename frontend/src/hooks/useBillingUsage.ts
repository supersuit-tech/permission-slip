import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type UsageDetail = components["schemas"]["UsageDetail"];

export function useBillingUsage(periodStart?: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["billing", "usage", userId ?? "", periodStart ?? "current"],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/billing/usage", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: periodStart ? { period_start: periodStart } : {} },
      });
      if (error) throw new Error("Failed to load usage details");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    usage: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load usage details. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
