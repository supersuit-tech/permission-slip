import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentWebhookConfig = components["schemas"]["AgentWebhookConfig"];
export type AgentWebhookTestResult =
  components["schemas"]["AgentWebhookTestResult"];

function webhookQueryKey(agentId: number) {
  return ["agent-webhook", agentId] as const;
}

export function useAgentWebhook(agentId: number) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: webhookQueryKey(agentId),
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/webhook",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
        },
      );
      if (error) throw new Error("Failed to load webhook configuration");
      return data;
    },
    enabled: !!accessToken && agentId > 0,
  });

  return {
    webhook: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load webhook configuration."
      : null,
    refetch: query.refetch,
  };
}

export function useSetAgentWebhook() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      url,
      token,
      provider,
    }: {
      agentId: number;
      url: string;
      token: string;
      provider?: "openclaw" | "grokbot";
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.PUT(
        "/v1/agents/{agent_id}/webhook",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
          body: { url, token, provider: provider ?? "openclaw" },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to save webhook";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (data, variables) => {
      queryClient.setQueryData(webhookQueryKey(variables.agentId), data);
    },
  });

  return {
    setWebhook: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useTestAgentWebhook() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ agentId }: { agentId: number }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/webhook",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: {
            path: { agent_id: agentId },
            query: { test: true },
          },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Webhook test failed";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (data, variables) => {
      queryClient.setQueryData(webhookQueryKey(variables.agentId), data);
    },
  });

  return {
    testWebhook: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useClearAgentWebhook() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ agentId }: { agentId: number }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { error } = await client.DELETE("/v1/agents/{agent_id}/webhook", {
        headers: { Authorization: `Bearer ${accessToken}` },
        params: { path: { agent_id: agentId } },
      });
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to clear webhook";
        throw new Error(msg);
      }
    },
    onSuccess: (_data, variables) => {
      queryClient.setQueryData(webhookQueryKey(variables.agentId), {
        configured: false,
      });
    },
  });

  return {
    clearWebhook: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}
