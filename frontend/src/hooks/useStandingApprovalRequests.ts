import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type StandingApprovalRequestSummary =
  components["schemas"]["StandingApprovalRequest"];

export function useStandingApprovalRequests() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["standing-approval-requests", userId ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/standing-approval-requests", {
        headers: { Authorization: `Bearer ${token}` },
        params: { query: { status: "pending" } },
      });
      if (error) throw new Error("Failed to load rule proposals");
      return data;
    },
    enabled: !!accessToken,
    refetchInterval: 30_000,
  });

  return {
    requests: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load rule proposals. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
