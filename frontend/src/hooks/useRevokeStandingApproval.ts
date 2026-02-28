import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useRevokeStandingApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (standingApprovalId: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/standing-approvals/{standing_approval_id}/revoke",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: {
            path: { standing_approval_id: standingApprovalId },
          },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to revoke standing approval"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
    },
  });

  return {
    revokeStandingApproval: (id: string) => mutation.mutateAsync(id),
    isPending: mutation.isPending,
  };
}
