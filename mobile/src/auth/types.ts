import type { AuthError, Session, User } from "@supabase/supabase-js";

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

export interface AuthState {
  session: Session | null;
  user: User | null;
  authStatus: AuthStatus;

  /** Send a magic-link OTP to the given email address. */
  sendOtp: (email: string) => Promise<AuthResult>;
  /** Verify an email OTP token to complete sign-in. */
  verifyOtp: (email: string, token: string) => Promise<AuthResult>;
  /** Sign the user out of this device only (local scope). */
  signOut: () => Promise<AuthResult>;
}
