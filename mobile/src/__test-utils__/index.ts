/**
 * Shared test utilities for the Permission Slip mobile app.
 *
 * Centralises mock factories, query-client setup, and renderer helpers
 * so each test file stays focused on its own assertions.
 *
 * Auth hook tests should `jest.mock("../../lib/authStorage")` and
 * `jest.mock("../../lib/authApi")`, then configure the mocked functions so
 * `AuthProvider` bootstraps into the desired session state.
 */
import { act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient } from "@tanstack/react-query";
import type { AppSession } from "../auth/types";
import type { components } from "../api/schema";

export type ApprovalSummary = components["schemas"]["ApprovalSummary"];

function makeTestAccessToken(userId: string, email: string): string {
  const header = btoa(JSON.stringify({ alg: "HS256", typ: "JWT" }))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  const payload = btoa(
    JSON.stringify({
      sub: userId,
      email,
      exp: Math.floor(Date.now() / 1000) + 3600,
    }),
  )
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  return `${header}.${payload}.sig`;
}

/** Returns a session-shaped object for auth + API hook tests. */
export function mockSession(): AppSession {
  const userId = "user-1";
  const email = "test@example.com";
  return {
    access_token: makeTestAccessToken(userId, email),
    expires_at: new Date(Date.now() + 3600_000).toISOString(),
    user: { id: userId, email },
  };
}

/** A realistic pending approval object for use in tests. */
export const mockApproval: ApprovalSummary = {
  approval_id: "appr_abc123",
  agent_id: 42,
  action: {
    type: "email.send",
    version: "1",
    parameters: { to: ["user@example.com"], subject: "Hello" },
  },
  context: {
    description: "Send welcome email to new user",
    risk_level: "low",
  },
  status: "pending",
  expires_at: "2026-03-02T13:25:00Z",
  created_at: "2026-03-02T13:20:00Z",
};

// ---------------------------------------------------------------------------
// React Query
// ---------------------------------------------------------------------------

/** Creates a QueryClient configured for tests (no retries). */
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

// ---------------------------------------------------------------------------
// Async helpers
// ---------------------------------------------------------------------------

/**
 * Wait for a predicate to become true, flushing React updates each iteration.
 * Uses real timers with small delays to avoid infinite loops from React Query's
 * refetch interval when fake timers are used.
 */
export async function waitFor(
  predicate: () => boolean,
  { timeout = 3000, interval = 10 } = {},
) {
  const start = Date.now();
  while (!predicate()) {
    if (Date.now() - start > timeout) {
      throw new Error("waitFor timed out");
    }
    await act(async () => {
      await new Promise((r) => setTimeout(r, interval));
    });
  }
}

// ---------------------------------------------------------------------------
// Renderer query helpers
// ---------------------------------------------------------------------------

/** Returns true if any node in the tree has the given testID. */
export function hasTestId(renderer: ReactTestRenderer, testID: string): boolean {
  return renderer.root.findAll((node) => node.props.testID === testID).length > 0;
}

/** Returns true if any text node in the tree contains the exact string. */
export function hasText(renderer: ReactTestRenderer, text: string): boolean {
  return (
    renderer.root.findAll(
      (node) =>
        typeof node.children?.[0] === "string" && node.children[0] === text,
    ).length > 0
  );
}

/** Returns the first node matching the given testID, or undefined. */
export function findFirstByTestId(renderer: ReactTestRenderer, testID: string) {
  const matches = renderer.root.findAll((node) => node.props.testID === testID);
  return matches[0];
}
