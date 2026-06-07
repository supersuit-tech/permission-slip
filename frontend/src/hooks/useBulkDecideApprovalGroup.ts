import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

export type BulkDecisionItem = components["schemas"]["BulkApprovalDecisionItem"];
export type BulkDecisionResponse =
  components["schemas"]["BulkApprovalDecisionResponse"];

export function useBulkDecideApprovalGroup() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      bulkGroupId,
      decisions,
    }: {
      bulkGroupId: string;
      decisions: BulkDecisionItem[];
    }): Promise<BulkDecisionResponse> => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }
      const { data, error } = await client.POST(
        "/v1/approval-groups/{group_id}/decide",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { group_id: bulkGroupId } },
          body: { decisions },
        },
      );
      if (error) throw new Error("Failed to submit bulk decisions");
      return data;
    },
    onSuccess: (_data, variables) => {
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["approvals"] });
        queryClient.invalidateQueries({
          queryKey: ["approval-bulk-group", variables.bulkGroupId],
        });
      }, 2_000);
    },
  });

  return {
    decideBulkGroup: mutation.mutateAsync,
    isPending: mutation.isPending,
    result: mutation.data,
    error: mutation.error,
  };
}
