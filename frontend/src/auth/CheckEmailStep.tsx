import { useState, useEffect } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";
import AuthLayout from "./AuthLayout";
import { Mail } from "lucide-react";
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
        setResendError(
          safeErrorMessage(error, {
            over_email_send_rate_limit:
              "Too many sign-in emails sent. If you already received a link, you can still use it — otherwise wait a few minutes and try again.",
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

  return (
    <AuthLayout>
      <div className="space-y-4 text-center">
        <div className="flex justify-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <Mail className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
          </div>
        </div>
        <div className="space-y-1">
          <p className="text-sm font-medium">Check your email</p>
          <p className="text-sm text-muted-foreground">
            We sent a sign-in link to <strong>{email}</strong>.
          </p>
          <p className="text-xs text-muted-foreground">
            Click the link in your email to continue. If you don&apos;t see it,
            check your spam folder.
          </p>
        </div>
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
