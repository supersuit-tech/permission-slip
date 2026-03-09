import { useState } from "react";
import { useAuth } from "./AuthContext";
import { useCooldown } from "./useCooldown";
import EmailStep from "./EmailStep";
import OtpStep from "./OtpStep";

export default function LoginPage() {
  const { sendOtp, verifyOtp } = useAuth();
  const [step, setStep] = useState<"email" | "otp">("email");
  const [email, setEmail] = useState("");
  const cooldown = useCooldown();

  if (step === "otp") {
    return (
      <OtpStep
        email={email}
        onVerify={(code) => verifyOtp(email, code)}
        onBack={() => setStep("email")}
        onResend={async () => {
          const result = await sendOtp(email);
          // Start cooldown on success or rate-limit error so the button
          // stays disabled even when the server rejects the request.
          if (!result.error || result.error.code === "over_email_send_rate_limit") {
            cooldown.start();
          }
          return result;
        }}
        resendCooldownSeconds={cooldown.secondsLeft}
      />
    );
  }

  return (
    <EmailStep
      onSubmit={async (inputEmail) => {
        const result = await sendOtp(inputEmail);
        if (!result.error) {
          setEmail(inputEmail);
          setStep("otp");
          cooldown.start();
        }
        return result;
      }}
    />
  );
}
