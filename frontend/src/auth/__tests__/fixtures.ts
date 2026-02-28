/**
 * Shared test fixtures for auth-related tests.
 *
 * Provides mock references (mockAuth, mockMfa), canonical test data
 * (mockUser, mockSession, verifiedFactor), and helpers (setupAuthMocks,
 * aalResponse) so individual test files don't duplicate boilerplate.
 */
import { vi } from "vitest";
import type { Session, User } from "@supabase/supabase-js";
import { supabase } from "../../lib/supabaseClient";

export const mockAuth = vi.mocked(supabase.auth);
// supabase.auth.mfa methods are vi.fn() stubs (from __mocks__/supabaseClient.ts).
// vi.mocked with deep: true requires exhaustive Supabase type shapes in every
// mock return value, which is overly verbose for tests that only inspect a few
// fields. Casting to a shallow mock avoids this while still providing IDE
// autocomplete for method names.
export const mockMfa = supabase.auth.mfa as unknown as {
  [K in keyof typeof supabase.auth.mfa]: ReturnType<typeof vi.fn>;
};

export const mockUser: User = {
  id: "user-123",
  email: "test@example.com",
  aud: "authenticated",
  created_at: "2024-01-01",
  app_metadata: {},
  user_metadata: {},
};

export const mockSession: Session = {
  access_token: "token",
  refresh_token: "refresh",
  expires_in: 3600,
  token_type: "bearer",
  user: mockUser,
};

/** Minimal verified TOTP factor matching the subset AuthContext inspects. */
export const verifiedFactor = {
  id: "factor-1",
  status: "verified" as const,
  factor_type: "totp" as const,
  created_at: "2024-01-01",
  updated_at: "2024-01-01",
};

/** Unverified TOTP factor (enrollment started but not completed). */
export const unverifiedFactor = {
  id: "factor-2",
  status: "unverified" as const,
  factor_type: "totp" as const,
  created_at: "2024-01-01",
  updated_at: "2024-01-01",
};

/** Builds an AAL mock return value without repeating boilerplate. */
export function aalResponse(current: string, next: string) {
  return {
    data: {
      currentLevel: current,
      nextLevel: next,
      currentAuthenticationMethods: [],
    },
    error: null,
  };
}

export function setupAuthMocks({ authenticated = false } = {}) {
  // restoreAllMocks (not clearAllMocks) so mock *implementations* from
  // previous tests are removed, not just call counts.
  vi.restoreAllMocks();
  // onAuthStateChange fires INITIAL_SESSION immediately on subscribe,
  // which drives the initial auth state (no separate getSession needed).
  mockAuth.onAuthStateChange.mockImplementation((callback) => {
    callback(
      "INITIAL_SESSION",
      authenticated ? mockSession : null
    );
    return {
      data: {
        subscription: { id: "test", callback: vi.fn(), unsubscribe: vi.fn() },
      },
    };
  });
}
