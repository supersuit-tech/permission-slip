import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

export type ApplyActionConfigTemplateResponse =
  components["schemas"]["ApplyActionConfigTemplateResponse"];

export function useApplyActionConfigTemplate() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const token = session?.access_token;

  const mutation = useMutation({
    mutationFn: async (input: {
      templateId: string;
      agentId: number;
      approvalMode?: "auto_approve" | "requires_approval";
    }) => {
      if (!token) throw new Error("Missing access token");
      const { data, error } = await client.POST(
        "/v1/action-config-templates/{id}/apply",
        {
          headers: { Authorization: `Bearer ${token}` },
          params: { path: { id: input.templateId } },
          body: {
            agent_id: input.agentId,
            ...(input.approvalMode && { approval_mode: input.approvalMode }),
          },
        },
      );
      if (error || !data) {
        throw new Error(
          getApiErrorMessage(error, "Failed to apply template"),
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
    applyTemplate: mutation.mutateAsync,
    isPending: mutation.isPending,
  };
}
