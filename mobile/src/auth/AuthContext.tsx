import {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { AuthError, Session, User } from "@supabase/supabase-js";
import { supabase } from "../lib/supabaseClient";
import type { AuthStatus, AuthState } from "./types";
import { AuthContext } from "./authContext";
import { tryAppReviewLogin } from "./appReviewAuth";

/**
 * Constructs a synthetic AuthError for cases where we need to generate an
 * error client-side (e.g. missing factors). Uses `as AuthError` because
 * Supabase doesn't export the AuthApiError class from @supabase/supabase-js.
 */
function createAuthError(
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

/**
 * Races `request` against a timeout. Rejects with an AuthError on timeout.
 */
async function withTimeout<T>(request: Promise<T>, ms: number): Promise<T> {
  let timerId: ReturnType<typeof setTimeout> | undefined;
  const timeoutPromise = new Promise<never>((_, reject) => {
    timerId = setTimeout(
      () => reject(new Error("Request timed out")),
      ms
    );
  });
  try {
    return await Promise.race([request, timeoutPromise]);
  } finally {
    clearTimeout(timerId);
  }
}

/**
 * Decode a JWT payload without verification (we trust the token — it came
 * from the Supabase client). Returns null on any parsing error.
 *
 * Uses atob which is available in React Native's Hermes engine.
 */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const seg = token.split(".")[1];
    if (!seg) return null;
    const base64 = seg.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    return JSON.parse(atob(padded));
  } catch {
    return null;
  }
}

/**
 * Provides Supabase auth state to the component tree. Mirrors the web
 * frontend's AuthProvider pattern — session, user, and authStatus are
 * resolved from `onAuthStateChange`, and AAL is determined from the JWT
 * payload (with a Supabase API fallback) to detect MFA requirements.
 *
 * Auth tokens are persisted in the platform's secure keychain via
 * `expo-secure-store` (configured in `supabaseClient.ts`).
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authStatus, setAuthStatus] = useState<AuthStatus>("loading");

  // Monotonic counter to discard stale AAL results. Each auth event
  // increments the version; if a newer event fires while the previous
  // AAL check is still in-flight, the older callback sees a mismatch
  // and skips setting state.
  const authVersionRef = useRef(0);

  useEffect(() => {
    // onAuthStateChange fires INITIAL_SESSION immediately with the current
    // session, so a separate getSession() call is unnecessary.
    const {
      data: { subscription },
    } = supabase.auth.onAuthStateChange(async (_event, newSession) => {
      const version = ++authVersionRef.current;
      const isStale = () => version !== authVersionRef.current;

      setSession(newSession);
      setUser(newSession?.user ?? null);

      if (!newSession) {
        setAuthStatus("unauthenticated");
        return;
      }

      // Determine AAL from the JWT to check if MFA is required.
      const jwt = decodeJwtPayload(newSession.access_token);
      if (jwt && typeof jwt.aal === "string") {
        const hasVerifiedTotp =
          newSession.user?.factors?.some(
            (f) => f.status === "verified" && f.factor_type === "totp"
          ) ?? false;

        if (jwt.aal === "aal1" && hasVerifiedTotp) {
          setAuthStatus("mfa_required");
        } else {
          setAuthStatus("authenticated");
        }
        return;
      }

      // Fallback: JWT couldn't be decoded — use the Supabase API.
      const { data: aal, error: aalError } =
        await supabase.auth.mfa.getAuthenticatorAssuranceLevel();

      if (isStale()) return;

      if (aalError) {
        // If we can't determine AAL, be safe and require MFA if factors exist.
        setAuthStatus("mfa_required");
        return;
      }

      if (aal?.currentLevel === "aal1" && aal?.nextLevel === "aal2") {
        setAuthStatus("mfa_required");
      } else {
        setAuthStatus("authenticated");
      }
    });

    return () => subscription.unsubscribe();
  }, []);

  /**
   * Calls challengeAndVerify and promotes authStatus to "authenticated" on
   * success. We eagerly set "authenticated" rather than waiting for
   * onAuthStateChange because Supabase may not re-fire the event for an
   * AAL promotion within the same session.
   */
  const challengeAndVerifyTotp = useCallback(
    async (factorId: string, code: string) => {
      const { error } = await withTimeout(
        supabase.auth.mfa.challengeAndVerify({ factorId, code }),
        10000
      ).catch((err) => ({
        error: createAuthError(
          "unknown",
          err instanceof Error ? err.message : String(err),
          500
        ),
      }));
      if (!error) {
        setAuthStatus("authenticated");
      }
      return { error: error ?? null };
    },
    []
  );

  const sendOtp = useCallback(async (email: string) => {
    const { error } = await supabase.auth.signInWithOtp({ email });
    return { error: error ?? null };
  }, []);

  const verifyOtp = useCallback(async (email: string, token: string) => {
    const { error } = await supabase.auth.verifyOtp({
      email,
      token,
      type: "email",
    });
    if (!error) return { error: null };

    // If Supabase OTP verification failed, try the backend's app-review-login
    // endpoint as a fallback. This allows App Store reviewers to sign in with
    // a pre-configured static OTP code.
    const reviewResult = await tryAppReviewLogin(email, token);
    if (!reviewResult.error) return { error: null };

    // Return the original Supabase error, not the review fallback error,
    // since most users won't have review credentials configured.
    return { error };
  }, []);

  const signOut = useCallback(async () => {
    // Use "local" scope so signing out on mobile doesn't invalidate
    // the user's web session. The web app uses "global" scope.
    const { error } = await supabase.auth.signOut({ scope: "local" });
    return { error: error ?? null };
  }, []);

  const verifyMfa = useCallback(
    async (code: string) => {
      const totpFactor = (user?.factors ?? []).find(
        (f) => f.factor_type === "totp" && f.status === "verified"
      );
      if (!totpFactor) {
        return {
          error: createAuthError(
            "mfa_factor_not_found",
            "No authenticator found. Please re-enroll.",
            400
          ),
        };
      }
      return challengeAndVerifyTotp(totpFactor.id, code);
    },
    [challengeAndVerifyTotp, user]
  );

  const value = useMemo<AuthState>(
    () => ({
      session,
      user,
      authStatus,
      sendOtp,
      verifyOtp,
      verifyMfa,
      signOut,
    }),
    [session, user, authStatus, sendOtp, verifyOtp, verifyMfa, signOut]
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
