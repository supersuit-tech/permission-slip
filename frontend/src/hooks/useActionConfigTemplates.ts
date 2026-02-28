import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ActionConfigTemplate =
  components["schemas"]["ActionConfigTemplate"];

export function useActionConfigTemplates(connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const tokenRef = useRef(accessToken);
  if (accessToken) {
    tokenRef.current = accessToken;
  }

  const query = useQuery({
    queryKey: ["action-config-templates", connectorId],
    queryFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/action-config-templates",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { query: { connector_id: connectorId } },
        },
      );
      if (error) throw new Error("Failed to load action configuration templates");
      return data;
    },
    enabled: !!accessToken && !!connectorId,
  });

  return {
    templates: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load configuration templates. Please try again later."
      : null,
  };
}
