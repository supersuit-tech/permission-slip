/**
 * Shared test fixtures for audit-event related tests.
 *
 * Provides canonical mock data so individual test files don't duplicate
 * event objects that need to stay in sync with the API schema.
 */
import type { Mock } from "vitest";

/** Fixed reference time for deterministic relative-time formatting. */
export const FIXED_NOW = new Date("2026-02-20T12:00:00Z");

/**
 * Extracts the query params from the last call to the audit-events endpoint
 * within a mock function's call history. Avoids repetitive cast-and-index
 * boilerplate in test assertions.
 */
export function getLastAuditQuery(mockFn: Mock): Record<string, unknown> | undefined {
  const calls = mockFn.mock.calls.filter(
    (c: unknown[]) => c[0] === "/v1/audit-events",
  );
  if (calls.length === 0) return undefined;
  const lastCall = calls[calls.length - 1] as unknown[];
  const opts = lastCall?.[1] as { params?: { query?: Record<string, unknown> } } | undefined;
  return opts?.params?.query;
}

export interface MockAuditEvent {
  event_type: string;
  timestamp: string;
  agent_id: number;
  agent_metadata: { name: string } | null;
  action: Record<string, unknown> | null;
  outcome: string;
}

export interface MockAuditResponse {
  data: MockAuditEvent[];
  has_more: boolean;
  next_cursor?: string;
}

export const mockAuditEvents: MockAuditEvent[] = [
  {
    event_type: "approval.approved",
    timestamp: "2026-02-20T11:55:00Z", // 5 min before FIXED_NOW
    agent_id: 1,
    agent_metadata: { name: "My Bot" },
    action: {
      type: "email.send",
      version: "1",
      parameters: { to: "alice@example.com", subject: "Hello" },
    },
    outcome: "approved",
  },
  {
    event_type: "standing_approval.executed",
    timestamp: "2026-02-20T11:30:00Z", // 30 min before FIXED_NOW
    agent_id: 2,
    agent_metadata: { name: "Data Agent" },
    action: {
      type: "email.read",
      version: "1",
      parameters: { folder: "inbox" },
      constraints: { sender: "*@mycompany.com" },
    },
    outcome: "auto_executed",
  },
  {
    event_type: "approval.denied",
    timestamp: "2026-02-20T11:00:00Z", // 1 hr before FIXED_NOW
    agent_id: 1,
    agent_metadata: { name: "My Bot" },
    action: {
      type: "file.delete",
      version: "1",
      parameters: { path: "/tmp/secret.txt" },
    },
    outcome: "denied",
  },
  {
    event_type: "agent.registered",
    timestamp: "2026-02-20T10:00:00Z", // 2 hr before FIXED_NOW
    agent_id: 3,
    agent_metadata: { name: "New Agent" },
    action: null,
    outcome: "registered",
  },
];

/** Payment method charged event fixture for testing payment-specific rendering. */
export const mockPaymentChargedEvent: MockAuditEvent = {
  event_type: "payment_method.charged",
  timestamp: "2026-02-20T11:45:00Z",
  agent_id: 1,
  agent_metadata: { name: "Travel Bot" },
  action: {
    type: "expedia.create_booking",
    payment_method_id: "pm_test123",
    brand: "visa",
    last4: "4242",
    amount_cents: 15000,
    currency: "usd",
    description: "Hotel booking — 2 nights",
  },
  outcome: "charged",
};

export const mockAuditEventsResponse: MockAuditResponse = {
  data: mockAuditEvents,
  has_more: true,
  next_cursor: "cursor123",
};

export const emptyAuditResponse: MockAuditResponse = {
  data: [],
  has_more: false,
};

/**
 * Minimal response for the useAuditEvents hook tests (fewer events,
 * focused on verifying API plumbing rather than UI rendering).
 */
export const mockHookAuditResponse: MockAuditResponse = {
  data: [
    {
      event_type: "approval.approved",
      timestamp: "2026-02-20T10:00:00Z",
      agent_id: 1,
      agent_metadata: { name: "Test Bot" },
      action: {
        type: "email.send",
        version: "1",
        parameters: { to: "a@b.com" },
      },
      outcome: "approved",
    },
    {
      event_type: "approval.denied",
      timestamp: "2026-02-20T09:00:00Z",
      agent_id: 2,
      agent_metadata: { name: "Other Bot" },
      action: {
        type: "file.delete",
        version: "1",
        parameters: { path: "/tmp" },
      },
      outcome: "denied",
    },
  ],
  has_more: true,
  next_cursor: "cursor_abc",
};
