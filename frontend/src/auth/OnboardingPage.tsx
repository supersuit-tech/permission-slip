import { useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "./AuthContext";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import client from "@/api/client";
import validation from "@/lib/validation";

export default function OnboardingPage() {
  const { session, signOut } = useAuth();
  const queryClient = useQueryClient();
  const [username, setUsername] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      const { error: apiError } = await client.POST("/v1/onboarding", {
        headers: { Authorization: `Bearer ${session?.access_token}` },
        body: { username: username.trim() },
      });

      if (apiError) {
        setError(
          apiError.error?.message ??
            "Something went wrong. Please try again."
        );
        return;
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
        <FormError error={error} prefix />
        <div className="flex gap-2">
          <Button type="submit" className="flex-1" disabled={isSubmitting}>
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
