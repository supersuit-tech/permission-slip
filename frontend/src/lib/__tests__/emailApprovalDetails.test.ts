import { describe, it, expect } from "vitest";
import {
  hasPartialEmailApprovalDetails,
  parseEmailApprovalDetails,
  shouldShowEmailApprovalSection,
} from "../emailApprovalDetails";

describe("parseEmailApprovalDetails", () => {
  it("extracts from, subject, and date from flat resource_details", () => {
    const details = parseEmailApprovalDetails("protonmail.read_email", {
      subject: "Receipt",
      from: ["Anthropic <invoice@anthropic.com>"],
      date: "2026-07-01T12:00:00Z",
    });
    expect(details).toEqual({
      subject: "Receipt",
      from: "Anthropic <invoice@anthropic.com>",
      date: "2026-07-01T12:00:00Z",
    });
  });

  it("returns partial details when only sender resolved", () => {
    const details = parseEmailApprovalDetails("protonmail.read_email", {
      from: ["sender@example.com"],
    });
    expect(hasPartialEmailApprovalDetails(details)).toBe(true);
    expect(details?.from).toBe("sender@example.com");
    expect(details?.subject).toBeNull();
  });

  it("is unavailable when enrichment keys are all missing", () => {
    expect(
      shouldShowEmailApprovalSection("protonmail.read_email", { unrelated: "x" }),
    ).toBe(true);
    expect(parseEmailApprovalDetails("protonmail.read_email", { unrelated: "x" })).toBeNull();
  });
});

describe("shouldShowEmailApprovalSection", () => {
  it("is true for email actions with unavailable enrichment", () => {
    expect(shouldShowEmailApprovalSection("protonmail.read_email", undefined)).toBe(true);
  });

  it("is false for non-email actions", () => {
    expect(shouldShowEmailApprovalSection("github.create_issue", undefined)).toBe(false);
  });
});
