import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type ConnectorSummary = components["schemas"]["ConnectorSummary"];

export function useConnectors() {
  const query = useQuery({
    queryKey: ["connectors"],
    queryFn: async () => {
      const { data, error } = await client.GET("/v1/connectors");
      if (error) throw new Error("Failed to load connectors");
      return data;
    },
  });

  return {
    connectors: query.data?.data ?? [],
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load connectors. Please try again later."
      : null,
    refetch: query.refetch,
  };
}
