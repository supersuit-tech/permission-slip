/**
 * React Query hook for fetching the authenticated user's approval requests.
 * Polls every 30 seconds and supports filtering by status.
 */
import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type ApprovalSummary = components["schemas"]["ApprovalSummary"];
export type ApprovalStatus = ApprovalSummary["status"];

/**
 * Fetches approval requests for the current user, filtered by status.
 * Uses a stable query key (keyed by userId + status) and auto-refetches
 * every 30 seconds. The access token is stored in a ref to avoid
 * unnecessary query invalidation on token refresh.
 */
export function useApprovals(status: "pending" | "approved" | "denied" | "cancelled" | "all" = "pending") {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  // Keep the latest token in a ref so the query key stays stable across
  // token refreshes (e.g. Supabase re-issues tokens on AAL promotion).
  // Clear the ref when the token is gone (sign-out) to avoid stale tokens.
  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: ["approvals", userId ?? "", status],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/approvals", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { status } },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Unable to load approvals. Please try again later."),
        );
      }
      return data;
    },
    enabled: !!accessToken,
    refetchInterval: 30_000,
    // Cache stays fresh for 10 s — avoids redundant refetches when the user
    // switches tabs back and forth quickly.
    staleTime: 10_000,
  });

  return {
    approvals: query.data?.data ?? [],
    hasMore: query.data?.has_more ?? false,
    isLoading: query.isLoading,
    isRefetching: query.isRefetching,
    error: query.isError
      ? (query.error?.message ?? "Unable to load approvals. Please try again later.")
      : null,
    refetch: query.refetch,
  };
}
