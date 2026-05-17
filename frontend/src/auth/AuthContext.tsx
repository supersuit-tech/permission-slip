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
import { postAuth, type AuthTokenResponse } from "@/lib/authApi";
import {
  clearStoredRefreshToken,
  getStoredRefreshToken,
  setStoredRefreshToken,
} from "@/lib/authStorage";
import { createAuthError } from "./errors";
import type { AppSession, AppUser, AuthState, AuthStatus, AuthError } from "./types";

const AuthContext = createContext<AuthState | undefined>(undefined);

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const seg = token.split(".")[1];
    if (!seg) return null;
    const base64 = seg.replace(/-/g, "+").replace(/_/g, "/");
    const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
    return JSON.parse(atob(padded)) as Record<string, unknown>;
  } catch {
    return null;
  }
}

function userFromAccessToken(accessToken: string): AppUser {
  const payload = decodeJwtPayload(accessToken);
  const id = typeof payload?.sub === "string" ? payload.sub : "";
  const email = typeof payload?.email === "string" ? payload.email : undefined;
  return { id, email };
}

function toSession(data: AuthTokenResponse): AppSession {
  return {
    access_token: data.access_token,
    expires_at: data.expires_at,
    user: userFromAccessToken(data.access_token),
  };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AppSession | null>(null);
  const [user, setUser] = useState<AppUser | null>(null);
  const [authStatus, setAuthStatus] = useState<AuthStatus>("loading");
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearRefreshTimer = useCallback(() => {
    if (refreshTimerRef.current !== null) {
      clearTimeout(refreshTimerRef.current);
      refreshTimerRef.current = null;
    }
  }, []);

  const applyAuthBundle = useCallback((data: AuthTokenResponse) => {
    setStoredRefreshToken(data.refresh_token);
    const next = toSession(data);
    setSession(next);
    setUser(next.user);
  }, []);

  const refreshAccessToken = useCallback(async () => {
    const rt = getStoredRefreshToken();
    if (!rt) {
      clearStoredRefreshToken();
      setSession(null);
      setUser(null);
      setAuthStatus("unauthenticated");
      return { error: createAuthError("invalid_token", "Not signed in", 401) };
    }
    const { data, error } = await postAuth("refresh", { refresh_token: rt });
    if (error || !data) {
      clearStoredRefreshToken();
      setSession(null);
      setUser(null);
      setAuthStatus("unauthenticated");
      return { error: error ?? createAuthError("invalid_token", "Session expired", 401) };
    }
    applyAuthBundle(data);
    setAuthStatus("authenticated");
    return { error: null };
  }, [applyAuthBundle]);

  // Bootstrap session from refresh token in localStorage.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const rt = getStoredRefreshToken();
      if (!rt) {
        if (!cancelled) setAuthStatus("unauthenticated");
        return;
      }
      const { data, error } = await postAuth("refresh", { refresh_token: rt });
      if (cancelled) return;
      if (error || !data) {
        clearStoredRefreshToken();
        setAuthStatus("unauthenticated");
        return;
      }
      applyAuthBundle(data);
      setAuthStatus("authenticated");
    })();
    return () => {
      cancelled = true;
    };
  }, [applyAuthBundle]);

  // Schedule background refresh ~1 minute before access token expiry.
  useEffect(() => {
    clearRefreshTimer();
    if (!session?.expires_at) return;

    const expMs = new Date(session.expires_at).getTime();
    if (Number.isNaN(expMs)) return;

    const delay = Math.max(10_000, expMs - Date.now() - 60_000);
    refreshTimerRef.current = setTimeout(() => {
      void refreshAccessToken();
    }, delay);

    return () => {
      clearRefreshTimer();
    };
  }, [session?.access_token, session?.expires_at, clearRefreshTimer, refreshAccessToken]);

  const signInWithPassword = useCallback(async (email: string, password: string) => {
    const { data, error } = await postAuth("login", { email, password });
    if (error || !data) {
      return { error: error ?? createAuthError("request_failed", "Login failed", 500) };
    }
    applyAuthBundle(data);
    setAuthStatus("authenticated");
    return { error: null };
  }, [applyAuthBundle]);

  const signUpWithPassword = useCallback(async (email: string, password: string) => {
    const { data, error } = await postAuth("signup", { email, password });
    if (error || !data) {
      return { error: error ?? createAuthError("request_failed", "Signup failed", 500) };
    }
    applyAuthBundle(data);
    setAuthStatus("authenticated");
    return { error: null };
  }, [applyAuthBundle]);

  const updateEmail = useCallback(async (_newEmail: string) => {
    return {
      error: createAuthError(
        "email_change_unavailable",
        "Email change is not available.",
        501
      ),
    };
  }, []);

  const signOut = useCallback(async () => {
    const rt = getStoredRefreshToken();
    clearRefreshTimer();
    let logoutErr: AuthError | null = null;
    if (rt) {
      const { error } = await postAuth("logout", { refresh_token: rt });
      logoutErr = error;
    }
    clearStoredRefreshToken();
    setSession(null);
    setUser(null);
    setAuthStatus("unauthenticated");
    return { error: logoutErr };
  }, [clearRefreshTimer]);

  const value = useMemo<AuthState>(
    () => ({
      session,
      user,
      authStatus,
      signInWithPassword,
      signUpWithPassword,
      updateEmail,
      signOut,
    }),
    [
      session,
      user,
      authStatus,
      signInWithPassword,
      signUpWithPassword,
      updateEmail,
      signOut,
    ]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
