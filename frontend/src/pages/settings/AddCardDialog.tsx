import { useState } from "react";
import {
  Elements,
  CardElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js";
import type { Stripe, StripeCardElementChangeEvent } from "@stripe/stripe-js";
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
import { useConfig } from "@/hooks/useConfig";

const stripePromiseCache: Record<string, Promise<Stripe | null>> = {};

function getStripe(testMode: boolean): Promise<Stripe | null> {
  const key = testMode
    ? (import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY_TEST as string | undefined)
    : (import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as string | undefined);
  if (!key) return Promise.resolve(null);
  const cached = stripePromiseCache[key];
  if (cached) return cached;
  const promise = loadStripe(key);
  stripePromiseCache[key] = promise;
  return promise;
}

interface AddCardDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddCardDialog({ open, onOpenChange }: AddCardDialogProps) {
  const { config } = useConfig();
  const testMode = config?.stripe_test_mode ?? false;
  const key = testMode
    ? (import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY_TEST as string | undefined)
    : (import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY as string | undefined);

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
          <div className="flex justify-end">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </div>
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
        <Elements stripe={getStripe(testMode)}>
          <CardForm onSuccess={() => onOpenChange(false)} />
        </Elements>
      </DialogContent>
    </Dialog>
  );
}

type SubmitStep = "idle" | "creating_intent" | "confirming_card" | "saving";

const stepLabels: Record<SubmitStep, string> = {
  idle: "Add Card",
  creating_intent: "Preparing secure form...",
  confirming_card: "Processing with Stripe...",
  saving: "Saving card...",
};

function CardForm({ onSuccess }: { onSuccess: () => void }) {
  const stripe = useStripe();
  const elements = useElements();
  const { createSetupIntent } = useCreateSetupIntent();
  const { confirmPaymentMethod } = useConfirmPaymentMethod();
  const [label, setLabel] = useState("");
  const [step, setStep] = useState<SubmitStep>("idle");
  const [cardComplete, setCardComplete] = useState(false);
  const [cardError, setCardError] = useState<string | null>(null);

  const isLoading = step !== "idle";

  function handleCardChange(event: StripeCardElementChangeEvent) {
    setCardComplete(event.complete);
    setCardError(event.error?.message ?? null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!stripe || !elements) return;

    try {
      // 1. Create SetupIntent on our server.
      setStep("creating_intent");
      const { client_secret } = await createSetupIntent();

      // 2. Confirm the SetupIntent with Stripe Elements.
      setStep("confirming_card");
      const cardElement = elements.getElement(CardElement);
      if (!cardElement) {
        toast.error("Card element not found. Please refresh and try again.");
        return;
      }

      const result = await stripe.confirmCardSetup(client_secret, {
        payment_method: { card: cardElement },
      });

      if (result.error) {
        const msg = result.error.message ?? "Card could not be verified.";
        if (result.error.type === "card_error") {
          toast.error(`Card declined: ${msg}`);
        } else if (result.error.type === "validation_error") {
          toast.error(`Invalid card details: ${msg}`);
        } else {
          toast.error(msg);
        }
        return;
      }

      const paymentMethodId = result.setupIntent.payment_method;
      if (!paymentMethodId || typeof paymentMethodId !== "string") {
        toast.error("Unexpected response from Stripe. Please try again.");
        return;
      }

      // 3. Save the payment method in our database.
      setStep("saving");
      await confirmPaymentMethod({
        payment_method_id: paymentMethodId,
        label: label || undefined,
        is_default: false,
      });

      toast.success("Payment method added successfully.");
      onSuccess();
    } catch {
      toast.error(
        "Something went wrong while adding your card. Please try again.",
      );
    } finally {
      setStep("idle");
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
        <div
          className={`rounded-md border p-3 transition-colors ${
            cardError
              ? "border-red-500"
              : "focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500"
          }`}
        >
          <CardElement
            onChange={handleCardChange}
            options={{
              style: {
                base: {
                  fontSize: "16px",
                  color: "hsl(var(--foreground))",
                  "::placeholder": {
                    color: "hsl(var(--muted-foreground))",
                  },
                },
                invalid: {
                  color: "hsl(var(--destructive))",
                  iconColor: "hsl(var(--destructive))",
                },
              },
            }}
          />
        </div>
        {cardError && (
          <p className="text-destructive text-xs">{cardError}</p>
        )}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button
          type="submit"
          disabled={!stripe || isLoading || !cardComplete}
        >
          {isLoading ? (
            <>
              <Loader2 className="mr-2 size-4 animate-spin" />
              {stepLabels[step]}
            </>
          ) : (
            "Add Card"
          )}
        </Button>
      </div>
    </form>
  );
}
