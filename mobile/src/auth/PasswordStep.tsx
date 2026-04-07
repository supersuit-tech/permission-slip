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

interface PasswordStepProps {
  email: string;
  onSubmit: (password: string) => Promise<{ error: AuthError | null }>;
  onBack: () => void;
}

export default function PasswordStep({
  email,
  onSubmit,
  onBack,
}: PasswordStepProps) {
  const [password, setPassword] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();
  const inputRef = useRef<TextInput>(null);

  useEffect(() => {
    const timer = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(timer);
  }, []);

  const submit = () => handleSubmit(() => onSubmit(password));

  return (
    <KeyboardAvoidingView
      style={authStyles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <Pressable style={authStyles.content} onPress={Keyboard.dismiss}>
        <Text style={authStyles.title}>Sign in</Text>
        <Text style={authStyles.subtitle}>
          Sign in as{" "}
          <Text style={localStyles.bold}>{email}</Text>
        </Text>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Password</Text>
          <TextInput
            ref={inputRef}
            testID="password-input"
            accessibilityLabel="Password"
            style={authStyles.input}
            value={password}
            onChangeText={setPassword}
            placeholder="Enter password"
            placeholderTextColor={colors.gray400}
            secureTextEntry
            autoComplete="password"
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
        </View>

        {error ? (
          <Text testID="password-error" style={authStyles.error}>
            {error}
          </Text>
        ) : null}

        <View style={localStyles.buttonRow}>
          <TouchableOpacity
            testID="password-submit"
            accessibilityLabel={isSubmitting ? "Signing in" : "Sign in"}
            accessibilityRole="button"
            style={[
              authStyles.button,
              authStyles.primaryButton,
              localStyles.flexButton,
              (isSubmitting || password.length === 0) &&
                authStyles.buttonDisabled,
            ]}
            onPress={submit}
            disabled={isSubmitting || password.length === 0}
          >
            <Text style={authStyles.primaryButtonText}>
              {isSubmitting ? "Signing in..." : "Sign In"}
            </Text>
          </TouchableOpacity>

          <TouchableOpacity
            testID="password-back"
            accessibilityLabel="Go back to email"
            accessibilityRole="button"
            style={[authStyles.button, localStyles.outlineButton]}
            onPress={onBack}
            disabled={isSubmitting}
          >
            <Text style={localStyles.outlineButtonText}>Back</Text>
          </TouchableOpacity>
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
