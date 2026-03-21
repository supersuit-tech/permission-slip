import { useState } from "react";
import { Ticket } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { FormError } from "@/components/FormError";
import { useRedeemCoupon } from "@/hooks/useRedeemCoupon";

interface CouponRedemptionCardProps {
  onRedeemed: () => void;
}

export function CouponRedemptionCard({ onRedeemed }: CouponRedemptionCardProps) {
  const [couponCode, setCouponCode] = useState("");
  const { redeemCoupon, isLoading, error } = useRedeemCoupon();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const success = await redeemCoupon(couponCode.trim());
    if (success) {
      onRedeemed();
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Ticket className="text-muted-foreground size-5" />
          <CardTitle>Redeem Coupon</CardTitle>
        </div>
        <CardDescription>
          Have a coupon code? Enter it below to activate your free pro account.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-3">
          <Input
            type="text"
            placeholder="Enter coupon code"
            value={couponCode}
            onChange={(e) => setCouponCode(e.target.value)}
            disabled={isLoading}
          />
          {error && <FormError error={error} />}
          <Button
            type="submit"
            disabled={isLoading || couponCode.trim().length === 0}
            size="sm"
          >
            {isLoading ? "Redeeming..." : "Redeem Coupon"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
