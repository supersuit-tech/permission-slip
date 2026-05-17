import type { ReactNode } from "react";
import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { AuthProvider, useAuth } from "../AuthContext";
import { makeTestAccessToken, setupAuthMocks } from "./fixtures";

function wrap({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("useAuth throws outside provider", () => {
    expect(() => renderHook(() => useAuth())).toThrow(
      "useAuth must be used within an AuthProvider",
    );
  });

  it("resolves to unauthenticated when no refresh token", async () => {
    setupAuthMocks({ authenticated: false });
    const { result } = renderHook(() => useAuth(), { wrapper: wrap });

    await waitFor(() => {
      expect(result.current.authStatus).toBe("unauthenticated");
    });
    expect(result.current.session).toBeNull();
  });

  it("bootstraps session from refresh token", async () => {
    setupAuthMocks({ authenticated: true });
    const { result } = renderHook(() => useAuth(), { wrapper: wrap });

    await waitFor(() => {
      expect(result.current.authStatus).toBe("authenticated");
    });
    expect(result.current.session?.access_token).toBeTruthy();
    expect(result.current.user?.id).toBe("user-123");
  });

  it("signInWithPassword calls login endpoint", async () => {
    setupAuthMocks({ authenticated: false });
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/login")) {
        return new Response(
          JSON.stringify({
            access_token: makeTestAccessToken("u1", "a@b.co"),
            refresh_token: "rt",
            expires_at: new Date(Date.now() + 900_000).toISOString(),
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response("{}", { status: 404 });
    });
    globalThis.fetch = fetchMock as typeof fetch;

    const { result } = renderHook(() => useAuth(), { wrapper: wrap });

    await waitFor(() => {
      expect(result.current.authStatus).toBe("unauthenticated");
    });

    await act(async () => {
      const r = await result.current.signInWithPassword("a@b.co", "password123");
      expect(r.error).toBeNull();
    });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/auth/login"),
      expect.objectContaining({ method: "POST" }),
    );
    expect(result.current.authStatus).toBe("authenticated");
  });
});
