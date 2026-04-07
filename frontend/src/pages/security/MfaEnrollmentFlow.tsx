import { useCallback, useEffect, useState, type FormEvent } from "react";
import { useAuth } from "@/auth/AuthContext";
import { safeErrorMessage } from "@/auth/errors";
import {
  savePendingEnrollment,
  hasPendingEnrollment,
  clearPendingEnrollment,
} from "@/auth/mfaPendingEnrollment";
import type { AuthError } from "@supabase/supabase-js";
import { Button } from "@/components/ui/button";
import { FormError } from "@/components/FormError";
import { OtpCodeInput } from "@/components/OtpCodeInput";

type Step = "idle" | "qr";

interface MfaEnrollmentFlowProps {
  onEnrolled: () => void;
}

export function MfaEnrollmentFlow({ onEnrolled }: MfaEnrollmentFlowProps) {
  const { user, enrollMfa, confirmMfaEnrollment, listMfaFactors, unenrollMfa } =
    useAuth();
  const userId = user?.id ?? "";
  const [step, setStep] = useState<Step>("idle");
  const [qrCode, setQrCode] = useState("");
  const [secret, setSecret] = useState("");
  const [showSecret, setShowSecret] = useState(false);
  const [factorId, setFactorId] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Cleans up stale unverified factors and creates a fresh enrollment.
  // Wrapped in useCallback so it can be called from both the button handler
  // and the restore useEffect. All deps are stable (memoized in AuthContext).
  const startEnrollment = useCallback(async () => {
    setError(null);
    setIsLoading(true);
    try {
      // Best-effort cleanup of stale unverified factors so Supabase
      // doesn't reject the new enrollment. If the list or removal fails
      // (e.g. flaky connection after a mobile tab refresh), we still
      // attempt enrollment — it may succeed if there's nothing to clean.
      const { factors } = await listMfaFactors();
      const unverified = factors.filter((f) => f.status === "unverified");
      for (const factor of unverified) {
        const { error: unenrollError } = await unenrollMfa(factor.id);
        if (unenrollError) {
          console.warn("Failed to clean up unverified factor:", unenrollError);
        }
      }

      const { data, error } = await enrollMfa();
      if (error) {
        // Clear the pending marker so the auto-start doesn't loop on
        // every dialog open when enrollment is stuck (e.g. server-side
        // factor limit reached). The user can still click the button to retry.
        clearPendingEnrollment();
        setError(safeErrorMessage(error));
        return;
      }
      if (data) {
        setQrCode(data.qrCode);
        setSecret(data.secret);
        setFactorId(data.factorId);
        setStep("qr");
      }
    } catch (err) {
      console.error("MFA enrollment failed:", err);
      setError(
        "Could not reach the server. Please check your connection and try again."
      );
    } finally {
      setIsLoading(false);
    }
  }, [listMfaFactors, unenrollMfa, enrollMfa]);

  // Restore pending enrollment from sessionStorage (survives mobile tab
  // refresh, e.g. when switching to the authenticator app on mobile).
  // On restore, we create a fresh enrollment so the user gets a new QR code
  // and secret together — nothing sensitive is stored in sessionStorage.
  useEffect(() => {
    if (!userId) return;
    if (!hasPendingEnrollment(userId)) return;
    startEnrollment();
  }, [userId, startEnrollment]);

  const handleCancel = useCallback(() => {
    if (factorId) {
      unenrollMfa(factorId).catch((err: unknown) => {
        console.error("Failed to unenroll MFA factor on cancel:", err);
      });
    }
    clearPendingEnrollment();
    setFactorId("");
    setQrCode("");
    setSecret("");
    setShowSecret(false);
    setCode("");
    setStep("idle");
  }, [factorId, unenrollMfa]);

  const handleButtonClick = useCallback(() => {
    savePendingEnrollment(userId);
    startEnrollment();
  }, [userId, startEnrollment]);

  const handleVerify = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);
    try {
      const { error } = await confirmMfaEnrollment(factorId, code);
      if (error) {
        setError(safeErrorMessage(error as AuthError));
        return;
      }
      clearPendingEnrollment();
      onEnrolled();
    } catch (err) {
      console.error("MFA verification failed:", err);
      setError(
        "Could not reach the server. Please check your connection and try again."
      );
    } finally {
      setIsLoading(false);
    }
  };

  if (step === "qr") {
    return (
      <form onSubmit={handleVerify} className="space-y-4" noValidate>
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Scan this QR code with your authenticator app (Google Authenticator,
            Authy, 1Password, etc.), then enter the 6-digit code it generates.
          </p>
          {showSecret ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                Enter this key manually in your authenticator app:
              </p>
              <code className="block break-all rounded-md border bg-muted px-3 py-2 text-center font-mono text-sm tracking-wide sm:tracking-widest select-all">
                {secret}
              </code>
              <button
                type="button"
                onClick={() => setShowSecret(false)}
                className="text-sm text-primary underline-offset-4 hover:underline"
              >
                Show QR code
              </button>
            </div>
          ) : (
            <div className="space-y-2">
              <div className="flex justify-center">
                {/* QR code is a data URI SVG from Supabase */}
                <img
                  src={qrCode}
                  alt="Scan this QR code with your authenticator app"
                  className="h-48 w-48 rounded-md border bg-white p-2"
                />
              </div>
              <button
                type="button"
                onClick={() => setShowSecret(true)}
                className="text-sm text-primary underline-offset-4 hover:underline"
              >
                Can&apos;t scan this?
              </button>
            </div>
          )}
        </div>
        <OtpCodeInput
          id="mfa-enroll-code"
          label="Verification Code"
          variant="totp"
          value={code}
          onChange={setCode}
          autoFocus
        />
        <FormError error={error} />
        <div className="flex gap-2">
          <Button type="submit" className="flex-1" disabled={isLoading}>
            Verify &amp; Enable
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
        </div>
      </form>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Add an extra layer of security by requiring a code from an authenticator
        app when you sign in. This protects your account even if someone gains
        access to your email.
      </p>
      <FormError error={error} />
      <Button onClick={handleButtonClick} disabled={isLoading || !userId}>
        Set Up Authenticator App
      </Button>
    </div>
  );
}
