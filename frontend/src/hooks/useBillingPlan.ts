import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type BillingPlanResponse = components["schemas"]["BillingPlanResponse"];
export type Plan = components["schemas"]["Plan"];
export type Subscription = components["schemas"]["Subscription"];
export type UsageSummary = components["schemas"]["UsageSummary"];

export function useBillingPlan() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["billing", "plan", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/billing/plan", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) throw new Error("Failed to load billing plan");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    billingPlan: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load billing plan. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
