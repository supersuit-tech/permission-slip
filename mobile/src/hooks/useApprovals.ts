/**
 * React Query hook for fetching the authenticated user's approval requests.
 *
 * Pending approvals poll every 10 seconds (time-sensitive — users need to act
 * quickly). Historical tabs (approved / denied) don't poll since they rarely
 * change. When the app returns to the foreground the focus manager (configured
 * in App.tsx) triggers an immediate refetch so data is always fresh.
 */
import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type ApprovalSummary = components["schemas"]["ApprovalSummary"];
export type ApprovalStatus = ApprovalSummary["status"];

/** Polling interval in ms — only pending approvals auto-refresh. */
const PENDING_POLL_INTERVAL_MS = 10_000;

/**
 * Fetches approval requests for the current user, filtered by status.
 * Uses a stable query key (keyed by userId + status). The access token is
 * stored in a ref to avoid unnecessary query invalidation on token refresh.
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
    // Only pending approvals auto-poll — approved / denied are historical.
    refetchInterval: status === "pending" ? PENDING_POLL_INTERVAL_MS : false,
    // Immediately refetch when the app returns to the foreground.
    refetchOnWindowFocus: true,
    // Cache stays fresh for 5 s — avoids redundant refetches when the user
    // switches tabs back and forth quickly.
    staleTime: 5_000,
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
    /** Epoch ms when the query data was last successfully fetched (0 if never). */
    dataUpdatedAt: query.dataUpdatedAt,
  };
}
