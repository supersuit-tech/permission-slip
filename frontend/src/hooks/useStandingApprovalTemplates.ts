import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type StandingApprovalTemplate =
  components["schemas"]["StandingApprovalTemplate"];

export function useStandingApprovalTemplates(connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["standing-approval-templates", connectorId],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/standing-approval-templates",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { query: { connector_id: connectorId } },
        },
      );
      if (error) {
        throw new Error("Failed to load standing approval templates");
      }
      return data;
    },
    enabled: !!accessToken && !!connectorId,
  });

  return {
    templates: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load standing approval templates. Please try again later."
      : null,
  };
}
