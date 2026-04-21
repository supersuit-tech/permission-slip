import React, { createElement } from "react";
import { Alert } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import type { components } from "../../../api/schema";
import { EmailThreadCard } from "../EmailThreadCard";

type EmailThread = components["schemas"]["EmailThread"];

function findByTestId(renderer: ReactTestRenderer, testID: string) {
  return renderer.root.find(
    (node) => node.props.testID === testID,
  );
}

function hasTestId(renderer: ReactTestRenderer, testID: string) {
  try {
    findByTestId(renderer, testID);
    return true;
  } catch {
    return false;
  }
}

describe("EmailThreadCard", () => {
  let renderer: ReactTestRenderer;

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  it("renders subject and latest message body", async () => {
    const thread: EmailThread = {
      subject: "Re: Project",
      messages: [
        {
          from: "alice@example.com",
          to: ["bob@example.com"],
          cc: [],
          date: "2026-04-20T15:00:00Z",
          body_html: "<p>Hi</p>",
          body_text: "Hi there",
          snippet: "Hi",
          message_id: "m1",
          truncated: false,
        },
        {
          from: "bob@example.com",
          to: ["alice@example.com"],
          cc: ["cc@example.com"],
          date: "2026-04-20T16:00:00Z",
          body_html: "",
          body_text: "Latest reply body",
          snippet: "",
          message_id: "m2",
          truncated: false,
        },
      ],
    };
    await act(async () => {
      renderer = create(createElement(EmailThreadCard, { thread }));
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("Re: Project");
    expect(json).toContain("Latest reply body");
    expect(json).toContain("bob@example.com");
    expect(json).toContain("Cc");
    expect(json).toContain("cc@example.com");
  });

  it("shows empty fallback when thread has no content", async () => {
    await act(async () => {
      renderer = create(
        createElement(EmailThreadCard, { thread: { subject: "", messages: [] } }),
      );
    });
    expect(hasTestId(renderer, "email-thread-empty")).toBe(true);
  });

  it("shows truncation affordance when truncated is true", async () => {
    const thread: EmailThread = {
      subject: "S",
      messages: [
        {
          from: "a@b.com",
          to: [],
          cc: [],
          date: "2026-04-20T12:00:00Z",
          body_html: "",
          body_text: "long",
          snippet: "",
          message_id: "x",
          truncated: true,
        },
      ],
    };
    await act(async () => {
      renderer = create(createElement(EmailThreadCard, { thread }));
    });
    expect(hasTestId(renderer, "email-thread-truncation-note")).toBe(true);

    const alertSpy = jest.spyOn(Alert, "alert");
    const note = findByTestId(renderer, "email-thread-truncation-note");
    await act(async () => {
      note.props.onPress();
    });
    expect(alertSpy).toHaveBeenCalledWith(
      "Message truncated",
      expect.stringContaining("20 KB"),
    );
    alertSpy.mockRestore();
  });

  it("lists attachment filenames as chips", async () => {
    const thread: EmailThread = {
      subject: "With files",
      messages: [
        {
          from: "a@b.com",
          to: [],
          cc: [],
          date: "2026-04-20T12:00:00Z",
          body_html: "",
          body_text: "See attached",
          snippet: "",
          message_id: "x",
          truncated: false,
          attachments: [{ filename: "report.pdf", size_bytes: 2048 }],
        },
      ],
    };
    await act(async () => {
      renderer = create(createElement(EmailThreadCard, { thread }));
    });
    const json = JSON.stringify(renderer.toJSON());
    expect(json).toContain("report.pdf");
    expect(json).toContain("KB");
  });

  it("expands earlier messages when toggled", async () => {
    const thread: EmailThread = {
      subject: "Thread",
      messages: [
        {
          from: "old@example.com",
          to: [],
          cc: [],
          date: "2026-04-20T10:00:00Z",
          body_html: "",
          body_text: "First message",
          snippet: "",
          message_id: "m0",
          truncated: false,
        },
        {
          from: "new@example.com",
          to: [],
          cc: [],
          date: "2026-04-20T11:00:00Z",
          body_html: "",
          body_text: "Second message",
          snippet: "",
          message_id: "m1",
          truncated: false,
        },
      ],
    };
    await act(async () => {
      renderer = create(createElement(EmailThreadCard, { thread }));
    });
    const jsonCollapsed = JSON.stringify(renderer.toJSON());
    expect(jsonCollapsed).not.toContain("First message");

    const toggle = findByTestId(renderer, "email-thread-earlier-toggle");
    await act(async () => {
      toggle.props.onPress();
    });
    const jsonOpen = JSON.stringify(renderer.toJSON());
    expect(jsonOpen).toContain("First message");
    expect(hasTestId(renderer, "email-thread-earlier-list")).toBe(true);
  });
});
