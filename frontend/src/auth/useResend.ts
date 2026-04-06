import { useState, useCallback } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";

interface UseResendOptions {
  onResend: () => Promise<{ error: AuthError | null }>;
}

interface UseResendResult {
  error: string | null;
  success: boolean;
  isResending: boolean;
  handleResend: () => Promise<void>;
}

/**
 * Encapsulates the resend state machine used by OtpStep:
 * loading flag and success/error banners. Supabase enforces per-email rate
 * limits server-side so we treat `over_email_send_rate_limit` as success
 * (the previous email was already delivered).
 */
export function useResend({
  onResend,
}: UseResendOptions): UseResendResult {
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [isResending, setIsResending] = useState(false);

  const handleResend = useCallback(async () => {
    setError(null);
    setSuccess(false);
    setIsResending(true);
    try {
      const { error: resendError } = await onResend();
      // Treat rate limit as success — the previous email was already sent.
      // Supabase enforces the real cooldown server-side.
      if (resendError && resendError.code !== "over_email_send_rate_limit") {
        setError(safeErrorMessage(resendError));
      } else {
        setSuccess(true);
      }
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setIsResending(false);
    }
  }, [onResend]);

  return { error, success, isResending, handleResend };
}
