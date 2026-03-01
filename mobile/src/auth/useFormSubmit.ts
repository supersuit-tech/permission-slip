import { useState } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";

interface UseFormSubmitResult {
  /** User-safe error message from the last submission, or null. */
  error: string | null;
  /** Whether a submission is currently in flight. */
  isSubmitting: boolean;
  /** Wraps an async auth action with loading/error state management.
   *  Supabase AuthErrors are converted to safe user-facing messages
   *  via `safeErrorMessage`; unexpected exceptions get a generic fallback. */
  handleSubmit: (
    action: () => Promise<{ error: AuthError | null }>
  ) => Promise<void>;
}

/**
 * Manages form submission state (loading, error) for auth screens.
 * Wraps Supabase auth calls and converts errors to safe user-facing messages.
 *
 * @example
 * ```tsx
 * const { error, isSubmitting, handleSubmit } = useFormSubmit();
 * const submit = () => handleSubmit(() => sendOtp(email));
 * ```
 */
export function useFormSubmit(): UseFormSubmitResult {
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (
    action: () => Promise<{ error: AuthError | null }>
  ) => {
    setError(null);
    setIsSubmitting(true);

    try {
      const { error } = await action();
      if (error) {
        setError(safeErrorMessage(error));
      }
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return { error, isSubmitting, handleSubmit };
}
