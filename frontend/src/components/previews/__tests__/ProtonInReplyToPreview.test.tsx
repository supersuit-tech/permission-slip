import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  ProtonInReplyToPreview,
  parseProtonInReplyTo,
} from "../ProtonInReplyToPreview";

describe("parseProtonInReplyTo", () => {
  it("returns null for missing or invalid in_reply_to", () => {
    expect(parseProtonInReplyTo(undefined)).toBeNull();
    expect(parseProtonInReplyTo({ in_reply_to: "bad" })).toBeNull();
    expect(parseProtonInReplyTo({ in_reply_to: {} })).toBeNull();
  });

  it("parses valid metadata", () => {
    const parsed = parseProtonInReplyTo({
      in_reply_to: {
        subject: "Weekly Update",
        from: ["alice@example.com"],
        to: ["me@proton.me"],
        date: "2026-01-15T10:00:00Z",
      },
    });
    expect(parsed?.subject).toBe("Weekly Update");
    expect(parsed?.from).toEqual(["alice@example.com"]);
  });
});

describe("ProtonInReplyToPreview", () => {
  it("shows empty state when metadata is null", () => {
    render(<ProtonInReplyToPreview metadata={null} />);
    expect(
      screen.getByText(/No source email details were included/),
    ).toBeInTheDocument();
  });

  it("renders in-reply-to email metadata", () => {
    render(
      <ProtonInReplyToPreview
        metadata={{
          subject: "Question about billing",
          from: ["support@example.com"],
          to: ["me@proton.me"],
          date: "2026-03-01T12:00:00Z",
        }}
      />,
    );
    expect(screen.getByText("In reply to")).toBeInTheDocument();
    expect(screen.getByText("Question about billing")).toBeInTheDocument();
    expect(screen.getByText(/support@example.com/)).toBeInTheDocument();
    expect(screen.getByText(/me@proton.me/)).toBeInTheDocument();
  });
});
