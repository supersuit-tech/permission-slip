import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import type { components } from "../api/schema";

export type ConnectorDetailResponse =
  components["schemas"]["ConnectorDetailResponse"];

export function useConnectorDetail(connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  const query = useQuery({
    queryKey: ["connector", connectorId],
    queryFn: async (): Promise<ConnectorDetailResponse> => {
      const token = tokenRef.current;
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.GET("/v1/connectors/{connector_id}", {
        headers: { Authorization: `Bearer ${token}` },
        params: { path: { connector_id: connectorId } },
      });
      if (error) throw new Error("Failed to load connector details");
      return data;
    },
    enabled: !!accessToken && !!connectorId,
  });

  return {
    connector: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load connector details. Please try again later."
      : null,
  };
}
