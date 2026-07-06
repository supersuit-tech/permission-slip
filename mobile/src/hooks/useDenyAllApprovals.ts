import { useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";
import type { components } from "../api/schema";

type ApprovalListResponse = components["schemas"]["ApprovalListResponse"];

export function useDenyAllApprovals(status: "pending" | "approved" | "denied" = "pending") {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const userId = session?.user?.id ?? "";
  const tokenRef = useRef(session?.access_token);
  if (session?.access_token) {
    tokenRef.current = session.access_token;
  }

  const mutation = useMutation({
    mutationFn: async () => {
      const token = tokenRef.current;
      if (!token) throw new Error("Not authenticated");

      const { data, error } = await client.POST("/v1/approvals/deny-all", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to decline pending requests"),
        );
      }
      return data;
    },
    onMutate: async () => {
      const queryKey = ["approvals", userId, status] as const;
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<ApprovalListResponse>(queryKey);
      if (previous) {
        queryClient.setQueryData<ApprovalListResponse>(queryKey, {
          ...previous,
          data: previous.data.filter((item) => item.bulk_group_id),
        });
      }
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["approvals", userId, status], context.previous);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
    },
  });

  return {
    denyAllApprovals: () => mutation.mutateAsync(),
    isPending: mutation.isPending,
    error: mutation.error,
    reset: mutation.reset,
  };
}
