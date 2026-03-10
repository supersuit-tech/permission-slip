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
}

export default function EmailStep({ onSubmit }: EmailStepProps) {
  const [email, setEmail] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Enter your email and we'll send you a sign-in link.
      </p>
      <form
        onSubmit={(e) => handleSubmit(e, () => onSubmit(email.trim()))}
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
    </AuthLayout>
  );
}
