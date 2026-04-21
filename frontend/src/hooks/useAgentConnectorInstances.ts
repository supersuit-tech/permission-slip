import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentConnectorInstance = components["schemas"]["AgentConnectorInstance"];

export function useAgentConnectorInstances(agentId: number, connectorId: string) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent-connector-instances", agentId, connectorId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
        },
      );
      if (error) throw new Error("Failed to load connector instances");
      return data?.data ?? [];
    },
    enabled: !!accessToken && agentId > 0 && !!connectorId,
  });

  return {
    instances: query.data ?? [],
    isLoading: query.isLoading,
    error: query.isError ? "Unable to load connector instances." : null,
    refetch: query.refetch,
  };
}

export function useCreateAgentConnectorInstance() {
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
      const { data, error } = await client.POST(
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId, connector_id: connectorId } },
          body: {},
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to create instance";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-connector-instances", variables.agentId, variables.connectorId],
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
    create: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useSetDefaultAgentConnectorInstance() {
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
      const { data, error } = await client.PATCH(
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: {
            path: {
              agent_id: agentId,
              connector_id: connectorId,
              instance_id: instanceId,
            },
          },
          body: { is_default: true },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to set default instance";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-connector-instances", variables.agentId, variables.connectorId],
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
    setDefault: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useDeleteAgentConnectorInstance() {
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
        "/v1/agents/{agent_id}/connectors/{connector_id}/instances/{instance_id}",
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
          "Failed to delete instance";
        throw new Error(msg);
      }
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-connector-instances", variables.agentId, variables.connectorId],
      });
      queryClient.removeQueries({
        queryKey: [
          "agent-connector-instance-credential",
          variables.agentId,
          variables.connectorId,
          variables.instanceId,
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
    deleteInstance: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}
