import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type InstanceCredentialBinding =
  components["schemas"]["AgentConnectorCredentialResponse"];

/**
 * Loads credential bindings for each connector instance row (batch GET per instance).
 */
export function useConnectorInstanceCredentialBindings(
  agentId: number,
  connectorId: string,
  instanceIds: string[],
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const sortedIds = [...instanceIds].sort().join(",");

  return useQuery({
    queryKey: [
      "connector-instance-credential-bindings",
      agentId,
      connectorId,
      sortedIds,
    ],
    queryFn: async (): Promise<Map<string, InstanceCredentialBinding | null>> => {
      if (!accessToken) throw new Error("Missing access token");
      const out = new Map<string, InstanceCredentialBinding | null>();
      await Promise.all(
        instanceIds.map(async (instanceId) => {
          const { data, error } = await client.GET(
            "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential",
            {
              headers: { Authorization: `Bearer ${accessToken}` },
              params: {
                path: {
                  agent_id: agentId,
                  connector_id: connectorId,
                  instance_id: instanceId,
                },
              },
            },
          );
          if (error) throw new Error("Failed to load instance credential binding");
          out.set(instanceId, data ?? null);
        }),
      );
      return out;
    },
    enabled:
      !!accessToken && agentId > 0 && !!connectorId && instanceIds.length > 0,
  });
}
