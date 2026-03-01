import { useState } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";

interface UseFormSubmitResult {
  error: string | null;
  isSubmitting: boolean;
  handleSubmit: (
    action: () => Promise<{ error: AuthError | null }>
  ) => Promise<void>;
}

/** Manages form submission state for auth screens. */
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
