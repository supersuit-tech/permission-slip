/**
 * React Query mutation hook for approving an approval request.
 * Returns the confirmation code on success (XXX-XXX format).
 */
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";

export function useApproveApproval() {
  const { session } = useAuth();

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
    // No query invalidation — the detail screen stays visible so the user
    // can read/copy the confirmation code. The 30-second polling in
    // useApprovals handles eventual list updates.
  });

  return {
    approveApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}
