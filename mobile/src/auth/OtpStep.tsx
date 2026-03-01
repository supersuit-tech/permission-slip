import { useCallback, useEffect, useRef, useState } from "react";
import {
  Keyboard,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from "react-native";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";
import { authStyles } from "./styles";
import { colors } from "../theme/colors";

interface OtpStepProps {
  email: string;
  onVerify: (code: string) => Promise<{ error: AuthError | null }>;
  onResend: () => Promise<{ error: AuthError | null }>;
  onBack: () => void;
}

const RESEND_COOLDOWN_SECONDS = 30;

type ResendStatus = "idle" | "sent" | "failed";

export default function OtpStep({
  email,
  onVerify,
  onResend,
  onBack,
}: OtpStepProps) {
  const [otpCode, setOtpCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const inputRef = useRef<TextInput>(null);
  const [resendCooldown, setResendCooldown] = useState(0);
  const [resendStatus, setResendStatus] = useState<ResendStatus>("idle");
  const resendTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  // Auto-focus the OTP input on mount.
  useEffect(() => {
    const timer = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(timer);
  }, []);

  // Countdown timer for resend cooldown. Only re-creates when cooldown
  // transitions between active/inactive (not on every tick) to avoid
  // tearing down and re-creating the interval each second.
  const isCooldownActive = resendCooldown > 0;
  useEffect(() => {
    if (!isCooldownActive) return;
    const interval = setInterval(
      () =>
        setResendCooldown((prev) => {
          if (prev <= 1) {
            clearInterval(interval);
            return 0;
          }
          return prev - 1;
        }),
      1000
    );
    return () => clearInterval(interval);
  }, [isCooldownActive]);

  // Clean up the resend feedback timeout on unmount.
  useEffect(() => {
    return () => {
      if (resendTimerRef.current) clearTimeout(resendTimerRef.current);
    };
  }, []);

  const submit = () => handleSubmit(() => onVerify(otpCode));

  const handleResend = useCallback(async () => {
    setResendStatus("idle");
    const { error: resendError } = await onResend();
    if (resendError) {
      setResendStatus("failed");
    } else {
      setResendStatus("sent");
      setResendCooldown(RESEND_COOLDOWN_SECONDS);
      // Clear the success message after a few seconds.
      if (resendTimerRef.current) clearTimeout(resendTimerRef.current);
      resendTimerRef.current = setTimeout(
        () => setResendStatus("idle"),
        3000
      );
    }
  }, [onResend]);

  return (
    <KeyboardAvoidingView
      style={authStyles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <Pressable style={authStyles.content} onPress={Keyboard.dismiss}>
        <Text style={authStyles.title}>Check your email</Text>
        <Text style={authStyles.subtitle}>
          Enter the code sent to{" "}
          <Text style={localStyles.bold}>{email}</Text>
        </Text>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Code</Text>
          <TextInput
            ref={inputRef}
            testID="otp-input"
            accessibilityLabel="Verification code"
            style={[authStyles.input, localStyles.otpInput]}
            value={otpCode}
            onChangeText={setOtpCode}
            placeholder="000000"
            placeholderTextColor={colors.gray400}
            keyboardType="number-pad"
            autoComplete="one-time-code"
            maxLength={6}
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
        </View>

        {error ? (
          <Text testID="otp-error" style={authStyles.error}>
            {error}
          </Text>
        ) : null}

        <View style={localStyles.buttonRow}>
          <TouchableOpacity
            testID="otp-submit"
            accessibilityLabel={isSubmitting ? "Verifying code" : "Verify code"}
            accessibilityRole="button"
            style={[
              authStyles.button,
              authStyles.primaryButton,
              localStyles.flexButton,
              (isSubmitting || otpCode.length < 6) &&
                authStyles.buttonDisabled,
            ]}
            onPress={submit}
            disabled={isSubmitting || otpCode.length < 6}
          >
            <Text style={authStyles.primaryButtonText}>
              {isSubmitting ? "Verifying..." : "Verify"}
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            testID="otp-back"
            accessibilityLabel="Go back to email"
            accessibilityRole="button"
            style={[authStyles.button, localStyles.outlineButton]}
            onPress={onBack}
            disabled={isSubmitting}
          >
            <Text style={localStyles.outlineButtonText}>Back</Text>
          </TouchableOpacity>
        </View>

        <View style={localStyles.resendRow}>
          <TouchableOpacity
            testID="otp-resend"
            accessibilityLabel="Resend verification code"
            accessibilityRole="button"
            onPress={handleResend}
            disabled={resendCooldown > 0 || isSubmitting}
          >
            <Text
              style={[
                localStyles.resendText,
                (resendCooldown > 0 || isSubmitting) &&
                  localStyles.resendDisabled,
              ]}
            >
              {resendCooldown > 0
                ? `Resend code (${resendCooldown}s)`
                : "Resend code"}
            </Text>
          </TouchableOpacity>
          {resendStatus !== "idle" ? (
            <Text
              testID="resend-message"
              style={[
                localStyles.resendFeedback,
                resendStatus === "sent" && localStyles.resendSuccess,
              ]}
            >
              {resendStatus === "sent"
                ? "Code sent!"
                : "Failed to resend. Please try again."}
            </Text>
          ) : null}
        </View>
      </Pressable>
    </KeyboardAvoidingView>
  );
}

const localStyles = StyleSheet.create({
  bold: {
    fontWeight: "600",
    color: colors.gray900,
  },
  otpInput: {
    fontSize: 24,
    fontWeight: "600",
    letterSpacing: 8,
    textAlign: "center",
  },
  buttonRow: {
    flexDirection: "row",
    gap: 12,
    marginTop: 8,
  },
  flexButton: {
    flex: 1,
  },
  outlineButton: {
    borderWidth: 1,
    borderColor: colors.gray300,
    paddingHorizontal: 20,
  },
  outlineButtonText: {
    color: colors.gray700,
    fontSize: 16,
    fontWeight: "500",
  },
  resendRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    marginTop: 20,
    justifyContent: "center",
  },
  resendText: {
    color: colors.gray500,
    fontSize: 14,
    fontWeight: "500",
  },
  resendDisabled: {
    opacity: 0.4,
  },
  resendFeedback: {
    fontSize: 14,
    color: colors.error,
  },
  resendSuccess: {
    color: colors.success,
  },
});
