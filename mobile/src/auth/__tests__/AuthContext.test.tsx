/**
 * Tests for AuthProvider logic. Since react-test-renderer has peer dep
 * conflicts with React 19.2.0 in Expo SDK 55, these tests validate
 * the auth state machine via createElement + act rather than renderHook.
 */
import { createElement, useEffect, type ReactNode } from "react";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import { AuthProvider, useAuth } from "../AuthContext";

// --- Mocks ---

// Module-level state shared between mock factory and tests.
// Using an object so references remain stable after jest.mock hoisting.
const mocks = {
  authChangeCallback: null as
    | ((event: AuthChangeEvent, session: Session | null) => void)
    | null,
  signInWithOtp: jest.fn(),
  verifyOtp: jest.fn(),
  signOut: jest.fn(),
};

jest.mock("../../lib/supabaseClient", () => ({
  supabase: {
    auth: {
      onAuthStateChange: jest.fn(
        (cb: (event: AuthChangeEvent, session: Session | null) => void) => {
          mocks.authChangeCallback = cb;
          // Fire INITIAL_SESSION with null (no session) asynchronously.
          setTimeout(() => cb("INITIAL_SESSION" as AuthChangeEvent, null), 0);
          return {
            data: { subscription: { unsubscribe: jest.fn() } },
          };
        }
      ),
      signInWithOtp: (...args: unknown[]) => mocks.signInWithOtp(...args),
      verifyOtp: (...args: unknown[]) => mocks.verifyOtp(...args),
      signOut: (...args: unknown[]) => mocks.signOut(...args),
      mfa: {
        getAuthenticatorAssuranceLevel: jest.fn().mockResolvedValue({
          data: { currentLevel: "aal1", nextLevel: "aal1" },
          error: null,
        }),
      },
    },
  },
}));

// --- Helpers ---

/** Builds a minimal mock session for testing. */
function mockSession(overrides?: Partial<Session>): Session {
  const aal1Payload = btoa(JSON.stringify({ aal: "aal1" }));
  const defaultToken = `header.${aal1Payload}.signature`;

  return {
    access_token: defaultToken,
    refresh_token: "mock-refresh",
    expires_in: 3600,
    expires_at: Date.now() / 1000 + 3600,
    token_type: "bearer",
    user: {
      id: "user-1",
      email: "test@example.com",
      app_metadata: {},
      user_metadata: {},
      aud: "authenticated",
      created_at: new Date().toISOString(),
      factors: [],
    },
    ...overrides,
  } as Session;
}

/**
 * Captures the useAuth() return value via a child component.
 * We track the latest value via a shared ref object.
 */
interface AuthCapture {
  authStatus: string;
  email: string | undefined;
  session: unknown;
}

function createAuthCapture() {
  const capture: AuthCapture = {
    authStatus: "unknown",
    email: undefined,
    session: null,
  };

  function Consumer() {
    const auth = useAuth();
    // Update capture on every render.
    capture.authStatus = auth.authStatus;
    capture.email = auth.user?.email ?? undefined;
    capture.session = auth.session;
    return null;
  }

  return { capture, Consumer };
}

// --- Tests ---

describe("AuthProvider", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("starts in loading state then transitions to unauthenticated", async () => {
    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(
        createElement(AuthProvider, null, createElement(Consumer))
      );
    });

    // Initially loading before the INITIAL_SESSION callback fires.
    // After the setTimeout(0) fires, should transition.
    await act(async () => {
      jest.runAllTimers();
    });

    expect(capture.authStatus).toBe("unauthenticated");
    expect(capture.session).toBeNull();
  });

  it("transitions to authenticated when session event fires", async () => {
    const { capture, Consumer } = createAuthCapture();

    let renderer: ReactTestRenderer;
    await act(async () => {
      renderer = create(
        createElement(AuthProvider, null, createElement(Consumer))
      );
    });

    await act(async () => {
      jest.runAllTimers();
    });

    expect(capture.authStatus).toBe("unauthenticated");

    const session = mockSession();
    await act(async () => {
      mocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    expect(capture.authStatus).toBe("authenticated");
    expect(capture.email).toBe("test@example.com");
  });

  it("detects mfa_required when user has verified TOTP factor at AAL1", async () => {
    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(
        createElement(AuthProvider, null, createElement(Consumer))
      );
    });

    await act(async () => {
      jest.runAllTimers();
    });

    const session = mockSession({
      user: {
        id: "user-mfa",
        email: "mfa@example.com",
        app_metadata: {},
        user_metadata: {},
        aud: "authenticated",
        created_at: new Date().toISOString(),
        factors: [
          {
            id: "factor-1",
            friendly_name: "TOTP",
            factor_type: "totp",
            status: "verified",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ],
      } as Session["user"],
    });

    await act(async () => {
      mocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });

    expect(capture.authStatus).toBe("mfa_required");
  });

  it("transitions to unauthenticated on sign-out event", async () => {
    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(
        createElement(AuthProvider, null, createElement(Consumer))
      );
    });

    await act(async () => {
      jest.runAllTimers();
    });

    // Sign in first.
    const session = mockSession();
    await act(async () => {
      mocks.authChangeCallback!("SIGNED_IN" as AuthChangeEvent, session);
    });
    expect(capture.authStatus).toBe("authenticated");

    // Fire sign out.
    await act(async () => {
      mocks.authChangeCallback!("SIGNED_OUT" as AuthChangeEvent, null);
    });

    expect(capture.authStatus).toBe("unauthenticated");
    expect(capture.session).toBeNull();
  });

  it("sendOtp calls supabase.auth.signInWithOtp", async () => {
    mocks.signInWithOtp.mockResolvedValue({ error: null });

    let sendOtp: ((email: string) => Promise<unknown>) | undefined;

    function CaptureSendOtp() {
      const auth = useAuth();
      sendOtp = auth.sendOtp;
      return null;
    }

    await act(async () => {
      create(
        createElement(AuthProvider, null, createElement(CaptureSendOtp))
      );
    });

    await act(async () => {
      jest.runAllTimers();
    });

    expect(sendOtp).toBeDefined();

    let response: unknown;
    await act(async () => {
      response = await sendOtp!("test@example.com");
    });

    expect(mocks.signInWithOtp).toHaveBeenCalledWith({
      email: "test@example.com",
    });
    expect(response).toEqual({ error: null });
  });

  it("signOut calls supabase.auth.signOut with local scope", async () => {
    mocks.signOut.mockResolvedValue({ error: null });

    let signOut: (() => Promise<unknown>) | undefined;

    function CaptureSignOut() {
      const auth = useAuth();
      signOut = auth.signOut;
      return null;
    }

    await act(async () => {
      create(
        createElement(AuthProvider, null, createElement(CaptureSignOut))
      );
    });

    await act(async () => {
      jest.runAllTimers();
    });

    await act(async () => {
      await signOut!();
    });

    expect(mocks.signOut).toHaveBeenCalledWith({ scope: "local" });
  });
});
