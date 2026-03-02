/**
 * Shared test utilities for the Permission Slip mobile app.
 *
 * Centralises mock factories, query-client setup, and renderer helpers
 * so each test file stays focused on its own assertions.
 */
import { act, type ReactTestRenderer } from "react-test-renderer";
import { QueryClient } from "@tanstack/react-query";
import type { Session } from "@supabase/supabase-js";
import type { components } from "../api/schema";

export type ApprovalSummary = components["schemas"]["ApprovalSummary"];

// ---------------------------------------------------------------------------
// Mock data factories
// ---------------------------------------------------------------------------

/** Returns a valid-looking Supabase session for use in auth mocks. */
export function mockSession(): Session {
  const payload = btoa(JSON.stringify({ aal: "aal1" }));
  return {
    access_token: `header.${payload}.signature`,
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
  } as Session;
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
