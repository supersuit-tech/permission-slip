import { useState, useCallback, lazy, Suspense } from "react";
import { useAuth } from "./AuthContext";
import { useCooldown } from "./useCooldown";
import EmailStep from "./EmailStep";
import CheckEmailStep from "./CheckEmailStep";
import AuthLayout from "./AuthLayout";

// Lazy-load OtpStep so it (and its dev-only dependencies like OtpCodeInput,
// DevOnly, Mailpit auto-fill) are tree-shaken from the production bundle.
// In production, the "otp" step is never reached.
const OtpStep = lazy(() => import("./OtpStep"));

type Step = "email" | "otp" | "check-email";

export default function LoginPage() {
  const { sendOtp, verifyOtp } = useAuth();
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const cooldown = useCooldown();

  const handleSendSuccess = (inputEmail: string) => {
    setEmail(inputEmail);
    setStep(import.meta.env.DEV ? "otp" : "check-email");
    cooldown.start();
  };

  const handleResend = useCallback(async () => {
    const result = await sendOtp(email);
    if (!result.error || result.error.code === "over_email_send_rate_limit") {
      cooldown.start();
    }
    return result;
  }, [sendOtp, email, cooldown.start]);

  if (step === "otp") {
    return (
      <Suspense fallback={<AuthLayout>{null}</AuthLayout>}>
        <OtpStep
          email={email}
          onVerify={(code) => verifyOtp(email, code)}
          onBack={() => setStep("email")}
          onResend={handleResend}
          resendCooldownSeconds={cooldown.secondsLeft}
        />
      </Suspense>
    );
  }

  if (step === "check-email") {
    return (
      <CheckEmailStep
        email={email}
        onBack={() => setStep("email")}
        onResend={handleResend}
        resendCooldownSeconds={cooldown.secondsLeft}
      />
    );
  }

  return (
    <EmailStep
      onSubmit={async (inputEmail) => {
        const result = await sendOtp(inputEmail);
        if (!result.error) {
          handleSendSuccess(inputEmail);
        }
        return result;
      }}
    />
  );
}
