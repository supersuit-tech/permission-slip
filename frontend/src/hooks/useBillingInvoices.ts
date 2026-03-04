import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type Invoice = components["schemas"]["Invoice"];

export function useBillingInvoices(limit = 10) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["billing", "invoices", userId ?? "", limit],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/billing/invoices", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { limit } },
      });
      if (error) throw new Error("Failed to load invoices");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    invoices: query.data?.invoices ?? [],
    hasMore: query.data?.has_more ?? false,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load invoices. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
