import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import {
  ProtonBatchEmailsPreview,
  parseProtonBatchEmails,
} from "../ProtonBatchEmailsPreview";

const BATCH_DETAILS = {
  messages: {
    "232": {
      subject: "Second email",
      from: ["b@example.com"],
      to: ["me@proton.me"],
      date: "2026-03-02T09:00:00Z",
    },
    "231": {
      subject: "First email",
      from: ["a@example.com"],
      to: ["me@proton.me"],
      date: "2026-03-01T09:00:00Z",
    },
  },
};

describe("parseProtonBatchEmails", () => {
  it("returns null for missing or invalid messages", () => {
    expect(parseProtonBatchEmails(undefined)).toBeNull();
    expect(parseProtonBatchEmails({})).toBeNull();
    expect(parseProtonBatchEmails({ messages: "bad" })).toBeNull();
    expect(parseProtonBatchEmails({ messages: [] })).toBeNull();
  });

  it("returns null for a single resolved message (flat fields cover it)", () => {
    expect(
      parseProtonBatchEmails({
        messages: { "10": { subject: "Only one" } },
      }),
    ).toBeNull();
  });

  it("parses and sorts batch messages by UID", () => {
    const parsed = parseProtonBatchEmails(BATCH_DETAILS);
    expect(parsed).toHaveLength(2);
    expect(parsed?.[0]?.uid).toBe("231");
    expect(parsed?.[0]?.subject).toBe("First email");
    expect(parsed?.[1]?.uid).toBe("232");
  });
});

describe("ProtonBatchEmailsPreview", () => {
  function renderPreview() {
    const emails = parseProtonBatchEmails(BATCH_DETAILS);
    if (!emails) throw new Error("expected batch emails");
    render(<ProtonBatchEmailsPreview emails={emails} />);
  }

  it("renders the count and one collapsed row per email", () => {
    renderPreview();
    expect(screen.getByText("Emails (2)")).toBeDefined();
    expect(screen.getByText("First email")).toBeDefined();
    expect(screen.getByText("Second email")).toBeDefined();
    expect(screen.queryByTestId("batch-email-details-231")).toBeNull();
    expect(screen.queryByTestId("batch-email-details-232")).toBeNull();
  });

  it("expands and collapses an email's details independently", () => {
    renderPreview();

    fireEvent.click(screen.getByTestId("batch-email-toggle-231"));
    const details = screen.getByTestId("batch-email-details-231");
    expect(details.textContent).toContain("a@example.com");
    expect(details.textContent).toContain("me@proton.me");
    expect(details.textContent).toContain("231");
    // The other row stays collapsed.
    expect(screen.queryByTestId("batch-email-details-232")).toBeNull();

    fireEvent.click(screen.getByTestId("batch-email-toggle-231"));
    expect(screen.queryByTestId("batch-email-details-231")).toBeNull();
  });

  it("shows a placeholder for empty subjects", () => {
    const emails = parseProtonBatchEmails({
      messages: {
        "1": { from: ["a@example.com"] },
        "2": { subject: "Has subject" },
      },
    });
    if (!emails) throw new Error("expected batch emails");
    render(<ProtonBatchEmailsPreview emails={emails} />);
    expect(screen.getByText("(No subject)")).toBeDefined();
  });
});
