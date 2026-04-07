import { useState } from "react";
import { useAuth } from "../auth/AuthContext";
import EmailStep from "../auth/EmailStep";
import OtpStep from "../auth/OtpStep";
import PasswordStep from "../auth/PasswordStep";

type Step = "email" | "otp" | "password";

export default function LoginScreen() {
  const { sendOtp, verifyOtp, signInWithPassword } = useAuth();
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");

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
        onResend={() => sendOtp(email)}
        onBack={() => setStep("email")}
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
          setEmail(inputEmail);
          setStep("otp");
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
