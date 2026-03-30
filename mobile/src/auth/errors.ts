import type { AuthError } from "@supabase/supabase-js";

// Maps known Supabase error codes to user-facing messages. Raw Supabase errors
// can expose internal details (provider names, rate-limit specifics, etc.), so
// we only surface messages from this allowlist and fall back to a generic
// message for anything unexpected.
const SAFE_ERROR_MESSAGES: Record<string, string> = {
  otp_expired: "Your code has expired. Please request a new one.",
  otp_disabled: "Something went wrong. Please try again.",
  over_request_rate_limit:
    "Too many attempts. Please wait a moment and try again.",
  over_email_send_rate_limit:
    "Too many requests. Please wait a moment and try again.",
  mfa_factor_not_found: "No authenticator found. Please re-enroll.",
  mfa_verification_failed:
    "Invalid code. Please check your authenticator app and try again.",
  mfa_challenge_expired: "Verification timed out. Please try again.",
};

/** Returns a user-safe message for a Supabase AuthError by matching its code
 *  against a known allowlist. Unknown codes get a generic fallback. */
export function safeErrorMessage(error: AuthError): string {
  if (error.code) {
    const message = SAFE_ERROR_MESSAGES[error.code];
    if (message) return message;
  }
  return "Something went wrong. Please try again.";
}
