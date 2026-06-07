import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

type BulkDecisionItem = components["schemas"]["BulkApprovalDecisionItem"];

export function useBulkDecideApprovalGroup() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      bulkGroupId,
      decisions,
    }: {
      bulkGroupId: string;
      decisions: BulkDecisionItem[];
    }) => {
      if (!session?.access_token) throw new Error("Not authenticated");
      const { data, error } = await client.POST(
        "/v1/approval-groups/{group_id}/decide",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { group_id: bulkGroupId } },
          body: { decisions },
        },
      );
      if (error) {
        throw new Error(getApiErrorMessage(error, "Failed to submit bulk review"));
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
      queryClient.invalidateQueries({
        queryKey: ["approval-bulk-group", variables.bulkGroupId],
      });
    },
  });
}
