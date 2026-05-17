/**
 * Shared test fixtures for auth-related tests.
 *
 * Uses localStorage + global fetch mocking to drive AuthProvider bootstrap,
 * matching the self-hosted `/api/auth/*` flow.
 */
import { act } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import type { AppSession, AppUser } from "../types";

export const mockUser: AppUser = {
  id: "user-123",
  email: "test@example.com",
};

/** Minimal JWT-shaped token (payload decodes to sub + email). */
export function makeTestAccessToken(userId: string, email: string): string {
  const header = btoa(JSON.stringify({ alg: "HS256", typ: "JWT" }))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  const payload = btoa(
    JSON.stringify({
      sub: userId,
      email,
      exp: Math.floor(Date.now() / 1000) + 3600,
    })
  )
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  return `${header}.${payload}.sig`;
}

export const mockSession: AppSession = {
  access_token: makeTestAccessToken(mockUser.id, mockUser.email ?? ""),
  expires_at: new Date(Date.now() + 3600_000).toISOString(),
  user: mockUser,
};

let savedFetch: typeof fetch | undefined;

export type SetupAuthMocksOptions = {
  authenticated?: boolean;
  /** HTTP status for POST /auth/logout (default 204). Use 5xx to simulate failure. */
  logoutStatus?: number;
};

export function setupAuthMocks(options: SetupAuthMocksOptions = {}) {
  const { authenticated = false, logoutStatus = 204 } = options;
  if (savedFetch === undefined) {
    savedFetch = globalThis.fetch;
  }

  if (authenticated) {
    localStorage.setItem("ps_refresh_token", "mock-refresh-token");
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/auth/refresh")) {
        return new Response(
          JSON.stringify({
            access_token: mockSession.access_token,
            refresh_token: "mock-refresh-token-2",
            expires_at: mockSession.expires_at,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.includes("/auth/logout")) {
        if (logoutStatus === 204) {
          return new Response(null, { status: 204 });
        }
        return new Response("Logout failed", {
          status: logoutStatus,
          headers: { "Content-Type": "text/plain" },
        });
      }
      return new Response("not found", { status: 404 });
    }) as typeof fetch;
  } else {
    localStorage.removeItem("ps_refresh_token");
    globalThis.fetch = vi.fn(() =>
      Promise.resolve(new Response("{}", { status: 404 }))
    ) as typeof fetch;
  }
}

export function restoreFetch() {
  if (savedFetch !== undefined) {
    globalThis.fetch = savedFetch;
  }
}

afterEach(() => {
  restoreFetch();
  try {
    localStorage.removeItem("ps_refresh_token");
  } catch {
    // ignore
  }
});

/** Lets AuthProvider finish async refresh bootstrap before hook assertions. */
export async function settleAuthHydration(): Promise<void> {
  await act(async () => {
    await new Promise((r) => setTimeout(r, 0));
  });
}
