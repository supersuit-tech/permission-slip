import { useState } from "react";
import { Gift, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useRedeemCoupon } from "@/hooks/useRedeemCoupon";

export function CouponRedemptionCard() {
  const [code, setCode] = useState("");
  const { mutateAsync, isPending } = useRedeemCoupon();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = code.trim();
    if (!trimmed) {
      toast.error("Enter a coupon code");
      return;
    }
    try {
      const result = await mutateAsync(trimmed);
      if (result?.status === "already_redeemed") {
        toast.success("You already have complimentary Pro access.");
      } else {
        toast.success("Complimentary Pro activated.");
      }
      setCode("");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not redeem coupon.",
      );
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Gift className="text-muted-foreground size-5" aria-hidden="true" />
          <CardTitle>Have a coupon?</CardTitle>
        </div>
        <CardDescription>
          Enter your code to unlock complimentary Pro access (unlimited usage at
          no charge), subject to our Terms of Service.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="billing-coupon-code">Coupon code</Label>
            <Input
              id="billing-coupon-code"
              name="coupon"
              autoComplete="off"
              spellCheck={false}
              placeholder="Paste your 64-character hex code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              disabled={isPending}
            />
          </div>
          <div className="flex justify-end">
            <Button type="submit" disabled={isPending}>
              {isPending ? (
                <>
                  <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />
                  Redeeming…
                </>
              ) : (
                "Redeem"
              )}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
