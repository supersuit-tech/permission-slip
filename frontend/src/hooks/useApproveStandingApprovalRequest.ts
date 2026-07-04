import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useApproveStandingApprovalRequest() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (requestId: string) => {
      if (!session?.access_token) throw new Error("Not authenticated");
      const { data, error } = await client.POST(
        "/v1/standing-approval-requests/{request_id}/approve",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { request_id: requestId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to approve rule proposal"),
        );
      }
      return data;
    },
    onSuccess: () => {
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["standing-approval-requests"] });
        queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
      }, 2_000);
    },
  });

  return {
    approveRequest: (requestId: string) => mutation.mutateAsync(requestId),
    isPending: mutation.isPending,
  };
}
