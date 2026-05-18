import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type AgentPaymentMethodResponse =
  components["schemas"]["AgentPaymentMethodResponse"];

export function useAgentPaymentMethod(agentId: number) {
  const { session } = useAuth();
  const accessToken = session?.access_token;

  const query = useQuery({
    queryKey: ["agent-payment-method", agentId],
    queryFn: async () => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.GET(
        "/v1/agents/{agent_id}/payment-method",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
        },
      );
      if (error) throw new Error("Failed to load agent payment method");
      return data;
    },
    enabled: !!accessToken && agentId > 0,
  });

  return {
    binding: query.data ?? null,
    isLoading: query.isLoading,
    error: query.isError
      ? "Unable to load payment method assignment."
      : null,
  };
}

export function useAssignAgentPaymentMethod() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      paymentMethodId,
    }: {
      agentId: number;
      paymentMethodId: string;
    }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.PUT(
        "/v1/agents/{agent_id}/payment-method",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
          body: { payment_method_id: paymentMethodId },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to assign payment method";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-payment-method", variables.agentId],
      });
    },
  });

  return {
    assign: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}

export function useRemoveAgentPaymentMethod() {
  const { session } = useAuth();
  const accessToken = session?.access_token;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ agentId }: { agentId: number }) => {
      if (!accessToken) throw new Error("Missing access token");
      const { data, error } = await client.DELETE(
        "/v1/agents/{agent_id}/payment-method",
        {
          headers: { Authorization: `Bearer ${accessToken}` },
          params: { path: { agent_id: agentId } },
        },
      );
      if (error) {
        const msg =
          (error as { error?: { message?: string } }).error?.message ??
          "Failed to remove payment method";
        throw new Error(msg);
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["agent-payment-method", variables.agentId],
      });
    },
  });

  return {
    remove: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}
