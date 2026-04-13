import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

export type BulkApplyResult = components["schemas"]["BulkApplyResult"];
export type BulkApplyActionConfigTemplateResponse =
  components["schemas"]["BulkApplyActionConfigTemplateResponse"];

export function useBulkApplyActionConfigTemplates() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const token = session?.access_token;

  const mutation = useMutation({
    mutationFn: async (input: {
      templateIds: string[];
      agentId: number;
      approvalModes?: Record<string, "auto_approve" | "requires_approval">;
    }): Promise<BulkApplyActionConfigTemplateResponse> => {
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.POST(
        "/v1/action-config-templates/bulk-apply",
        {
          headers: { Authorization: `Bearer ${token}` },
          body: {
            agent_id: input.agentId,
            template_ids: input.templateIds,
            ...(input.approvalModes && { approval_modes: input.approvalModes }),
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
        queryKey: ["action-configs", variables.agentId],
      });
      void queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
    },
  });

  return {
    bulkApply: mutation.mutateAsync,
    isBulkPending: mutation.isPending,
  };
}
