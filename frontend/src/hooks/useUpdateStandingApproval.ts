import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";
import { trackEvent, PostHogEvents } from "@/lib/posthog";

type UpdateStandingApprovalRequest =
  components["schemas"]["UpdateStandingApprovalRequest"];

export function useUpdateStandingApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      standingApprovalId,
      req,
    }: {
      standingApprovalId: string;
      req: UpdateStandingApprovalRequest;
    }) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/standing-approvals/{standing_approval_id}/update",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { standing_approval_id: standingApprovalId } },
          body: req,
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to update standing approval"),
        );
      }
      return data;
    },
    onSuccess: () => {
      trackEvent(PostHogEvents.STANDING_APPROVAL_UPDATED);
      queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
    },
  });

  return {
    updateStandingApproval: (
      standingApprovalId: string,
      req: UpdateStandingApprovalRequest,
    ) => mutation.mutateAsync({ standingApprovalId, req }),
    isPending: mutation.isPending,
  };
}
