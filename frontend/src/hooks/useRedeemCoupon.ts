import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";
import { getApiErrorMessage } from "@/api/errors";

export function useRedeemCoupon() {
  const { session } = useAuth();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (coupon: string) => {
      if (!session?.access_token) {
        throw new Error("Not authenticated");
      }
      const { data, error } = await client.POST("/v1/billing/redeem-coupon", {
        headers: { Authorization: `Bearer ${session.access_token}` },
        body: { coupon: coupon.trim() },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to redeem coupon"),
        );
      }
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing"] });
    },
  });
}
