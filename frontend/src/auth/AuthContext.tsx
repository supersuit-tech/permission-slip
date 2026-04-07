import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { Factor, Session, User } from "@supabase/supabase-js";
import { supabase } from "../lib/supabaseClient";
import { tryAppReviewLogin } from "./appReviewAuth";
import { createAuthError } from "./errors";
import type { AuthStatus, AuthState } from "./types";

const AuthContext = createContext<AuthState | undefined>(undefined);

/**
 * Races `request` against a timeout. The timeout timer is always cleared once
 * the request settles (via `finally`), so it never lingers in the event loop
 * after the request resolves quickly.
 *
 * Rejects with an `AuthError` on timeout; re-throws any other rejection so
 * callers can handle it uniformly.
 */
async function withTimeout<T>(
  request: Promise<T>,
  ms: number
): Promise<T> {
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

/** Returns the first verified TOTP factor from a factor list, or undefined. */
function getVerifiedTotpFactor(factors?: Factor[]): Factor | undefined {
  return factors?.find((f) => f.status === "verified");
}

/**
 * Decode a JWT payload without verification (we trust the token — it came
 * from the Supabase client). Returns null on any parsing error.
 */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const seg = token.split(".")[1];
    if (!seg) return null;
    // Convert base64url → base64: replace URL-safe chars and add padding.
    const base64 = seg.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    return JSON.parse(atob(padded));
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [user, setUser] = useState<User | null>(null);
  const [authStatus, setAuthStatus] = useState<AuthStatus>("loading");

  // Monotonic counter to discard stale AAL results. Each auth event
  // increments the version; if a newer event fires while the previous
  // AAL check is still in-flight, the older callback sees a mismatch
  // and skips setting state.
  const authVersionRef = useRef(0);

  /**
   * Calls challengeAndVerify and promotes authStatus to "authenticated" on
   * success. Shared by verifyMfa (login challenge) and confirmMfaEnrollment.
   *
   * We eagerly set "authenticated" rather than waiting for onAuthStateChange
   * because Supabase may not re-fire the event for an AAL promotion within
   * the same session.
   */
  const challengeAndVerifyTotp = useCallback(
    async (factorId: string, code: string) => {
      const { error } = await withTimeout(
        supabase.auth.mfa.challengeAndVerify({ factorId, code }),
        10000
      ).catch((err) => ({
        error: createAuthError("unknown", err instanceof Error ? err.message : String(err), 500),
      }));
      if (!error) {
        setAuthStatus("authenticated");
      }
      return { error: error ?? null };
    },
    []
  );

  useEffect(() => {
    // onAuthStateChange fires INITIAL_SESSION immediately with the current
    // session, so a separate getSession() call is unnecessary and would
    // introduce a race condition.
    const {
      data: { subscription },
    } = supabase.auth.onAuthStateChange(async (_event, session) => {
      const version = ++authVersionRef.current;
      const isStale = () => version !== authVersionRef.current;

      setSession(session);
      setUser(session?.user ?? null);

      if (!session) {
        setAuthStatus("unauthenticated");
        return;
      }

      // Try to determine AAL from the JWT first. This avoids calling
      // getAuthenticatorAssuranceLevel() which acquires the Supabase
      // internal lock and deadlocks when this callback fires during
      // challengeAndVerify's lock drain loop.
      const jwt = decodeJwtPayload(session.access_token);
      if (jwt && typeof jwt.aal === "string") {
        const currentAal = jwt.aal;
        // Check if user has verified TOTP factors requiring AAL2.
        const hasVerifiedTotp =
          session.user?.factors?.some(
            (f) => f.status === "verified" && f.factor_type === "totp"
          ) ?? false;

        if (currentAal === "aal1" && hasVerifiedTotp) {
          setAuthStatus("mfa_required");
        } else {
          setAuthStatus("authenticated");
        }
        return;
      }

      // Fallback: JWT couldn't be decoded (e.g. mock tokens in tests).
      // Use the Supabase API which acquires a lock — safe here since
      // this path only runs outside of challengeAndVerify.
      const { data: aal, error: aalError } =
        await supabase.auth.mfa.getAuthenticatorAssuranceLevel();

      if (isStale()) return;

      if (aalError) {
        const { data: factors, error: factorsError } =
          await supabase.auth.mfa.listFactors();

        if (isStale()) return;

        if (factorsError) {
          setAuthStatus("mfa_required");
          return;
        }
        const hasVerifiedTotp = !!getVerifiedTotpFactor(factors?.totp);
        setAuthStatus(hasVerifiedTotp ? "mfa_required" : "authenticated");
        return;
      }

      if (
        aal &&
        aal.currentLevel === "aal1" &&
        aal.nextLevel === "aal2"
      ) {
        setAuthStatus("mfa_required");
      } else {
        setAuthStatus("authenticated");
      }
    });

    return () => subscription.unsubscribe();
  }, []);

  // -- Memoized auth operations --------------------------------------------------
  // All functions below only reference `supabase` (module-level singleton) and
  // stable React state setters, so they are safe to memoize with empty deps.
  // Keeping them referentially stable prevents downstream useCallback/useEffect
  // chains from re-firing on every AuthProvider render (e.g. MfaSettingsDialog's
  // loadFactors → listMfaFactors dependency).

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

  const updateEmail = useCallback(async (newEmail: string) => {
    const { error } = await supabase.auth.updateUser({ email: newEmail });
    return { error: error ?? null };
  }, []);

  const signOut = useCallback(async () => {
    const { error } = await supabase.auth.signOut({ scope: "global" });
    return { error: error ?? null };
  }, []);

  const verifyMfa = useCallback(
    async (code: string) => {
      // Read from user.factors (already in React state) instead of calling
      // supabase.auth.mfa.listFactors(), which internally calls getUser() and
      // acquires the Supabase internal lock — causing lock contention with the
      // onAuthStateChange listener that fires during session init.
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

  const enrollMfa = useCallback(async () => {
    const { data, error } = await withTimeout(
      supabase.auth.mfa.enroll({
        factorType: "totp",
        friendlyName: "Authenticator App",
      }),
      10000
    ).catch((err) => ({
      data: null,
      error: createAuthError("unknown", err instanceof Error ? err.message : String(err), 500),
    }));
    if (error) return { data: null, error };

    // Guard against partial/unexpected response shapes from GoTrue.
    const qrCode = data?.totp?.qr_code;
    const secret = data?.totp?.secret;
    if (!data || !qrCode || !secret) {
      console.error("MFA enroll: unexpected response shape", data);
      return {
        data: null,
        error: createAuthError(
          "mfa_enroll_failed",
          "Failed to start authenticator setup. Please try again.",
          500
        ),
      };
    }
    return {
      data: { factorId: data.id, qrCode, secret },
      error: null,
    };
  }, []);

  const confirmMfaEnrollment = useCallback(
    async (factorId: string, code: string) => {
      return challengeAndVerifyTotp(factorId, code);
    },
    [challengeAndVerifyTotp]
  );

  const unenrollMfa = useCallback(async (factorId: string) => {
    const { error } = await withTimeout(
      supabase.auth.mfa.unenroll({ factorId }),
      10000
    ).catch((err) => ({
      error: createAuthError("unknown", err instanceof Error ? err.message : String(err), 500),
    }));
    return { error: error ?? null };
  }, []);

  const listMfaFactors = useCallback(async () => {
    // Read from user.factors (already in React state) instead of calling
    // supabase.auth.mfa.listFactors(), which internally calls getUser() and
    // acquires the Supabase internal lock — causing lock contention with the
    // onAuthStateChange listener that fires during session init, hanging the
    // UI for up to 10 seconds.
    //
    // This mirrors exactly what the Supabase SDK's own _listFactors() does
    // internally: it reads user.factors from the already-fetched user object.
    // user.factors includes both verified and unverified factors, so
    // MfaEnrollmentFlow can still clean up stale abandoned enrollments.
    const allTotp = (user?.factors ?? []).filter(
      (f) => f.factor_type === "totp"
    );
    return { factors: allTotp, error: null };
  }, [user]);

  const value = useMemo<AuthState>(
    () => ({
      session,
      user,
      authStatus,
      sendOtp,
      verifyOtp,
      updateEmail,
      signOut,
      verifyMfa,
      enrollMfa,
      confirmMfaEnrollment,
      unenrollMfa,
      listMfaFactors,
    }),
    [
      session,
      user,
      authStatus,
      sendOtp,
      verifyOtp,
      updateEmail,
      signOut,
      verifyMfa,
      enrollMfa,
      confirmMfaEnrollment,
      unenrollMfa,
      listMfaFactors,
    ]
  );

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
