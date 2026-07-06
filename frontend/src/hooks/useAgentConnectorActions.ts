import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";
import client from "@/api/client";
import { useAgentConnectors } from "@/hooks/useAgentConnectors";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";

export type AgentActionOption = Pick<ConnectorAction, "action_type" | "name"> & {
  connector_id: string;
};

export function useAgentConnectorActions(agentId: number, enabled = true) {
  const { connectors, isLoading: connectorsLoading } =
    useAgentConnectors(agentId);

  const detailQueries = useQueries({
    queries: connectors.map((connector) => ({
      queryKey: ["connector", connector.id],
      queryFn: async () => {
        const { data, error } = await client.GET(
          "/v1/connectors/{connector_id}",
          {
            params: { path: { connector_id: connector.id } },
          },
        );
        if (error) throw new Error("Failed to load connector details");
        return data;
      },
      enabled: enabled && agentId > 0,
    })),
  });

  const actionsByConnector = useMemo(() => {
    const out: Record<string, AgentActionOption[]> = {};
    connectors.forEach((connector, index) => {
      const detail = detailQueries[index]?.data;
      if (!detail) return;
      out[connector.id] = detail.actions.map((action) => ({
        connector_id: connector.id,
        action_type: action.action_type,
        name: action.name,
      }));
    });
    return out;
  }, [connectors, detailQueries]);

  const isLoading =
    connectorsLoading || detailQueries.some((q) => q.isLoading);

  return {
    actionsByConnector,
    isLoading,
  };
}
