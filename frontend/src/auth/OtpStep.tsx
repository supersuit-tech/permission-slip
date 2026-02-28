import { useState } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";
import AuthLayout from "./AuthLayout";
import DevOnly from "../components/DevOnly";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { OtpCodeInput } from "@/components/OtpCodeInput";

interface OtpStepProps {
  email: string;
  onVerify: (code: string) => Promise<{ error: AuthError | null }>;
  onBack: () => void;
}

export default function OtpStep({ email, onVerify, onBack }: OtpStepProps) {
  const [otpCode, setOtpCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();

  const handleAutoFill = async () => {
    // Dynamic import keeps dev-only Mailpit code out of the production bundle
    const { fetchOtpFromMailpit } = await import("./dev");
    const code = await fetchOtpFromMailpit(email);
    if (code) {
      setOtpCode(code);
    }
  };

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Enter the code sent to <strong>{email}</strong>
      </p>
      <form
        onSubmit={(e) => handleSubmit(e, () => onVerify(otpCode))}
        className="space-y-4"
      >
        <OtpCodeInput
          id="otp-code"
          label="Code"
          value={otpCode}
          onChange={setOtpCode}
          required
        />
        <FormError error={error} prefix />
        <div className="flex gap-2">
          <Button type="submit" className="flex-1" disabled={isSubmitting}>
            Verify
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={onBack}
            disabled={isSubmitting}
          >
            Back
          </Button>
        </div>
      </form>
      <DevOnly>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleAutoFill}
          className="mt-3 opacity-70"
        >
          Dev: Auto-fill code from Mailpit
        </Button>
      </DevOnly>
    </AuthLayout>
  );
}
