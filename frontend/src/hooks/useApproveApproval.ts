import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { trackEvent, PostHogEvents } from "@/lib/posthog";
import type { components } from "@/api/schema";

export type ApproveResult = components["schemas"]["ApproveApprovalResponse"];

export function useApproveApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (approvalId: string): Promise<ApproveResult> => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/approvals/{approval_id}/approve",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { approval_id: approvalId } },
        },
      );
      if (error) throw new Error("Failed to approve request");
      return data;
    },
    onSuccess: () => {
      trackEvent(PostHogEvents.APPROVAL_APPROVED);
      // Delay cache invalidation so the success state is visible in the
      // dialog before the row disappears from the pending list.
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["approvals"] });
      }, 2_000);
    },
  });

  return {
    approveApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
    result: mutation.data,
  };
}
