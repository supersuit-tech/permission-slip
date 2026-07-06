import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

export type BulkApplyStandingApprovalTemplateResponse =
  components["schemas"]["BulkApplyStandingApprovalTemplateResponse"];

export function useBulkApplyStandingApprovalTemplates() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const token = session?.access_token;

  const mutation = useMutation({
    mutationFn: async (input: {
      templateIds: string[];
      agentId: number;
    }): Promise<BulkApplyStandingApprovalTemplateResponse> => {
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.POST(
        "/v1/standing-approval-templates/bulk-apply",
        {
          headers: { Authorization: `Bearer ${token}` },
          body: {
            agent_id: input.agentId,
            template_ids: input.templateIds,
          },
        },
      );
      if (error || !data) {
        throw new Error(
          getApiErrorMessage(error, "Failed to apply templates"),
        );
      }
      return data;
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["standing-approvals", variables.agentId],
      });
      void queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
    },
  });

  return {
    bulkApply: mutation.mutateAsync,
    isBulkPending: mutation.isPending,
  };
}
