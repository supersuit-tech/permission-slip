import { useState } from "react";
import { useAuth } from "../auth/AuthContext";
import EmailStep from "../auth/EmailStep";
import OtpStep from "../auth/OtpStep";

export default function LoginScreen() {
  const { sendOtp, verifyOtp } = useAuth();
  const [step, setStep] = useState<"email" | "otp">("email");
  const [email, setEmail] = useState("");

  if (step === "otp") {
    return (
      <OtpStep
        email={email}
        onVerify={(code) => verifyOtp(email, code)}
        onBack={() => setStep("email")}
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
        }
        return result;
      }}
    />
  );
}
