import { useState, useEffect } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";

interface CheckEmailStepProps {
  email: string;
  onBack: () => void;
  onResend: () => Promise<{ error: AuthError | null }>;
  resendCooldownSeconds: number;
}

export default function CheckEmailStep({
  email,
  onBack,
  onResend,
  resendCooldownSeconds,
}: CheckEmailStepProps) {
  const [resendError, setResendError] = useState<string | null>(null);
  const [resendSuccess, setResendSuccess] = useState(false);
  const [isResending, setIsResending] = useState(false);

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
        setResendError(safeErrorMessage(error));
      } else {
        setResendSuccess(true);
      }
    } catch {
      setResendError("Something went wrong. Please try again.");
    } finally {
      setIsResending(false);
    }
  };

  return (
    <AuthLayout>
      <div className="space-y-4 text-center">
        <p className="text-sm text-muted-foreground">
          We sent a sign-in link to <strong>{email}</strong>. Click the link in
          your email to continue.
        </p>
        <Button variant="outline" className="w-full" onClick={onBack}>
          Back
        </Button>
      </div>
      <div className="mt-3 flex flex-col items-start gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleResend}
          disabled={resendCooldownSeconds > 0 || isResending}
          aria-label={
            resendCooldownSeconds > 0
              ? `Resend email in ${resendCooldownSeconds}s (on cooldown)`
              : isResending
                ? "Resending…"
                : "Resend email"
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
            "Resend email"
          )}
        </Button>
        {resendError && (
          <p role="alert" className="text-xs text-destructive">{resendError}</p>
        )}
        {resendSuccess && (
          <p role="status" className="text-xs text-muted-foreground">Email resent.</p>
        )}
      </div>
    </AuthLayout>
  );
}
