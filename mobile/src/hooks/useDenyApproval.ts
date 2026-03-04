/**
 * React Query mutation hook for denying a pending approval request.
 *
 * Calls `POST /v1/approvals/{approval_id}/deny` with the current session
 * token. On success, invalidates all `["approvals"]` queries so the list
 * screen refreshes. Errors are thrown as plain `Error` instances with the
 * server-provided message (or a generic fallback).
 *
 * Mirrors the web frontend's `useDenyApproval` pattern (`frontend/src/hooks/`).
 *
 * @returns `denyApproval(id)` — resolves on success, throws on failure.
 * @returns `isPending` — true while the deny request is in flight.
 */
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";

export function useDenyApproval() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (approvalId: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { error } = await client.POST(
        "/v1/approvals/{approval_id}/deny",
        {
          headers: { Authorization: `Bearer ${session.access_token}` },
          params: { path: { approval_id: approvalId } },
        },
      );
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to deny request"),
        );
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["approvals"] });
    },
  });

  return {
    denyApproval: (approvalId: string) => mutation.mutateAsync(approvalId),
    isPending: mutation.isPending,
  };
}
