import type { AuthError } from "@supabase/supabase-js";

// Maps known Supabase error codes to user-facing messages. Raw Supabase errors
// can expose internal details (provider names, rate-limit specifics, etc.), so
// we only surface messages from this allowlist and fall back to a generic
// message for anything unexpected.
const SAFE_ERROR_MESSAGES: Record<string, string> = {
  otp_expired: "Your sign-in link has expired. Please request a new one.",
  otp_disabled: "Something went wrong. Please try again.",
  over_request_rate_limit:
    "Too many attempts. Please wait a moment and try again.",
  over_email_send_rate_limit:
    "Too many login emails sent. Please wait a few minutes and try again.",
  mfa_factor_not_found: "No authenticator found. Please re-enroll.",
  mfa_verification_failed:
    "Invalid code. Please check your authenticator app and try again.",
  mfa_challenge_expired: "Verification timed out. Please try again.",
  mfa_enroll_failed:
    "Failed to start authenticator setup. Please try again.",
};

/** Returns a user-safe message for a Supabase AuthError by matching its code
 *  against a known allowlist. Unknown codes get a generic fallback.
 *
 *  Pass `overrides` to substitute a context-specific message for a given code
 *  while still falling back to the allowlist for everything else. */
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

/**
 * Constructs a synthetic AuthError for cases where we need to generate an
 * error client-side (e.g. unexpected response shapes, missing factors).
 *
 * Uses `as AuthError` because Supabase doesn't export the AuthApiError class
 * from @supabase/supabase-js — only from the internal @supabase/auth-js
 * package. Centralizing the cast here keeps it in one auditable location.
 */
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
  } as AuthError;
}
