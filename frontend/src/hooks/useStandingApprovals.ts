import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type StandingApproval = components["schemas"]["StandingApproval"];

export function useStandingApprovals(options?: {
  sourceActionConfigurationId?: string;
}) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const userId = session?.user?.id;
  const sourceConfig = options?.sourceActionConfigurationId;

  // Keep the latest token in a ref so the query key stays stable across
  // token refreshes (e.g. Supabase re-issues tokens on AAL promotion).
  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["standing-approvals", userId ?? "", sourceConfig ?? ""],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/standing-approvals", {
        headers: { Authorization: `Bearer ${token}` },
        params: {
          query: {
            status: "active",
            ...(sourceConfig
              ? { source_action_configuration_id: sourceConfig }
              : {}),
          },
        },
      });
      if (error) throw new Error("Failed to load standing approvals");
      return data;
    },
    enabled: !!accessToken,
  });

  return {
    standingApprovals: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load standing approvals. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
