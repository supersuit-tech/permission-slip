import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import type { components } from "@/api/schema";

type UpdateProfileRequest = components["schemas"]["UpdateProfileRequest"];

/**
 * Hook for updating the current user's contact fields (email, phone).
 * Supports partial updates — omitted fields are left unchanged.
 * Invalidates the profile cache on success.
 */
export function useUpdateProfile() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (body: UpdateProfileRequest) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.PATCH("/v1/profile", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body,
      });
      if (error) throw new Error("Failed to update profile");
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["profile"] });
    },
  });

  return {
    updateProfile: mutation.mutateAsync,
    isLoading: mutation.isPending,
    error: mutation.error,
  };
}
