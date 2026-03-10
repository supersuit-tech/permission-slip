import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import {
  FIXED_NOW,
  mockAuditEvents,
  mockPaymentChargedEvent,
} from "../../../lib/__tests__/auditEventFixtures";
import { ActivityDetailSheet } from "../ActivityDetailSheet";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

/** Build a minimal AuditEvent-like object for the sheet. */
function makeEvent(overrides: Record<string, unknown> = {}) {
  return {
    ...mockAuditEvents[0],
    ...overrides,
  };
}

describe("ActivityDetailSheet", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(FIXED_NOW);
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    wrapper = createAuthWrapper();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders nothing visible when open is false", () => {
    const event = makeEvent();
    const { container } = render(
      <ActivityDetailSheet
        event={event as never}
        open={false}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );
    // Sheet is in DOM but not visible — the title should not be shown
    expect(container.querySelector("[data-state='open']")).toBeNull();
  });

  it("renders sheet title when open", async () => {
    const event = makeEvent();
    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Activity Detail")).toBeInTheDocument();
    });
  });

  it("shows agent name and ID", async () => {
    const event = makeEvent();
    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("My Bot")).toBeInTheDocument();
    });
    expect(screen.getByText(/ID: 1/)).toBeInTheDocument();
  });

  it("shows outcome badge", async () => {
    const event = makeEvent({ outcome: "denied" });
    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Denied")).toBeInTheDocument();
    });
  });

  it("shows lifecycle event message when no action", async () => {
    const event = makeEvent({
      event_type: "agent.registered",
      action: null,
      outcome: "registered",
    });
    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Agent registered")).toBeInTheDocument();
    });
  });

  it("fetches approval details for approval events with source_id", async () => {
    const event = makeEvent({
      source_type: "approval",
      source_id: "apr_123",
      event_type: "approval.approved",
    });
    mockGet.mockImplementation((url: string) => {
      if (typeof url === "string" && url.includes("/v1/approvals/")) {
        return Promise.resolve({
          data: {
            approval_id: "apr_123",
            agent_id: 1,
            action: { type: "email.send", parameters: { to: "a@b.com" } },
            context: { risk_level: "low" },
            status: "approved",
            execution_status: "success",
            expires_at: "2026-02-21T12:00:00Z",
            created_at: "2026-02-20T11:00:00Z",
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    // Should show loading then approval details
    await waitFor(() => {
      expect(screen.getByText("Action")).toBeInTheDocument();
    });
  });

  it("shows payment details for payment events", async () => {
    render(
      <ActivityDetailSheet
        event={mockPaymentChargedEvent as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Payment Details")).toBeInTheDocument();
    });
  });

  it("shows inline action details for standing approval events", async () => {
    const event = makeEvent({
      event_type: "standing_approval.executed",
      source_type: "standing_approval",
      source_id: "sa_123",
      action: {
        type: "email.read",
        version: "1",
        parameters: { folder: "inbox" },
      },
      outcome: "auto_executed",
    });

    render(
      <ActivityDetailSheet
        event={event as never}
        open={true}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Action")).toBeInTheDocument();
    });
  });

  it("does not crash when event is present but open is false", () => {
    const { container } = render(
      <ActivityDetailSheet
        event={makeEvent() as never}
        open={false}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );
    expect(container).toBeTruthy();
  });

  it("does not crash when event is null and open is false", () => {
    const { container } = render(
      <ActivityDetailSheet
        event={null}
        open={false}
        onOpenChange={() => {}}
      />,
      { wrapper },
    );
    expect(container).toBeTruthy();
  });
});
