import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { trackEvent } from "@/lib/posthog";
import { PostHogEvents } from "@/lib/posthog-events";

export function useDenyApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (approvalId: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { error } = await client.POST(
        "/v1/approvals/{approval_id}/deny",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { approval_id: approvalId } },
        },
      );
      if (error) throw new Error("Failed to deny request");
    },
    onSuccess: () => {
      trackEvent(PostHogEvents.APPROVAL_DENIED);
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
    },
  });

  return {
    denyApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
  };
}
