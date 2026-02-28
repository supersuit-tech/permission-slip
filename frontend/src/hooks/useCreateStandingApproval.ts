import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type CreateStandingApprovalRequest =
  components["schemas"]["CreateStandingApprovalRequest"];

export function useCreateStandingApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (req: CreateStandingApprovalRequest) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST(
        "/v1/standing-approvals/create",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          body: req,
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to create standing approval"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["standing-approvals"] });
    },
  });

  return {
    createStandingApproval: (req: CreateStandingApprovalRequest) =>
      mutation.mutateAsync(req),
    isPending: mutation.isPending,
  };
}
