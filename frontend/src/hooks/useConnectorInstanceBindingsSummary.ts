import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

export interface ConnectorInstanceBindingsSummary {
  instanceCount: number;
  oauthConnectionIds: string[];
  hasApiKeyCredential: boolean;
}

/**
 * Loads connector instances and each instance's credential binding for danger-zone
 * copy and OAuth disconnect-on-remove.
 */
export function useConnectorInstanceBindingsSummary(
  agentId: number,
  connectorId: string,
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  return useQuery({
    queryKey: ["connector-instance-bindings-summary", agentId, connectorId],
    queryFn: async (): Promise<ConnectorInstanceBindingsSummary> => {
      if (!accessToken) throw new Error("Missing access token");
      const { data: listData, error: listErr } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
        },
      );
      if (listErr) throw new Error("Failed to list instances");
      const instances = listData?.data ?? [];
      const oauthIds: string[] = [];
      let hasApiKeyCredential = false;

      await Promise.all(
        instances.map(async (inst) => {
          const { data: credData, error: credErr } = await client.GET(
            "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}/credential",
            {
              headers: { Authorization: `Bearer ${accessToken}` },
              params: {
                path: {
                  agent_id: agentId,
                  connector_id: connectorId,
                  instance_id: inst.connector_instance_id,
                },
              },
            },
          );
          if (credErr || !credData) return;
          if (credData.oauth_connection_id) {
            oauthIds.push(credData.oauth_connection_id);
          }
          if (credData.credential_id) {
            hasApiKeyCredential = true;
          }
        }),
      );

      return {
        instanceCount: instances.length,
        oauthConnectionIds: [...new Set(oauthIds)],
        hasApiKeyCredential,
      };
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId,
  });
}
