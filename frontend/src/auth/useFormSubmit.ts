import { useState, type FormEvent } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { safeErrorMessage } from "./errors";

interface UseFormSubmitResult {
  error: string | null;
  isSubmitting: boolean;
  handleSubmit: (
    e: FormEvent<HTMLFormElement>,
    action: () => Promise<{ error: AuthError | null }>
  ) => Promise<void>;
}

export function useFormSubmit(): UseFormSubmitResult {
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (
    e: FormEvent<HTMLFormElement>,
    action: () => Promise<{ error: AuthError | null }>
  ) => {
    e.preventDefault();
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
