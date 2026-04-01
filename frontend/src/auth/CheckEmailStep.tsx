import type { AuthError } from "@supabase/supabase-js";
import { useResend } from "./useResend";
import AuthLayout from "./AuthLayout";
import { ResendButton } from "./ResendButton";
import { Mail } from "lucide-react";
import { Button } from "@/components/ui/button";

interface CheckEmailStepProps {
  email: string;
  onBack: () => void;
  onResend: () => Promise<{ error: AuthError | null }>;
}

export default function CheckEmailStep({
  email,
  onBack,
  onResend,
}: CheckEmailStepProps) {
  const resend = useResend({ onResend });

  return (
    <AuthLayout>
      <div className="space-y-4 text-center">
        <div className="flex justify-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <Mail className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
          </div>
        </div>
        <div className="space-y-1">
          <p className="text-sm font-medium">Check your email</p>
          <p className="text-sm text-muted-foreground">
            We sent a sign-in link to <strong>{email}</strong>.
          </p>
          <p className="text-xs text-muted-foreground">
            Click the link in your email to continue. If you don&apos;t see it,
            check your spam folder.
          </p>
        </div>
        <Button variant="outline" className="w-full" onClick={onBack}>
          Back
        </Button>
      </div>
      <ResendButton
        isResending={resend.isResending}
        error={resend.error}
        success={resend.success}
        onResend={resend.handleResend}
        label="Resend email"
        successMessage="Email resent."
      />
    </AuthLayout>
  );
}
