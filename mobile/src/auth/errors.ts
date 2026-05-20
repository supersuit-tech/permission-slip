import type { AuthError } from "./types";

const SAFE_ERROR_MESSAGES: Record<string, string> = {
  invalid_credentials: "Invalid email or password. Please try again.",
  invalid_token: "Invalid email or password. Please try again.",
  over_request_rate_limit:
    "Too many attempts. Please wait a moment and try again.",
  request_failed: "Something went wrong. Please try again.",
  logout_failed: "Sign out failed. Please try again.",
  invalid_response: "Something went wrong. Please try again.",
  network_unreachable:
    "Couldn't reach the server. Check the server URL and your connection.",
};

export function safeErrorMessage(error: AuthError): string {
  if (error.code) {
    const message = SAFE_ERROR_MESSAGES[error.code];
    if (message) return message;
  }
  return "Something went wrong. Please try again.";
}

export function createAuthError(
  code: string,
  message: string,
  status: number,
): AuthError {
  return {
    message,
    name: "AuthApiError",
    status,
    code,
  };
}
