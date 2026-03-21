import { useState } from "react";
import { useAuth } from "@/auth/AuthContext";
import client from "@/api/client";

export function useRedeemCoupon() {
  const { session } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function redeemCoupon(couponCode: string): Promise<boolean> {
    const token = session?.access_token;
    if (!token) {
      setError("Not authenticated");
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const { data, error: apiError } = await client.POST(
        "/v1/billing/redeem-coupon",
        {
          headers: { Authorization: `Bearer ${token}` },
          body: { coupon_code: couponCode },
        },
      );

      if (apiError) {
        const message =
          (apiError as { error?: { message?: string } })?.error?.message ??
          "Failed to redeem coupon";
        setError(message);
        return false;
      }

      return data?.status === "redeemed" || data?.status === "already_redeemed";
    } catch {
      setError("Failed to redeem coupon. Please try again.");
      return false;
    } finally {
      setIsLoading(false);
    }
  }

  return { redeemCoupon, isLoading, error };
}
