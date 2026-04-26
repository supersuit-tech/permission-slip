/**
 * OTP digit lengths for the web app, sourced from `shared/validation.json`
 * via `@/lib/validation` so Mailpit dev auto-fill, inputs, and submit gating
 * all use the same build-time values.
 */
import validation from "@/lib/validation";

export const EMAIL_OTP_DIGIT_LENGTH = validation.emailOtpCode.length;
export const TOTP_DIGIT_LENGTH = validation.totpCode.length;
