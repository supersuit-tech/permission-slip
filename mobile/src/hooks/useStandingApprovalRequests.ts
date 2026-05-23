import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

export type StandingApprovalRequestSummary =
  components["schemas"]["StandingApprovalRequest"];

const PENDING_POLL_INTERVAL_MS = 10_000;

export function useStandingApprovalRequests() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;
  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: ["standing-approval-requests", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/standing-approval-requests", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { status: "pending" } },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Unable to load rule proposals."),
        );
      }
      return data;
    },
    enabled: !!accessToken,
    refetchInterval: PENDING_POLL_INTERVAL_MS,
  });

  return {
    requests: query.data?.data ?? [],
    isLoading: query.isLoading,
    isRefetching: query.isRefetching,
    error: query.error?.message ?? null,
    refetch: query.refetch,
    dataUpdatedAt: query.dataUpdatedAt,
  };
}
