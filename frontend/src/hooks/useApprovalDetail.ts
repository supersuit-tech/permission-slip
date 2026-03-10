import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ApprovalDetail = components["schemas"]["ApprovalSummary"];

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
      if (!token) throw new Error("Missing access token");
      if (!approvalId) throw new Error("Missing approval ID");
      const { data, error } = await client.GET(
        "/v1/approvals/{approval_id}",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { approval_id: approvalId } },
        },
      );
      if (error) throw new Error("Failed to load approval details");
      return data;
    },
    enabled: !!accessToken && !!approvalId,
    staleTime: 60_000,
  });

  return {
    approval: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load approval details."
      : null,
  };
}
