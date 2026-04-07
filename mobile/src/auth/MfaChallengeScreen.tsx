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
import { useAuth } from "./AuthContext";
import { useFormSubmit } from "./useFormSubmit";
import { authStyles } from "./styles";
import { colors } from "../theme/colors";
import validation from "../lib/validation";

/**
 * Full-screen MFA challenge presented when authStatus is "mfa_required".
 * The user enters a 6-digit TOTP code from their authenticator app.
 * On success, authStatus transitions to "authenticated" and the navigator
 * swaps to the main app screens automatically.
 */
export default function MfaChallengeScreen() {
  const { verifyMfa, signOut } = useAuth();
  const [code, setCode] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const inputRef = useRef<TextInput>(null);

  // Auto-focus the code input on mount.
  useEffect(() => {
    const timer = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(timer);
  }, []);

  // Strip non-digit characters (matches web OtpCodeInput behavior).
  const handleChangeText = (text: string) => {
    setCode(text.replace(/\D/g, ""));
  };

  const submit = () => handleSubmit(() => verifyMfa(code));
  const totpLen = validation.totpCode.length;

  return (
    <KeyboardAvoidingView
      style={authStyles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <Pressable style={authStyles.content} onPress={Keyboard.dismiss}>
        <Text style={authStyles.title}>Two-factor authentication</Text>
        <Text style={authStyles.subtitle}>
          Enter the 6-digit code from your authenticator app to continue.
        </Text>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Authenticator Code</Text>
          <TextInput
            ref={inputRef}
            testID="mfa-code-input"
            accessibilityLabel="Authenticator code"
            style={[authStyles.input, localStyles.codeInput]}
            value={code}
            onChangeText={handleChangeText}
            placeholder="000000"
            placeholderTextColor={colors.gray400}
            keyboardType="number-pad"
            autoComplete="one-time-code"
            maxLength={totpLen}
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
        </View>

        {error ? (
          <Text testID="mfa-error" style={authStyles.error}>
            {error}
          </Text>
        ) : null}

        <View style={localStyles.buttonRow}>
          <TouchableOpacity
            testID="mfa-verify"
            accessibilityLabel={isSubmitting ? "Verifying code" : "Verify code"}
            accessibilityRole="button"
            style={[
              authStyles.button,
              authStyles.primaryButton,
              localStyles.flexButton,
              (isSubmitting || code.length < totpLen) && authStyles.buttonDisabled,
            ]}
            onPress={submit}
            disabled={isSubmitting || code.length < totpLen}
          >
            <Text style={authStyles.primaryButtonText}>
              {isSubmitting ? "Verifying..." : "Verify"}
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            testID="mfa-sign-out"
            accessibilityLabel="Sign out"
            accessibilityRole="button"
            style={[authStyles.button, localStyles.outlineButton]}
            onPress={() => signOut()}
            disabled={isSubmitting}
          >
            <Text style={localStyles.outlineButtonText}>Sign Out</Text>
          </TouchableOpacity>
        </View>
      </Pressable>
    </KeyboardAvoidingView>
  );
}

const localStyles = StyleSheet.create({
  codeInput: {
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
});
