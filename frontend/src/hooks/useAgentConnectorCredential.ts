import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentConnectorCredential =
  components["schemas"]["AgentConnectorCredentialResponse"];

export function useAgentConnectorCredential(
  agentId: number,
  connectorId: string,
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent-connector-credential", agentId, connectorId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/credential",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
        },
      );
      if (error) throw new Error("Failed to load credential binding");
      return data;
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId,
  });

  return {
    binding: query.data ?? null,
    isLoading: query.isLoading,
    /** True while the first credential binding fetch is in flight (avoids UI flash). */
    isCredentialBindingPending: query.isPending,
    error: query.isError
      ? "Unable to load credential binding."
      : null,
  };
}

export function useAssignAgentConnectorCredential() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      connectorId,
      credentialId,
      oauthConnectionId,
    }: {
      agentId: number;
      connectorId: string;
      credentialId?: string;
      oauthConnectionId?: string;
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.PUT(
        "/v1/agents/{agent_id}/connectors/{connector_id}/credential",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
          body: {
            credential_id: credentialId,
            oauth_connection_id: oauthConnectionId,
          },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to assign credential";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-credential",
          variables.agentId,
          variables.connectorId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-calendars",
          variables.agentId,
          variables.connectorId,
        ],
      });
    },
  });

  return {
    assign: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useRemoveAgentConnectorCredential() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      connectorId,
    }: {
      agentId: number;
      connectorId: string;
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.DELETE(
        "/v1/agents/{agent_id}/connectors/{connector_id}/credential",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to remove credential";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-credential",
          variables.agentId,
          variables.connectorId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-calendars",
          variables.agentId,
          variables.connectorId,
        ],
      });
    },
  });

  return {
    remove: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}
