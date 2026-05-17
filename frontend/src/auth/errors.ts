import type { AuthError } from "./types";

// Maps known API / auth error codes to user-facing messages.
const SAFE_ERROR_MESSAGES: Record<string, string> = {
  invalid_credentials: "Invalid email or password. Please try again.",
  invalid_token: "Invalid email or password. Please try again.",
  over_request_rate_limit:
    "Too many attempts. Please wait a moment and try again.",
  request_failed: "Something went wrong. Please try again.",
  logout_failed: "Sign out failed. Please try again.",
  invalid_response: "Something went wrong. Please try again.",
  email_change_unavailable:
    "Changing your email address is not available in this version.",
};

export function safeErrorMessage(
  error: AuthError,
  overrides?: Partial<Record<string, string>>
): string {
  if (error.code) {
    const override = overrides?.[error.code];
    if (override !== undefined) return override;
    const message = SAFE_ERROR_MESSAGES[error.code];
    if (message) return message;
  }
  return "Something went wrong. Please try again.";
}

export function createAuthError(
  code: string,
  message: string,
  status: number
): AuthError {
  return {
    message,
    name: "AuthApiError",
    status,
    code,
  };
}
