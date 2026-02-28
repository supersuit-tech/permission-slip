import { useState } from "react";
import { useAuth } from "./AuthContext";
import { useFormSubmit } from "./useFormSubmit";
import AuthLayout from "./AuthLayout";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { OtpCodeInput } from "@/components/OtpCodeInput";
import { useSignOut } from "@/hooks/useSignOut";

export default function MfaChallengePage() {
  const { verifyMfa } = useAuth();
  const [code, setCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const handleSignOut = useSignOut();

  return (
    <AuthLayout>
      <p className="text-sm text-muted-foreground">
        Enter the 6-digit code from your authenticator app to continue.
      </p>
      <form
        onSubmit={(e) => handleSubmit(e, () => verifyMfa(code))}
        className="space-y-4"
        noValidate
      >
        <OtpCodeInput
          id="mfa-code"
          label="Authenticator Code"
          value={code}
          onChange={setCode}
          autoFocus
        />
        <FormError error={error} prefix />
        <div className="flex gap-2">
          <Button type="submit" className="flex-1" disabled={isSubmitting}>
            Verify
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={handleSignOut}
            disabled={isSubmitting}
          >
            Sign Out
          </Button>
        </div>
      </form>
    </AuthLayout>
  );
}
