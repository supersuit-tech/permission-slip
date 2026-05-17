import { useState } from "react";
import * as Sentry from "@sentry/react";
import type { AuthError } from "./types";

interface UseFormSubmitResult {
  error: AuthError | null;
  isSubmitting: boolean;
  runSubmit: (
    action: () => Promise<{ error: AuthError | null }>
  ) => Promise<void>;
}

export function useFormSubmit(): UseFormSubmitResult {
  const [error, setError] = useState<AuthError | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const runSubmit = async (
    action: () => Promise<{ error: AuthError | null }>
  ) => {
    setError(null);
    setIsSubmitting(true);

    try {
      const { error: actionError } = await action();
      setError(actionError);
    } catch (err) {
      console.error("[auth] form submit threw:", err);
      Sentry.captureException(err);
      setError({
        message: "Something went wrong. Please try again.",
        code: "unknown",
        name: "AuthApiError",
        status: 500,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return { error, isSubmitting, runSubmit };
}
