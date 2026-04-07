import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "./AuthContext";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { FormError } from "@/components/FormError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { supabase } from "@/lib/supabaseClient";
import client from "@/api/client";
import validation from "@/lib/validation";

export default function OnboardingPage() {
  const { session, signOut } = useAuth();
  const queryClient = useQueryClient();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [marketingOptIn, setMarketingOptIn] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const { error: apiError } = await client.POST("/v1/onboarding", {
        headers: { Authorization: `Bearer ${session?.access_token}` },
        body: { username: username.trim(), marketing_opt_in: marketingOptIn },
      });

      if (apiError) {
        setError(
          apiError.error?.message ??
            "Something went wrong. Please try again."
        );
        return;
      }

      // Set the user's password via Supabase after profile creation.
      const { error: pwError } = await supabase.auth.updateUser({
        password,
      });
      if (pwError) {
        // Profile was created but password failed — log the user in
        // anyway so they aren't stuck. They can set a password later.
        console.error("Failed to set password during onboarding:", pwError.message);
      }

      // Invalidate the profile query so App.tsx re-fetches and routes to dashboard
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
    } catch {
      setError("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Welcome! Choose a username to finish setting up your account.
      </p>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="username">Username</Label>
          <Input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="e.g. alice or my-team"
            minLength={validation.username.minLength}
            maxLength={validation.username.maxLength}
            required
            autoFocus
          />
          <p className="text-xs text-muted-foreground">
            3–32 characters. Letters, digits, underscores, and hyphens only.
          </p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={validation.password.minLength}
            required
          />
          <p className="text-xs text-muted-foreground">
            At least {validation.password.minLength} characters.
          </p>
        </div>
        <div className="flex items-start gap-2">
          <Checkbox
            id="agree-tos"
            checked={agreedToTerms}
            onCheckedChange={(checked) => setAgreedToTerms(checked === true)}
            required
          />
          <div className="grid gap-1">
            <Label
              htmlFor="agree-tos"
              className="text-sm font-normal leading-snug"
            >
              I agree to the Terms of Service and Privacy Policy
            </Label>
            <div className="flex gap-3 text-sm">
              <Link
                to="/policy/terms"
                target="_blank"
                className="text-primary underline underline-offset-2 hover:text-primary/80"
              >
                Terms of Service
              </Link>
              <Link
                to="/policy/privacy"
                target="_blank"
                className="text-primary underline underline-offset-2 hover:text-primary/80"
              >
                Privacy Policy
              </Link>
            </div>
          </div>
        </div>
        <div className="flex items-start gap-2">
          <Checkbox
            id="marketing-opt-in"
            checked={marketingOptIn}
            onCheckedChange={(checked) => setMarketingOptIn(checked === true)}
          />
          <Label
            htmlFor="marketing-opt-in"
            className="text-sm font-normal leading-snug"
          >
            Keep me in the loop — Get occasional emails about new features,
            platform updates, and tips to get more out of Permission Slip.
          </Label>
        </div>
        <FormError error={error} prefix />
        <div className="flex gap-2">
          <Button
            type="submit"
            className="flex-1"
            disabled={isSubmitting || !agreedToTerms || password.length < validation.password.minLength}
          >
            {isSubmitting ? "Creating account…" : "Create account"}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => signOut()}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
        </div>
      </form>
    </AuthLayout>
  );
}
