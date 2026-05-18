import { useState, type FormEvent } from "react";
import { useAuth } from "./AuthContext";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useFormSubmit } from "./useFormSubmit";
import { safeErrorMessage } from "./errors";
import validation from "@/lib/validation";

type Mode = "login" | "signup";

export default function LoginPage() {
  const { signInWithPassword, signUpWithPassword } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { error, isSubmitting, runSubmit } = useFormSubmit();

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    await runSubmit(async () => {
      const fn =
        mode === "login" ? signInWithPassword : signUpWithPassword;
      return fn(email.trim(), password);
    });
  };

  return (
    <AuthLayout>
      <div className="mb-4 flex gap-2 rounded-md border border-border p-1 text-sm">
        <button
          type="button"
          className={`flex-1 rounded px-3 py-1.5 font-medium ${
            mode === "login"
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => {
            setMode("login");
          }}
        >
          Sign in
        </button>
        <button
          type="button"
          className={`flex-1 rounded px-3 py-1.5 font-medium ${
            mode === "signup"
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:text-foreground"
          }`}
          onClick={() => {
            setMode("signup");
          }}
        >
          Create account
        </button>
      </div>
      <p className="text-sm text-muted-foreground">
        {mode === "login"
          ? "Sign in with your email and password."
          : "Create an account with your email and password."}
      </p>
      <form role="form" className="mt-4 space-y-4" onSubmit={handleSubmit}>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={validation.password.minLength}
            required
          />
          <p className="text-xs text-muted-foreground">
            At least {validation.password.minLength} characters.
          </p>
        </div>
        <FormError
          error={error ? safeErrorMessage(error) : null}
          prefix
        />
        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting
            ? mode === "login"
              ? "Signing in…"
              : "Creating account…"
            : mode === "login"
              ? "Sign in"
              : "Create account"}
        </Button>
      </form>
    </AuthLayout>
  );
}
