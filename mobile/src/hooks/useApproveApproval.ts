/**
 * React Query mutation hook for approving an approval request.
 * Returns the confirmation code on success (XXX-XXX format).
 */
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";

export function useApproveApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (approvalId: string) => {
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
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to approve request"),
        );
      }
      return data;
    },
    onSuccess: () => {
      // Invalidate list cache so the approval moves from pending to approved
      // when the user navigates back.
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
    },
  });

  return {
    approveApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}
