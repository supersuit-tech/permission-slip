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
import type { Session, User } from "@supabase/supabase-js";
import { supabase } from "../lib/supabaseClient";
import type { AuthStatus, AuthState } from "./types";

const AuthContext = createContext<AuthState | undefined>(undefined);

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
    return { error: error ?? null };
  }, []);

  const signOut = useCallback(async () => {
    const { error } = await supabase.auth.signOut({ scope: "global" });
    return { error: error ?? null };
  }, []);

  const value = useMemo<AuthState>(
    () => ({
      session,
      user,
      authStatus,
      sendOtp,
      verifyOtp,
      signOut,
    }),
    [session, user, authStatus, sendOtp, verifyOtp, signOut]
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
