import { useState } from "react";
import {
  Elements,
  CardElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import type { Stripe } from "@stripe/stripe-js";
import { loadStripe } from "@stripe/stripe-js";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  useCreateSetupIntent,
  useConfirmPaymentMethod,
} from "@/hooks/usePaymentMethods";

let stripePromise: Promise<Stripe | null> | null = null;

function getStripe(): Promise<Stripe | null> {
  if (!stripePromise) {
    const key = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as
      | string
      | undefined;
    if (!key) {
      return Promise.resolve(null);
    }
    stripePromise = loadStripe(key);
  }
  return stripePromise;
}

interface AddCardDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddCardDialog({ open, onOpenChange }: AddCardDialogProps) {
  const key = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as
    | string
    | undefined;

  if (!key) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Payment Method</DialogTitle>
            <DialogDescription>
              Payment methods are not available. Stripe is not configured.
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Payment Method</DialogTitle>
          <DialogDescription>
            Enter your card details. Card data is sent directly to Stripe and
            never touches our servers.
          </DialogDescription>
        </DialogHeader>
        <Elements stripe={getStripe()}>
          <CardForm onSuccess={() => onOpenChange(false)} />
        </Elements>
      </DialogContent>
    </Dialog>
  );
}

function CardForm({ onSuccess }: { onSuccess: () => void }) {
  const stripe = useStripe();
  const elements = useElements();
  const { createSetupIntent, isLoading: isCreatingIntent } =
    useCreateSetupIntent();
  const { confirmPaymentMethod, isLoading: isConfirming } =
    useConfirmPaymentMethod();
  const [label, setLabel] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const isLoading = isCreatingIntent || isConfirming || isSubmitting;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!stripe || !elements) return;

    setIsSubmitting(true);
    try {
      // 1. Create SetupIntent on our server.
      const { client_secret } = await createSetupIntent();

      // 2. Confirm the SetupIntent with Stripe Elements.
      const cardElement = elements.getElement(CardElement);
      if (!cardElement) {
        toast.error("Card element not found.");
        return;
      }

      const result = await stripe.confirmCardSetup(client_secret, {
        payment_method: { card: cardElement },
      });

      if (result.error) {
        toast.error(result.error.message ?? "Failed to set up card.");
        return;
      }

      const paymentMethodId = result.setupIntent.payment_method;
      if (!paymentMethodId || typeof paymentMethodId !== "string") {
        toast.error("Unexpected response from Stripe.");
        return;
      }

      // 3. Save the payment method in our database.
      await confirmPaymentMethod({
        payment_method_id: paymentMethodId,
        label: label || undefined,
        is_default: false,
      });

      toast.success("Payment method added successfully.");
      onSuccess();
    } catch {
      toast.error("Failed to add payment method. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="card-label">Label (optional)</Label>
        <Input
          id="card-label"
          placeholder="e.g. Personal Visa, Work Card"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          disabled={isLoading}
        />
      </div>

      <div className="space-y-2">
        <Label>Card Details</Label>
        <div className="rounded-md border p-3">
          <CardElement
            options={{
              style: {
                base: {
                  fontSize: "16px",
                  color: "hsl(var(--foreground))",
                  "::placeholder": {
                    color: "hsl(var(--muted-foreground))",
                  },
                },
              },
            }}
          />
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button
          type="submit"
          disabled={!stripe || isLoading}
        >
          {isLoading ? (
            <>
              <Loader2 className="mr-2 size-4 animate-spin" />
              Adding...
            </>
          ) : (
            "Add Card"
          )}
        </Button>
      </div>
    </form>
  );
}
