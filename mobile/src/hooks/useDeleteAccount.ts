/**
 * React Query mutation hook for permanently deleting the user's account.
 * Requires `confirmation: "DELETE"` in the request body as a safety check.
 * On success, signs the user out locally.
 */
import { useRef } from "react";
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import client from "../api/client";
import { getApiErrorMessage } from "../api/errors";

export function useDeleteAccount() {
  const { session, signOut } = useAuth();

  const tokenRef = useRef(session?.access_token);
  tokenRef.current = session?.access_token;

  const mutation = useMutation({
    mutationFn: async () => {
      const token = tokenRef.current;
      if (!token) {
        throw new Error("Not authenticated");
      }
      const { data, error } = await client.DELETE("/v1/profile", {
        headers: { Authorization: `Bearer ${token}` },
        body: { confirmation: "DELETE" },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to delete account. Please try again."),
        );
      }
      return data;
    },
    onSuccess: () => {
      signOut();
    },
  });

  return {
    deleteAccount: mutation.mutateAsync,
    isDeleting: mutation.isPending,
    error: mutation.error,
  };
}
