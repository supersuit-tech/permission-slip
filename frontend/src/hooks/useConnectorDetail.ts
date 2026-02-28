import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ConnectorDetailResponse =
  components["schemas"]["ConnectorDetailResponse"];
export type ConnectorAction = components["schemas"]["ConnectorAction"];
export type RequiredCredential = components["schemas"]["RequiredCredential"];

export function useConnectorDetail(connectorId: string) {
  const query = useQuery({
    queryKey: ["connector", connectorId],
    queryFn: async () => {
      const { data, error } = await client.GET(
        "/v1/connectors/{connector_id}",
        {
          params: { path: { connector_id: connectorId } },
        },
      );
      if (error) throw new Error("Failed to load connector details");
      return data;
    },
    enabled: !!connectorId,
  });

  return {
    connector: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load connector details. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
