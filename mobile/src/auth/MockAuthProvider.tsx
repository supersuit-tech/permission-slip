/**
 * Mock auth provider for local development with Expo Go.
 *
 * When EXPO_PUBLIC_MOCK_AUTH=true, this replaces the real AuthProvider so you
 * can test the full app UI without a running Supabase instance. The provider
 * immediately sets authStatus to "authenticated" with a fake session/user.
 *
 * This module is only imported in __DEV__ mode — it can never reach production.
 */
import { useCallback, useEffect, useMemo, type ReactNode } from "react";
import type { Session, User } from "@supabase/supabase-js";
import type { AuthState, AuthResult } from "./types";
import { AuthContext } from "./authContext";

const MOCK_USER: User = {
  id: "mock-user-00000000-0000-0000-0000-000000000000",
  email: "dev@permissionslip.local",
  app_metadata: {},
  user_metadata: { full_name: "Dev User" },
  aud: "authenticated",
  created_at: new Date().toISOString(),
  factors: [],
} as User; // Safe: mock object — Supabase required fields unused in this app

const MOCK_SESSION: Session = {
  access_token: "mock-access-token",
  refresh_token: "mock-refresh-token",
  expires_in: 86400,
  expires_at: Date.now() / 1000 + 86400,
  token_type: "bearer",
  user: MOCK_USER,
} as Session; // Safe: mock object — only access_token and user are read by the app

export function MockAuthProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    console.log(
      "[MockAuth] Running in mock auth mode — Supabase is bypassed. " +
        "Set EXPO_PUBLIC_MOCK_AUTH= (empty) to disable.",
    );
  }, []);

  const noOp = useCallback(
    async (): Promise<AuthResult> => ({ error: null }),
    [],
  );

  const value = useMemo<AuthState>(
    () => ({
      session: MOCK_SESSION,
      user: MOCK_USER,
      authStatus: "authenticated",
      sendOtp: noOp,
      verifyOtp: async (_email: string, _token: string) => ({ error: null }),
      signInWithPassword: async (_email: string, _password: string) => ({ error: null }),
      verifyMfa: async (_code: string) => ({ error: null }),
      signOut: noOp,
    }),
    [noOp],
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}
