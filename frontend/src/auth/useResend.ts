import { useState, useEffect, useCallback } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";

interface UseResendOptions {
  onResend: () => Promise<{ error: AuthError | null }>;
  cooldownSeconds: number;
  /** Context-specific error override for `over_email_send_rate_limit`. */
  rateLimitMessage: string;
}

interface UseResendResult {
  error: string | null;
  success: boolean;
  isResending: boolean;
  handleResend: () => Promise<void>;
}

/**
 * Encapsulates the resend state machine shared by OtpStep and CheckEmailStep:
 * loading flag, success/error banners, and auto-clearing when the cooldown expires.
 */
export function useResend({
  onResend,
  cooldownSeconds,
  rateLimitMessage,
}: UseResendOptions): UseResendResult {
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [isResending, setIsResending] = useState(false);

  // Clear banners when the cooldown expires so they don't linger
  // alongside an active resend button.
  useEffect(() => {
    if (cooldownSeconds === 0) {
      setSuccess(false);
      setError(null);
    }
  }, [cooldownSeconds]);

  const handleResend = useCallback(async () => {
    setError(null);
    setSuccess(false);
    setIsResending(true);
    try {
      const { error: resendError } = await onResend();
      if (resendError) {
        setError(
          safeErrorMessage(resendError, {
            over_email_send_rate_limit: rateLimitMessage,
          })
        );
      } else {
        setSuccess(true);
      }
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setIsResending(false);
    }
  }, [onResend, rateLimitMessage]);

  return { error, success, isResending, handleResend };
}
