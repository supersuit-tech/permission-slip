import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { trackEvent } from "@/lib/posthog";

export function useDeactivateAgent() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (agentId: number) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { error } = await client.POST(
        "/v1/agents/{agent_id}/deactivate",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { agent_id: agentId } },
        },
      );
      if (error) throw new Error("Failed to deactivate agent");
    },
    onSuccess: (_data, agentId) => {
      trackEvent("agent_deactivated", { agent_id: agentId });
      queryClient.invalidateQueries({ queryKey: ["agent", agentId] });
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });

  return {
    deactivateAgent: (agentId: number) => mutation.mutateAsync(agentId),
    isLoading: mutation.isPending,
  };
}
