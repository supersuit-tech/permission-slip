import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect } from "vitest";
import {
  EmailThreadPreview,
  parseEmailThreadFromDetails,
  type EmailThread,
} from "../EmailThreadPreview";

function makeMessage(
  overrides: Partial<EmailThread["messages"][number]> = {},
): EmailThread["messages"][number] {
  return {
    from: "alice@example.com",
    to: ["bob@example.com"],
    cc: [],
    date: "2026-04-20T12:00:00Z",
    body_html: "<p>Hello</p>",
    body_text: "Hello",
    snippet: "Hello",
    message_id: "m1",
    truncated: false,
    ...overrides,
  };
}

describe("parseEmailThreadFromDetails", () => {
  it("returns null for invalid shapes", () => {
    expect(parseEmailThreadFromDetails(null)).toBeNull();
    expect(parseEmailThreadFromDetails({})).toBeNull();
    expect(parseEmailThreadFromDetails({ email_thread: "x" })).toBeNull();
  });

  it("parses a valid email_thread object", () => {
    const thread: EmailThread = {
      subject: "Re: Plan",
      messages: [makeMessage({ message_id: "a" }), makeMessage({ message_id: "b", from: "bob@example.com" })],
    };
    const parsed = parseEmailThreadFromDetails({ email_thread: thread });
    expect(parsed).toEqual(thread);
  });
});

describe("EmailThreadPreview", () => {
  it("renders subject and messages", () => {
    const thread: EmailThread = {
      subject: "Project update",
      messages: [
        makeMessage({
          message_id: "1",
          from: "alice@example.com",
          body_html: "<p>First</p>",
          body_text: "First",
        }),
        makeMessage({
          message_id: "2",
          from: "bob@example.com",
          body_html: "<p>Second</p>",
          body_text: "Second",
        }),
      ],
    };
    render(<EmailThreadPreview thread={thread} />);
    expect(screen.getByText("Project update")).toBeInTheDocument();
    expect(screen.getByText("alice@example.com")).toBeInTheDocument();
    expect(screen.getByText("bob@example.com")).toBeInTheDocument();
  });

  it("shows empty-thread fallback when messages is empty", () => {
    render(<EmailThreadPreview thread={{ subject: "X", messages: [] }} />);
    expect(
      screen.getByText("No conversation history was included with this request."),
    ).toBeInTheDocument();
  });

  it("strips script tags from HTML via sanitization (iframe srcdoc)", () => {
    const thread: EmailThread = {
      subject: "Unsafe",
      messages: [
        makeMessage({
          message_id: "1",
          body_html: '<p>Hi</p><script>document.body.dataset.x="bad"</script>',
          body_text: "Hi",
        }),
      ],
    };
    const { container } = render(<EmailThreadPreview thread={thread} />);
    const iframe = container.querySelector("iframe[title='Email message body']");
    expect(iframe).toBeTruthy();
    const srcdoc = iframe?.getAttribute("srcdoc") ?? "";
    expect(srcdoc).not.toContain("<script");
    expect(srcdoc).toContain("<p>Hi</p>");
  });

  it("expands truncation note when Show more is clicked", async () => {
    const user = userEvent.setup();
    const thread: EmailThread = {
      subject: "Trunc",
      messages: [
        makeMessage({
          message_id: "1",
          truncated: true,
          body_html: "<p>Body</p>",
          body_text: "Body",
        }),
      ],
    };
    render(<EmailThreadPreview thread={thread} />);
    expect(screen.getByText(/truncated server-side/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /show more/i }));
    expect(
      screen.getByText(/shortened on the server/i),
    ).toBeInTheDocument();
  });

  it("toggles earlier messages in details", async () => {
    const user = userEvent.setup();
    const thread: EmailThread = {
      subject: "Thread",
      messages: [
        makeMessage({
          message_id: "old",
          from: "old@example.com",
          body_html: "<p>Old</p>",
          body_text: "Old",
        }),
        makeMessage({
          message_id: "new",
          from: "new@example.com",
          body_html: "<p>New</p>",
          body_text: "New",
        }),
      ],
    };
    const { container } = render(<EmailThreadPreview thread={thread} />);
    const detailsEl = container.querySelector("details");
    expect(detailsEl).toBeTruthy();
    if (!detailsEl) throw new Error("expected details");
    expect(detailsEl.open).toBe(false);
    await user.click(screen.getByText("Earlier in this thread"));
    expect(detailsEl.open).toBe(true);
    expect(within(detailsEl).getByText("old@example.com")).toBeVisible();
    expect(screen.getByText("new@example.com")).toBeVisible();
  });
});
