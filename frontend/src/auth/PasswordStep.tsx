import { useState } from "react";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface PasswordStepProps {
  email: string;
  onSubmit: (password: string) => Promise<{ error: AuthError | null }>;
  onBack: () => void;
}

export default function PasswordStep({
  email,
  onSubmit,
  onBack,
}: PasswordStepProps) {
  const [password, setPassword] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Sign in as <strong>{email}</strong>
      </p>
      <form
        onSubmit={(e) => handleSubmit(e, () => onSubmit(password))}
        className="space-y-4"
      >
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoFocus
            required
          />
        </div>
        <FormError error={error} prefix />
        <div className="flex gap-2">
          <Button type="submit" className="flex-1" disabled={isSubmitting}>
            Sign In
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
    </AuthLayout>
  );
}
