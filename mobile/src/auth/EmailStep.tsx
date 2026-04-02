import { useEffect, useRef, useState } from "react";
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

interface EmailStepProps {
  onSubmit: (email: string) => Promise<{ error: AuthError | null }>;
}

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function EmailStep({ onSubmit }: EmailStepProps) {
  const [email, setEmail] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const inputRef = useRef<TextInput>(null);

  const trimmedEmail = email.trim();
  const isValidEmail = EMAIL_REGEX.test(trimmedEmail);

  // Auto-focus the email input on mount.
  useEffect(() => {
    const timer = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(timer);
  }, []);

  const submit = () => handleSubmit(() => onSubmit(trimmedEmail));

  return (
    <KeyboardAvoidingView
      style={authStyles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <Pressable style={authStyles.content} onPress={Keyboard.dismiss}>
        <View style={brandBadgeStyles.row}>
          <View style={brandBadgeStyles.badge}>
            <Text style={brandBadgeStyles.badgeText}>P</Text>
          </View>
          <Text style={[authStyles.title, { marginBottom: 0 }]}>Permission Slip</Text>
        </View>
        <Text style={authStyles.subtitle}>
          Enter your email to sign in or create an account.
        </Text>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Email</Text>
          <TextInput
            ref={inputRef}
            testID="email-input"
            accessibilityLabel="Email address"
            style={authStyles.input}
            value={email}
            onChangeText={setEmail}
            placeholder="you@example.com"
            placeholderTextColor={colors.gray400}
            keyboardType="email-address"
            autoCapitalize="none"
            autoComplete="email"
            autoCorrect={false}
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
        </View>

        {error ? (
          <Text testID="email-error" style={authStyles.error}>
            {error}
          </Text>
        ) : null}

        <TouchableOpacity
          testID="email-submit"
          accessibilityLabel={isSubmitting ? "Sending code" : "Continue"}
          accessibilityRole="button"
          style={[
            authStyles.button,
            authStyles.primaryButton,
            (isSubmitting || !isValidEmail) && authStyles.buttonDisabled,
          ]}
          onPress={submit}
          disabled={isSubmitting || !isValidEmail}
        >
          <Text style={authStyles.primaryButtonText}>
            {isSubmitting ? "Sending..." : "Continue"}
          </Text>
        </TouchableOpacity>
      </Pressable>
    </KeyboardAvoidingView>
  );
}

const brandBadgeStyles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    marginBottom: 8,
  },
  badge: {
    width: 32,
    height: 32,
    borderRadius: 8,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
  },
  badgeText: {
    color: colors.white,
    fontSize: 16,
    fontWeight: "700",
  },
});
