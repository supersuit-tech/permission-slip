import type { AuthError, Factor, Session, User } from "@supabase/supabase-js";

/**
 * Authentication lifecycle states:
 * - `loading`           — Initial state while the session is being resolved.
 * - `unauthenticated`   — No active session.
 * - `authenticated`     — Fully authenticated (AAL2 if MFA is enrolled, AAL1 otherwise).
 * - `mfa_required`      — Session exists at AAL1 but the user has an enrolled
 *                          TOTP factor requiring a second-factor challenge.
 */
export type AuthStatus =
  | "loading"
  | "unauthenticated"
  | "authenticated"
  | "mfa_required";

/** Standard result shape for auth operations that only return an error. */
export type AuthResult = { error: AuthError | null };

export interface MfaEnrollResult {
  factorId: string;
  qrCode: string;
  secret: string;
}

export interface AuthState {
  session: Session | null;
  user: User | null;
  authStatus: AuthStatus;

  /** Send a magic-link OTP to the given email address. */
  sendOtp: (email: string) => Promise<AuthResult>;
  /** Verify an email OTP token to complete sign-in. */
  verifyOtp: (email: string, token: string) => Promise<AuthResult>;
  /** Request an email change for the current user. */
  updateEmail: (newEmail: string) => Promise<AuthResult>;
  /** Sign the user out of all sessions (global scope). */
  signOut: () => Promise<AuthResult>;
  /** Verify a TOTP code to complete MFA login (AAL1 → AAL2). */
  verifyMfa: (code: string) => Promise<AuthResult>;
  /** Begin TOTP enrollment. Returns a QR code URI for the authenticator app. */
  enrollMfa: () => Promise<{
    data: MfaEnrollResult | null;
    error: AuthError | null;
  }>;
  /** Confirm TOTP enrollment by verifying the first code from the authenticator app. */
  confirmMfaEnrollment: (
    factorId: string,
    code: string
  ) => Promise<AuthResult>;
  /** Unenroll a TOTP factor. */
  unenrollMfa: (factorId: string) => Promise<AuthResult>;
  /** List enrolled TOTP factors. */
  listMfaFactors: () => Promise<{
    factors: Factor[];
    error: AuthError | null;
  }>;
}
