import { vi } from "vitest";

/**
 * Mock Supabase client for tests. All methods are vi.fn() stubs —
 * set return values per-test as needed
 * (e.g. supabase.auth.signInWithOtp.mockResolvedValue(...)).
 */

/** Returns a chainable query builder mock (select/insert/update/delete/eq). */
function mockQueryBuilder() {
  return {
    select: vi.fn().mockReturnThis(),
    insert: vi.fn().mockReturnThis(),
    update: vi.fn().mockReturnThis(),
    delete: vi.fn().mockReturnThis(),
    eq: vi.fn().mockReturnThis(),
    single: vi.fn(),
  };
}

export const supabase = {
  // -- Auth --
  auth: {
    getUser: vi.fn(),
    getSession: vi
      .fn()
      .mockResolvedValue({ data: { session: null }, error: null }),
    signInWithOtp: vi.fn(),
    signInWithPassword: vi.fn(),
    verifyOtp: vi.fn(),
    updateUser: vi.fn(),
    signOut: vi.fn(),
    onAuthStateChange: vi.fn(() => ({
      data: { subscription: { unsubscribe: vi.fn() } },
    })),
    // -- MFA --
    mfa: {
      getAuthenticatorAssuranceLevel: vi.fn().mockResolvedValue({
        data: { currentLevel: "aal1", nextLevel: "aal1" },
        error: null,
      }),
      listFactors: vi.fn().mockResolvedValue({
        data: { totp: [] },
        error: null,
      }),
      enroll: vi.fn(),
      challengeAndVerify: vi.fn(),
      unenroll: vi.fn(),
    },
  },

  // -- Database --
  from: vi.fn(() => mockQueryBuilder()),

  // -- Storage --
  storage: {
    from: vi.fn(() => ({
      upload: vi.fn(),
      download: vi.fn(),
      getPublicUrl: vi.fn(),
    })),
  },

  // -- Realtime --
  channel: vi.fn(() => ({
    on: vi.fn().mockReturnThis(),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  })),
};
