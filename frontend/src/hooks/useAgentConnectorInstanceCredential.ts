import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentConnectorCredentialResponse =
  components["schemas"]["AgentConnectorCredentialResponse"];

export function useAgentConnectorInstanceCredential(
  agentId: number,
  connectorId: string,
  instanceId: string,
) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent-connector-instance-credential", agentId, connectorId, instanceId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
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
      if (error) throw new Error("Failed to load credential binding");
      return data;
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId && !!instanceId,
  });

  return {
    binding: query.data ?? null,
    isLoading: query.isLoading,
    isCredentialBindingPending: query.isPending,
    error: query.isError ? "Unable to load credential binding." : null,
  };
}

export function useAssignAgentConnectorInstanceCredential() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      connectorId,
      instanceId,
      credentialId,
      oauthConnectionId,
    }: {
      agentId: number;
      connectorId: string;
      instanceId: string;
      credentialId?: string;
      oauthConnectionId?: string;
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.PUT(
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
          "agent-connector-instance-credential",
          variables.agentId,
          variables.connectorId,
          variables.instanceId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-calendars",
          variables.agentId,
          variables.connectorId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "connector-instance-bindings-summary",
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

export function useRemoveAgentConnectorInstanceCredential() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      connectorId,
      instanceId,
    }: {
      agentId: number;
      connectorId: string;
      instanceId: string;
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { error } = await client.DELETE(
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
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to remove credential";
        throw new Error(msg);
      }
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-instance-credential",
          variables.agentId,
          variables.connectorId,
          variables.instanceId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "agent-connector-calendars",
          variables.agentId,
          variables.connectorId,
        ],
      });
      queryClient.invalidateQueries({
        queryKey: [
          "connector-instance-bindings-summary",
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
