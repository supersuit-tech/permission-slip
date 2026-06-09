import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import type { components } from "@/api/schema";

type TestRequest = components["schemas"]["TestCredentialConnectionRequest"];

export function useTestCredentialConnection() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (body: TestRequest) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST("/v1/credentials/test-connection", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Bridge connection test failed"),
        );
      }
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["credentials"] });
    },
  });

  return {
    testConnection: (body: TestRequest) => mutation.mutateAsync(body),
    isTesting: mutation.isPending,
  };
}
