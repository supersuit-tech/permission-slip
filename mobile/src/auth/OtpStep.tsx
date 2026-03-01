import { useState } from "react";
import {
  KeyboardAvoidingView,
  Platform,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from "react-native";
import type { AuthError } from "@supabase/supabase-js";
import { useFormSubmit } from "./useFormSubmit";

interface OtpStepProps {
  email: string;
  onVerify: (code: string) => Promise<{ error: AuthError | null }>;
  onBack: () => void;
}

export default function OtpStep({ email, onVerify, onBack }: OtpStepProps) {
  const [otpCode, setOtpCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();

  const submit = () => handleSubmit(() => onVerify(otpCode));

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <View style={styles.content}>
        <Text style={styles.title}>Check your email</Text>
        <Text style={styles.subtitle}>
          Enter the code sent to <Text style={styles.bold}>{email}</Text>
        </Text>

        <View style={styles.field}>
          <Text style={styles.label}>Code</Text>
          <TextInput
            style={styles.input}
            value={otpCode}
            onChangeText={setOtpCode}
            placeholder="000000"
            placeholderTextColor="#9CA3AF"
            keyboardType="number-pad"
            autoComplete="one-time-code"
            maxLength={6}
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
        </View>

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <View style={styles.buttonRow}>
          <TouchableOpacity
            style={[
              styles.button,
              styles.primaryButton,
              isSubmitting && styles.buttonDisabled,
            ]}
            onPress={submit}
            disabled={isSubmitting || otpCode.length < 6}
          >
            <Text style={styles.primaryButtonText}>
              {isSubmitting ? "Verifying..." : "Verify"}
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.button, styles.outlineButton]}
            onPress={onBack}
            disabled={isSubmitting}
          >
            <Text style={styles.outlineButtonText}>Back</Text>
          </TouchableOpacity>
        </View>
      </View>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#fff",
  },
  content: {
    flex: 1,
    justifyContent: "center",
    paddingHorizontal: 24,
  },
  title: {
    fontSize: 28,
    fontWeight: "700",
    color: "#111827",
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 15,
    color: "#6B7280",
    marginBottom: 32,
  },
  bold: {
    fontWeight: "600",
    color: "#111827",
  },
  field: {
    marginBottom: 16,
  },
  label: {
    fontSize: 14,
    fontWeight: "500",
    color: "#374151",
    marginBottom: 6,
  },
  input: {
    borderWidth: 1,
    borderColor: "#D1D5DB",
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 24,
    fontWeight: "600",
    color: "#111827",
    backgroundColor: "#fff",
    letterSpacing: 8,
    textAlign: "center",
  },
  error: {
    color: "#DC2626",
    fontSize: 14,
    marginBottom: 12,
  },
  buttonRow: {
    flexDirection: "row",
    gap: 12,
    marginTop: 8,
  },
  button: {
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: "center",
  },
  primaryButton: {
    flex: 1,
    backgroundColor: "#111827",
  },
  primaryButtonText: {
    color: "#fff",
    fontSize: 16,
    fontWeight: "600",
  },
  outlineButton: {
    borderWidth: 1,
    borderColor: "#D1D5DB",
    paddingHorizontal: 20,
  },
  outlineButtonText: {
    color: "#374151",
    fontSize: 16,
    fontWeight: "500",
  },
  buttonDisabled: {
    opacity: 0.5,
  },
});
