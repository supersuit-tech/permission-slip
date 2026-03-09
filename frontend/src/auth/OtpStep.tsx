import { useState, useEffect } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";
import { safeErrorMessage } from "./errors";
import AuthLayout from "./AuthLayout";
import DevOnly from "../components/DevOnly";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { OtpCodeInput } from "@/components/OtpCodeInput";

interface OtpStepProps {
  email: string;
  onVerify: (code: string) => Promise<{ error: AuthError | null }>;
  onBack: () => void;
  onResend: () => Promise<{ error: AuthError | null }>;
  resendCooldownSeconds: number;
}

export default function OtpStep({
  email,
  onVerify,
  onBack,
  onResend,
  resendCooldownSeconds,
}: OtpStepProps) {
  const [otpCode, setOtpCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const [resendError, setResendError] = useState<string | null>(null);
  const [resendSuccess, setResendSuccess] = useState(false);
  const [isResending, setIsResending] = useState(false);

  // Clear the success banner when the cooldown expires so it doesn't
  // linger indefinitely alongside an active "Resend code" button.
  useEffect(() => {
    if (resendCooldownSeconds === 0) {
      setResendSuccess(false);
      setResendError(null);
    }
  }, [resendCooldownSeconds]);

  const handleResend = async () => {
    setResendError(null);
    setResendSuccess(false);
    setIsResending(true);
    try {
      const { error } = await onResend();
      if (error) {
        setResendError(
          safeErrorMessage(error, {
            over_email_send_rate_limit:
              "Too many login emails sent. If you already received a code, you can still use it — otherwise wait a few minutes and try again.",
          })
        );
      } else {
        setResendSuccess(true);
      }
    } catch {
      setResendError("Something went wrong. Please try again.");
    } finally {
      setIsResending(false);
    }
  };

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
      <div className="mt-3 flex flex-col items-start gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleResend}
          disabled={resendCooldownSeconds > 0 || isResending}
          aria-label={
            resendCooldownSeconds > 0
              ? "Resend code (on cooldown)"
              : isResending
                ? "Resending…"
                : "Resend code"
          }
          className="opacity-70"
        >
          {resendCooldownSeconds > 0 ? (
            <>
              Resend in <span aria-hidden="true">{resendCooldownSeconds}s</span>
            </>
          ) : isResending ? (
            "Resending…"
          ) : (
            "Resend code"
          )}
        </Button>
        {resendError && (
          <p role="alert" className="text-xs text-destructive">{resendError}</p>
        )}
        {resendSuccess && (
          <p role="status" className="text-xs text-muted-foreground">Code resent.</p>
        )}
      </div>
      <DevOnly>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleAutoFill}
          className="mt-1 opacity-70"
        >
          Dev: Auto-fill code from Mailpit
        </Button>
      </DevOnly>
    </AuthLayout>
  );
}
