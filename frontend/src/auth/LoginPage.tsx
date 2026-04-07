import { useState, useCallback } from "react";
import { useAuth } from "./AuthContext";
import EmailStep from "./EmailStep";
import OtpStep from "./OtpStep";
import PasswordStep from "./PasswordStep";

type Step = "email" | "otp" | "password";

export default function LoginPage() {
  const { sendOtp, verifyOtp, signInWithPassword } = useAuth();
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");

  const handleSendSuccess = (inputEmail: string) => {
    setEmail(inputEmail);
    setStep("otp");
  };

  const handleResend = useCallback(async () => {
    return sendOtp(email);
  }, [sendOtp, email]);

  if (step === "password") {
    return (
      <PasswordStep
        email={email}
        onSubmit={(password) => signInWithPassword(email, password)}
        onBack={() => setStep("email")}
      />
    );
  }

  if (step === "otp") {
    return (
      <OtpStep
        email={email}
        onVerify={(code) => verifyOtp(email, code)}
        onBack={() => setStep("email")}
        onResend={handleResend}
        onUsePassword={() => setStep("password")}
      />
    );
  }

  return (
    <EmailStep
      onSubmit={async (inputEmail) => {
        const result = await sendOtp(inputEmail);
        // Advance even on over_email_send_rate_limit — the first email was
        // already sent. Blocking here just tempts users to retry and dig
        // deeper into Supabase's per-email cooldown.
        if (
          !result.error ||
          result.error.code === "over_email_send_rate_limit"
        ) {
          handleSendSuccess(inputEmail);
          return { error: null };
        }
        return result;
      }}
      onUsePassword={(inputEmail) => {
        setEmail(inputEmail);
        setStep("password");
      }}
    />
  );
}
