import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ApprovalSummary = components["schemas"]["ApprovalSummary"];

export function useApprovals() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  // Keep the latest token in a ref so the query key stays stable across
  // token refreshes (e.g. Supabase re-issues tokens on AAL promotion).
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["approvals", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/approvals", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { status: "pending" } },
      });
      if (error) throw new Error("Failed to load approvals");
      return data;
    },
    enabled: !!accessToken,
    // SSE (useApprovalEvents) provides instant updates by invalidating this
    // query when approval events arrive. The poll is a safety net for cases
    // where SSE is unavailable (e.g. corporate proxies that strip SSE, or
    // brief disconnections). 30 seconds is enough to catch up without
    // hammering the server.
    refetchInterval: 30_000,
  });

  return {
    approvals: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load approvals. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
