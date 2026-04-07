import { useState } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface EmailStepProps {
  onSubmit: (email: string) => Promise<{ error: AuthError | null }>;
  onUsePassword?: (email: string) => void;
}

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function EmailStep({ onSubmit, onUsePassword }: EmailStepProps) {
  const [email, setEmail] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const trimmedEmail = email.trim();
  const isValidEmail = EMAIL_REGEX.test(trimmedEmail);

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Enter your email to sign in or create an account.
      </p>
      <form
        onSubmit={(e) => handleSubmit(e, () => onSubmit(trimmedEmail))}
        className="space-y-4"
      >
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <FormError error={error} prefix />
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          Continue
        </Button>
      </form>
      {onUsePassword ? (
        <button
          type="button"
          className="text-sm text-muted-foreground underline hover:text-foreground"
          onClick={() => onUsePassword(trimmedEmail)}
          disabled={!isValidEmail}
        >
          Sign in with password instead
        </button>
      ) : null}
    </AuthLayout>
  );
}
