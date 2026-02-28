import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

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
      if (error) throw new Error("Failed to approve request");
      return data;
    },
    // No onSuccess invalidation — the approved row stays visible so the user
    // can read/copy the confirmation code. The 5-second polling in useApprovals
    // handles eventual removal from the pending list.
  });

  return {
    approveApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
  };
}
