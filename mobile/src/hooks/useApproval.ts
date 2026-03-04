/**
 * Hook to fetch a single approval by ID. Used when navigating via deep link
 * where only the approval_id is available (no pre-fetched ApprovalSummary).
 *
 * First checks the React Query cache for the approval (from a prior list
 * fetch), then falls back to fetching all approvals and finding the match.
 */
import { useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { ApprovalSummary } from "./useApprovals";

/**
 * Matches the approval_id format from the OpenAPI spec.
 * Defense-in-depth: prevents API calls with obviously invalid IDs
 * from crafted or malformed deep link URLs.
 */
const APPROVAL_ID_PATTERN = /^appr_[a-zA-Z0-9]{6,64}$/;

/**
 * Searches the React Query cache for an approval with the given ID.
 * Checks all cached approval list queries (pending, approved, denied, etc.).
 */
function findInCache(
  queryClient: ReturnType<typeof useQueryClient>,
  approvalId: string,
): ApprovalSummary | undefined {
  const queries = queryClient.getQueriesData<{ data?: ApprovalSummary[] }>({
    queryKey: ["approvals"],
  });
  for (const [, queryData] of queries) {
    const match = queryData?.data?.find(
      (a) => a.approval_id === approvalId,
    );
    if (match) return match;
  }
  return undefined;
}

export function useApproval(approvalId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  // Validate the approval ID format before making any API calls
  const isValidId = APPROVAL_ID_PATTERN.test(approvalId);

  const query = useQuery({
    queryKey: ["approval", approvalId],
    queryFn: async () => {
      // Check cache first
      const cached = findInCache(queryClient, approvalId);
      if (cached) return cached;

      // Fetch all statuses and find the matching approval
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");

      const { data, error } = await client.GET("/v1/approvals", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { status: "all" } },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Unable to load approval."),
        );
      }
      const match = data?.data?.find(
        (a: ApprovalSummary) => a.approval_id === approvalId,
      );
      if (!match) throw new Error("Approval not found");
      return match;
    },
    // Only enable if we have auth and a valid-looking approval ID
    enabled: !!accessToken && !!approvalId && isValidId,
    staleTime: 10_000,
    // Limit retries for deep links — the approval may be expired or deleted.
    // One retry handles transient network errors without hammering the API
    // for genuinely missing approvals.
    retry: 1,
  });

  // Return an immediate error for invalid IDs without making any API calls
  if (!isValidId && approvalId) {
    return {
      approval: null,
      isLoading: false,
      error: "Invalid approval ID format",
      refetch: query.refetch,
    };
  }

  return {
    approval: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? (query.error?.message ?? "Unable to load approval.")
      : null,
    refetch: query.refetch,
  };
}
