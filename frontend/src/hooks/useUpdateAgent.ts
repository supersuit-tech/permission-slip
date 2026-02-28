import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { trackEvent } from "@/lib/posthog";
import { PostHogEvents } from "@/lib/posthog-events";

export function useUpdateAgent() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      agentId,
      metadata,
    }: {
      agentId: number;
      metadata: Record<string, unknown>;
    }) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PATCH("/v1/agents/{agent_id}", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        params: { path: { agent_id: agentId } },
        body: { metadata },
      });
      if (error) throw new Error("Failed to update agent");
      return data;
    },
    onSuccess: (_data, variables) => {
      trackEvent(PostHogEvents.AGENT_UPDATED, { agent_id: variables.agentId });
      queryClient.invalidateQueries({ queryKey: ["agent", variables.agentId] });
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });

  return {
    updateAgent: mutation.mutateAsync,
    isLoading: mutation.isPending,
  };
}
