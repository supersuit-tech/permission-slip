/**
 * Tests for AuthProvider. Uses mocked auth storage + auth API (same pattern as hook tests).
 */
import { createElement } from "react";
import { create, act } from "react-test-renderer";
import { AuthProvider, useAuth } from "../AuthContext";
import { mockSession } from "../../__test-utils__";

jest.mock("../../lib/authStorage");
jest.mock("../../lib/authApi");

import * as authStorage from "../../lib/authStorage";
import * as authApi from "../../lib/authApi";

interface AuthCapture {
  authStatus: string;
  email: string | undefined;
  session: unknown;
  signInWithPassword: ((e: string, p: string) => Promise<{ error: unknown }>) | null;
  signOut: (() => Promise<{ error: unknown }>) | null;
}

function createAuthCapture() {
  const capture: AuthCapture = {
    authStatus: "unknown",
    email: undefined,
    session: null,
    signInWithPassword: null,
    signOut: null,
  };

  function Consumer() {
    const auth = useAuth();
    capture.authStatus = auth.authStatus;
    capture.email = auth.user?.email ?? undefined;
    capture.session = auth.session;
    capture.signInWithPassword = auth.signInWithPassword;
    capture.signOut = auth.signOut;
    return null;
  }

  return { capture, Consumer };
}

function bootstrapAuthenticated(session: ReturnType<typeof mockSession>) {
  authStorage.getStoredRefreshToken.mockResolvedValue("mock-refresh");
  authApi.postAuth.mockImplementation(async (path: string) => {
    if (path === "refresh") {
      return {
        data: {
          access_token: session.access_token,
          refresh_token: "mock-refresh-2",
          expires_at: session.expires_at,
        },
        error: null,
      };
    }
    return { data: null, error: null };
  });
}

describe("AuthProvider", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    authStorage.getStoredRefreshToken.mockResolvedValue(null);
    authStorage.setStoredRefreshToken.mockResolvedValue(undefined);
    authStorage.clearStoredRefreshToken.mockResolvedValue(undefined);
    authApi.postAuth.mockResolvedValue({ data: null, error: null });
  });

  it("transitions to unauthenticated when no refresh token", async () => {
    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(createElement(AuthProvider, null, createElement(Consumer)));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(capture.authStatus).toBe("unauthenticated");
    expect(capture.session).toBeNull();
  });

  it("bootstraps authenticated session from refresh token", async () => {
    const session = mockSession();
    bootstrapAuthenticated(session);

    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(createElement(AuthProvider, null, createElement(Consumer)));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(capture.authStatus).toBe("authenticated");
    expect(capture.email).toBe("test@example.com");
    expect(capture.session).not.toBeNull();
  });

  it("signInWithPassword calls login endpoint", async () => {
    const session = mockSession();
    authApi.postAuth.mockImplementation(async (path: string, body: Record<string, string>) => {
      if (path === "login") {
        expect(body.email).toBe("a@b.co");
        expect(body.password).toBe("secret12345");
        return {
          data: {
            access_token: session.access_token,
            refresh_token: "rt",
            expires_at: session.expires_at,
          },
          error: null,
        };
      }
      return { data: null, error: null };
    });

    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(createElement(AuthProvider, null, createElement(Consumer)));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    await act(async () => {
      const r = await capture.signInWithPassword!("a@b.co", "secret12345");
      expect(r.error).toBeNull();
    });

    expect(authApi.postAuth).toHaveBeenCalledWith("login", {
      email: "a@b.co",
      password: "secret12345",
    });
    expect(capture.authStatus).toBe("authenticated");
  });

  it("signOut clears session and calls logout", async () => {
    let storedRt = "mock-refresh";
    authStorage.getStoredRefreshToken.mockImplementation(() => Promise.resolve(storedRt));
    authStorage.setStoredRefreshToken.mockImplementation(async (t: string) => {
      storedRt = t;
    });

    const session = mockSession();
    authApi.postAuth.mockImplementation(async (path: string) => {
      if (path === "refresh") {
        return {
          data: {
            access_token: session.access_token,
            refresh_token: "post-refresh-rt",
            expires_at: session.expires_at,
          },
          error: null,
        };
      }
      return { data: null, error: null };
    });

    const { capture, Consumer } = createAuthCapture();

    await act(async () => {
      create(createElement(AuthProvider, null, createElement(Consumer)));
    });

    await act(async () => {
      await new Promise((r) => setTimeout(r, 0));
    });

    expect(capture.authStatus).toBe("authenticated");

    authApi.postAuth.mockImplementation(async (path: string) => {
      if (path === "logout") {
        return { data: null, error: null };
      }
      if (path === "refresh") {
        return {
          data: {
            access_token: session.access_token,
            refresh_token: "post-refresh-rt",
            expires_at: session.expires_at,
          },
          error: null,
        };
      }
      return { data: null, error: null };
    });

    await act(async () => {
      await capture.signOut!();
    });

    expect(authApi.postAuth).toHaveBeenCalledWith("logout", {
      refresh_token: "post-refresh-rt",
    });
    expect(capture.authStatus).toBe("unauthenticated");
    expect(capture.session).toBeNull();
  });
});
