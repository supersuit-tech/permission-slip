import { useState } from "react";
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
import { useAuth } from "../auth/AuthContext";
import { useFormSubmit } from "../auth/useFormSubmit";
import { authStyles } from "../auth/styles";
import { colors } from "../theme/colors";
import validation from "../lib/validation";

type Mode = "login" | "signup";

export default function LoginScreen() {
  const { signInWithPassword, signUpWithPassword } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { error, isSubmitting, handleSubmit } = useFormSubmit();

  const submit = () =>
    handleSubmit(async () => {
      const fn = mode === "login" ? signInWithPassword : signUpWithPassword;
      return fn(email.trim(), password);
    });

  const submitLabel =
    mode === "login"
      ? isSubmitting
        ? "Signing in..."
        : "Sign in"
      : isSubmitting
        ? "Creating account..."
        : "Create account";

  const a11ySubmit =
    mode === "login"
      ? isSubmitting
        ? "Signing in"
        : "Sign in"
      : isSubmitting
        ? "Creating account"
        : "Create account";

  return (
    <KeyboardAvoidingView
      style={authStyles.container}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <Pressable style={authStyles.content} onPress={Keyboard.dismiss}>
        <Text style={authStyles.title}>Permission Slip</Text>
        <Text style={authStyles.subtitle}>
          {mode === "login"
            ? "Sign in with your email and password."
            : "Create an account with your email and password."}
        </Text>

        <View style={localStyles.segment}>
          <TouchableOpacity
            testID="mode-login"
            accessibilityLabel="Sign in mode"
            accessibilityRole="button"
            style={[
              localStyles.segmentBtn,
              mode === "login" && localStyles.segmentBtnActive,
            ]}
            onPress={() => setMode("login")}
          >
            <Text
              style={[
                localStyles.segmentText,
                mode === "login" && localStyles.segmentTextActive,
              ]}
            >
              Sign in
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            testID="mode-signup"
            accessibilityLabel="Create account mode"
            accessibilityRole="button"
            style={[
              localStyles.segmentBtn,
              mode === "signup" && localStyles.segmentBtnActive,
            ]}
            onPress={() => setMode("signup")}
          >
            <Text
              style={[
                localStyles.segmentText,
                mode === "signup" && localStyles.segmentTextActive,
              ]}
            >
              Create account
            </Text>
          </TouchableOpacity>
        </View>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Email</Text>
          <TextInput
            testID="email-input"
            accessibilityLabel="Email"
            style={authStyles.input}
            value={email}
            onChangeText={setEmail}
            placeholder="you@example.com"
            placeholderTextColor={colors.gray400}
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
            autoComplete="email"
            editable={!isSubmitting}
          />
        </View>

        <View style={authStyles.field}>
          <Text style={authStyles.label}>Password</Text>
          <TextInput
            testID="password-input"
            accessibilityLabel="Password"
            style={authStyles.input}
            value={password}
            onChangeText={setPassword}
            placeholder="Enter password"
            placeholderTextColor={colors.gray400}
            secureTextEntry
            autoComplete={mode === "login" ? "password" : "password-new"}
            editable={!isSubmitting}
            onSubmitEditing={submit}
            returnKeyType="go"
          />
          <Text style={localStyles.hint}>
            At least {validation.password.minLength} characters.
          </Text>
        </View>

        {error ? (
          <Text testID="login-error" style={authStyles.error}>
            {error}
          </Text>
        ) : null}

        <TouchableOpacity
          testID="login-submit"
          accessibilityLabel={a11ySubmit}
          accessibilityRole="button"
          style={[
            authStyles.button,
            authStyles.primaryButton,
            (isSubmitting || email.trim().length === 0 || password.length < validation.password.minLength) &&
              authStyles.buttonDisabled,
          ]}
          onPress={submit}
          disabled={
            isSubmitting ||
            email.trim().length === 0 ||
            password.length < validation.password.minLength
          }
        >
          <Text style={authStyles.primaryButtonText}>{submitLabel}</Text>
        </TouchableOpacity>
      </Pressable>
    </KeyboardAvoidingView>
  );
}

const localStyles = StyleSheet.create({
  segment: {
    flexDirection: "row",
    marginBottom: 24,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.gray300,
    overflow: "hidden",
  },
  segmentBtn: {
    flex: 1,
    paddingVertical: 10,
    alignItems: "center",
    backgroundColor: colors.white,
  },
  segmentBtnActive: {
    backgroundColor: colors.primary,
  },
  segmentText: {
    fontSize: 14,
    fontWeight: "600",
    color: colors.gray500,
  },
  segmentTextActive: {
    color: colors.white,
  },
  hint: {
    marginTop: 6,
    fontSize: 12,
    color: colors.gray500,
  },
});
