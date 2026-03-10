import { useState, useCallback } from "react";
import { useAuth } from "./AuthContext";
import { useCooldown } from "./useCooldown";
import EmailStep from "./EmailStep";
import OtpStep from "./OtpStep";
import CheckEmailStep from "./CheckEmailStep";

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
      <OtpStep
        email={email}
        onVerify={(code) => verifyOtp(email, code)}
        onBack={() => setStep("email")}
        onResend={handleResend}
        resendCooldownSeconds={cooldown.secondsLeft}
      />
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
