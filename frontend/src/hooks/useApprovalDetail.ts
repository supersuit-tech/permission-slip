import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ApprovalDetail = components["schemas"]["ApprovalSummary"];

/** Error with an HTTP status code attached for downstream handling. */
class ApprovalFetchError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}

/**
 * Fetches a single approval by ID for the activity feed detail panel.
 * Only fetches when approvalId is non-null (lazy loading).
 */
export function useApprovalDetail(approvalId: string | null) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["approval-detail", approvalId],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new ApprovalFetchError("Missing access token", 401);
      if (!approvalId) throw new ApprovalFetchError("Missing approval ID", 400);
      const { data, error, response } = await client.GET(
        "/v1/approvals/{approval_id}",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { approval_id: approvalId } },
        },
      );
      if (error) {
        throw new ApprovalFetchError(
          "Failed to load approval details",
          response?.status ?? 500,
        );
      }
      return data;
    },
    enabled: !!accessToken && !!approvalId,
    staleTime: Infinity,
    // Don't retry auth or not-found errors — only retry server errors.
    retry: (failureCount, err) => {
      if (err instanceof ApprovalFetchError && err.status < 500) return false;
      return failureCount < 2;
    },
  });

  const errorStatus =
    query.error instanceof ApprovalFetchError ? query.error.status : null;

  return {
    approval: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError ? "Unable to load approval details." : null,
    /** HTTP status from the failed request (null when no error). */
    errorStatus,
  };
}
