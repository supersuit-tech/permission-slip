import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type ApprovalStatus = "pending" | "approved" | "denied";
export type ApprovalSummary = components["schemas"]["ApprovalSummary"];

/**
 * Fetches the list of approvals filtered by status.
 *
 * Mirrors the web frontend's useApprovals pattern: token stored in a ref
 * for stability, query key includes userId and status for correct
 * cache separation, and polling at 30 s as a fallback refresh mechanism.
 */
export function useApprovals(status: ApprovalStatus) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  // Keep token in a ref so query function always has the latest token
  // without causing refetches on every token refresh.
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

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
    isLoading: query.isLoading,
    isRefetching: query.isRefetching,
    error: query.isError ? (query.error?.message ?? "Unable to load approvals.") : null,
    refetch: query.refetch,
  };
}
