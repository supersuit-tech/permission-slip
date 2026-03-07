import { describe, it, expect } from "vitest";
import { getActionSummary } from "@/lib/auditEvents";
import type { AuditEvent } from "@/lib/auditEvents";
import { mockPaymentChargedEvent } from "./auditEventFixtures";

/** Minimal helper to create a typed AuditEvent from a fixture. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- test-only cast for partial fixtures
function asEvent(partial: any): AuditEvent {
  return partial as AuditEvent;
}

describe("getActionSummary", () => {
  it("formats payment_method.charged with action type, amount, card, and description", () => {
    const summary = getActionSummary(asEvent(mockPaymentChargedEvent));
    // Should include the action type so users know what triggered the charge.
    expect(summary).toContain("expedia.create_booking");
    // Should use locale-aware currency formatting.
    expect(summary).toContain("$150.00");
    // Should include card brand and last4.
    expect(summary).toContain("visa");
    expect(summary).toContain("4242");
    // Should include truncated description.
    expect(summary).toContain("Hotel booking");
  });

  it("handles payment event without description", () => {
    const event = {
      ...mockPaymentChargedEvent,
      action: {
        ...(mockPaymentChargedEvent.action as Record<string, unknown>),
        description: undefined,
      },
    };
    const summary = getActionSummary(asEvent(event));
    expect(summary).toContain("$150.00");
    expect(summary).not.toContain("—"); // no description separator
  });

  it("handles payment event without brand/last4", () => {
    const event = {
      ...mockPaymentChargedEvent,
      action: {
        type: "stripe.charge",
        payment_method_id: "pm_test",
        amount_cents: 999,
        currency: "eur",
      },
    };
    const summary = getActionSummary(asEvent(event));
    // EUR uses € symbol via Intl.NumberFormat.
    expect(summary).toMatch(/€|EUR/);
    expect(summary).toContain("9.99");
  });

  it("falls back for unknown currency codes", () => {
    const event = {
      ...mockPaymentChargedEvent,
      action: {
        type: "test.action",
        amount_cents: 500,
        currency: "xyz",
      },
    };
    const summary = getActionSummary(asEvent(event));
    // Should still show the amount even with an unknown currency.
    expect(summary).toContain("5.00");
  });

  it("formats standard approval event with parameters", () => {
    const event = {
      event_type: "approval.approved",
      timestamp: "2026-02-20T12:00:00Z",
      agent_id: 1,
      agent_metadata: { name: "Bot" },
      action: {
        type: "email.send",
        parameters: { to: "alice@example.com", subject: "Hello" },
      },
      outcome: "approved",
    };
    const summary = getActionSummary(asEvent(event));
    expect(summary).toContain("email.send");
    expect(summary).toContain("to:");
  });

  it("returns event_type for agent lifecycle events", () => {
    expect(
      getActionSummary(
        asEvent({
          event_type: "agent.registered",
          agent_id: 1,
          action: null,
          outcome: "registered",
        }),
      ),
    ).toBe("Agent registered");
  });
});
