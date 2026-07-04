import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useDenyStandingApprovalRequest() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (requestId: string) => {
      if (!session?.access_token) throw new Error("Not authenticated");
      const { data, error } = await client.POST(
        "/v1/standing-approval-requests/{request_id}/deny",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { request_id: requestId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to deny rule proposal"),
        );
      }
      return data;
    },
    onSuccess: () => {
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["standing-approval-requests"] });
      }, 2_000);
    },
  });

  return {
    denyRequest: (requestId: string) => mutation.mutateAsync(requestId),
    isPending: mutation.isPending,
  };
}
