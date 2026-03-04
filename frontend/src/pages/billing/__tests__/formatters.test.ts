import { describe, it, expect } from "vitest";
import { formatCents, isSafeUrl, isStripeUrl, formatDate } from "../formatters";

describe("formatCents", () => {
  it("formats cents to dollars", () => {
    expect(formatCents(271)).toBe("$2.71");
    expect(formatCents(0)).toBe("$0.00");
    expect(formatCents(10000)).toBe("$100.00");
  });
});

describe("isSafeUrl", () => {
  it("accepts https URLs", () => {
    expect(isSafeUrl("https://example.com")).toBe(true);
  });

  it("rejects http URLs", () => {
    expect(isSafeUrl("http://example.com")).toBe(false);
  });

  it("rejects javascript: URLs", () => {
    expect(isSafeUrl("javascript:alert(1)")).toBe(false);
  });

  it("rejects data: URLs", () => {
    expect(isSafeUrl("data:text/html,<script>alert(1)</script>")).toBe(false);
  });

  it("rejects invalid URLs", () => {
    expect(isSafeUrl("not a url")).toBe(false);
  });
});

describe("isStripeUrl", () => {
  it("accepts checkout.stripe.com", () => {
    expect(isStripeUrl("https://checkout.stripe.com/c/pay_abc123")).toBe(true);
  });

  it("accepts invoice.stripe.com", () => {
    expect(isStripeUrl("https://invoice.stripe.com/i/inv_abc123")).toBe(true);
  });

  it("accepts billing.stripe.com", () => {
    expect(isStripeUrl("https://billing.stripe.com/session/abc")).toBe(true);
  });

  it("rejects non-Stripe HTTPS URLs", () => {
    expect(isStripeUrl("https://evil.com/checkout")).toBe(false);
  });

  it("rejects Stripe subdomain spoofing", () => {
    expect(isStripeUrl("https://checkout.stripe.com.evil.com/pay")).toBe(false);
  });

  it("rejects http Stripe URLs", () => {
    expect(isStripeUrl("http://checkout.stripe.com/pay")).toBe(false);
  });

  it("rejects javascript: protocol", () => {
    expect(isStripeUrl("javascript:alert(1)")).toBe(false);
  });

  it("rejects invalid URLs", () => {
    expect(isStripeUrl("")).toBe(false);
    expect(isStripeUrl("not-a-url")).toBe(false);
  });
});

describe("formatDate", () => {
  it("formats an ISO date string", () => {
    const result = formatDate("2026-03-01T00:00:00Z");
    expect(result).toContain("2026");
    expect(result).toContain("Mar");
  });
});
