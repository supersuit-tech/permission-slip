import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";
import { isStripeUrl } from "@/pages/billing/formatters";

export function useBillingPortal() {
  const { session } = useAuth();

  const mutation = useMutation({
    mutationFn: async () => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }

      const { data, error } = await client.POST("/v1/billing/portal", {
        headers: { Authorization: `Bearer ${session.access_token}` },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to open billing portal"),
        );
      }
      return data;
    },
  });

  return {
    openPortal: async () => {
      const result = await mutation.mutateAsync();
      if (!result?.url || !isStripeUrl(result.url)) {
        throw new Error("Invalid billing portal URL received");
      }
      window.location.href = result.url;
    },
    isLoading: mutation.isPending,
    error: mutation.error?.message ?? null,
  };
}
