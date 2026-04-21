import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect } from "vitest";
import { SlackContextPreview } from "../SlackContextPreview";
import type { components } from "@/api/schema";

type SlackContext = components["schemas"]["SlackContext"];
type SlackContextMessage = components["schemas"]["SlackContextMessage"];
type SlackContextUserRef = components["schemas"]["SlackContextUserRef"];

const aliceUser: SlackContextUserRef = {
  id: "U001",
  name: "alice",
  real_name: "Alice Chen",
  title: "Engineer",
};

const bobUser: SlackContextUserRef = {
  id: "U002",
  name: "bob",
  real_name: "Bob Smith",
};

function makeMessage(
  text: string,
  user: SlackContextUserRef,
  ts = "1711200000",
): SlackContextMessage {
  return {
    text,
    user,
    ts,
    permalink: `https://example.slack.com/archives/C001/p${ts}`,
  };
}

const publicChannel: components["schemas"]["SlackContextChannel"] = {
  id: "C001",
  name: "general",
  is_private: false,
  is_dm: false,
  topic: "Company updates",
  member_count: 50,
  permalink: "https://example.slack.com/archives/C001",
};

describe("SlackContextPreview", () => {
  it("renders channel name and member count", () => {
    render(
      <SlackContextPreview
        slackContext={{
          context_scope: "recent_channel",
          channel: publicChannel,
          recent_messages: [],
        }}
      />,
    );
    expect(screen.getByText(/general/)).toBeInTheDocument();
    expect(screen.getByText(/50/)).toBeInTheDocument();
  });

  it("renders Open in Slack link with rel=noopener noreferrer", () => {
    render(
      <SlackContextPreview
        slackContext={{
          context_scope: "recent_channel",
          channel: publicChannel,
          recent_messages: [],
        }}
      />,
    );
    const link = screen.getByRole("link", { name: /open.*slack/i });
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
    expect(link).toHaveAttribute(
      "href",
      "https://example.slack.com/archives/C001",
    );
  });

  it('shows "Note to self" badge for self_dm scope', () => {
    render(
      <SlackContextPreview
        slackContext={{ context_scope: "self_dm" } as SlackContext}
      />,
    );
    expect(screen.getByText("Note to self")).toBeInTheDocument();
  });

  it('shows "No prior messages" badge for first_contact_dm scope', () => {
    render(
      <SlackContextPreview
        slackContext={{
          context_scope: "first_contact_dm",
          recipient: aliceUser,
        } as SlackContext}
      />,
    );
    expect(
      screen.getByText(/no prior messages/i),
    ).toBeInTheDocument();
  });

  it('shows "Context unavailable" badge for metadata_only scope', () => {
    render(
      <SlackContextPreview
        slackContext={{
          context_scope: "metadata_only",
          channel: publicChannel,
        } as SlackContext}
      />,
    );
    expect(screen.getByText(/context unavailable/i)).toBeInTheDocument();
  });

  it("renders recent activity toggle for recent_channel scope", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      recent_messages: [makeMessage("Hello team!", aliceUser)],
      context_window: { message_count: 1, hours: 24 },
    };
    render(<SlackContextPreview slackContext={ctx} />);
    // Accordion toggle is visible; messages are inside it (expanded in separate test)
    expect(
      screen.getByRole("button", { name: /channel activity/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/1 message/)).toBeInTheDocument();
  });

  it("recent activity accordion is collapsed by default", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      recent_messages: [makeMessage("Hidden message", aliceUser)],
      context_window: { message_count: 1, hours: 24 },
    };
    render(<SlackContextPreview slackContext={ctx} />);
    // The toggle button should exist
    expect(
      screen.getByRole("button", { name: /channel activity/i }),
    ).toBeInTheDocument();
    // But the message should not be visible yet (accordion collapsed)
    expect(screen.queryByText("Hidden message")).not.toBeInTheDocument();
  });

  it("expands recent activity accordion on click", async () => {
    const user = userEvent.setup();
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      recent_messages: [makeMessage("Expanded message", aliceUser)],
      context_window: { message_count: 1, hours: 24 },
    };
    render(<SlackContextPreview slackContext={ctx} />);
    await user.click(screen.getByRole("button", { name: /channel activity/i }));
    expect(screen.getByText("Expanded message")).toBeInTheDocument();
  });

  it("renders thread section with parent and replies for thread scope", () => {
    const ctx: SlackContext = {
      context_scope: "thread",
      channel: publicChannel,
      thread: {
        parent: makeMessage("Parent message", aliceUser, "1711200000"),
        replies: [makeMessage("Reply here", bobUser, "1711200300")],
        truncated: false,
      },
    };
    render(<SlackContextPreview slackContext={ctx} />);
    expect(screen.getByText("Parent message")).toBeInTheDocument();
    expect(screen.getByText("Reply here")).toBeInTheDocument();
    expect(screen.getByText(/original message/i)).toBeInTheDocument();
  });

  it("renders target message section when present", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      target_message: makeMessage("Target text here", aliceUser),
      recent_messages: [],
    };
    render(<SlackContextPreview slackContext={ctx} />);
    expect(screen.getByText(/target message/i)).toBeInTheDocument();
    expect(screen.getByText(/Target text here/)).toBeInTheDocument();
  });

  it("renders recipient card for recent_dm scope", () => {
    const ctx: SlackContext = {
      context_scope: "recent_dm",
      recipient: aliceUser,
      recent_messages: [],
      context_window: { message_count: 0, hours: 24 },
    };
    render(<SlackContextPreview slackContext={ctx} />);
    expect(screen.getByText("Alice Chen")).toBeInTheDocument();
    expect(screen.getByText("Engineer")).toBeInTheDocument();
  });

  it("renders file attachments on messages", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      target_message: {
        ...makeMessage("Has files", aliceUser),
        files: [{ filename: "report.pdf", size_bytes: 102400 }],
      },
      recent_messages: [],
    };
    render(<SlackContextPreview slackContext={ctx} />);
    expect(screen.getByText("report.pdf")).toBeInTheDocument();
    expect(screen.getByText(/100 KB/)).toBeInTheDocument();
  });

  it("renders mrkdwn bold text", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      target_message: makeMessage("*Bold words*", aliceUser),
      recent_messages: [],
    };
    render(<SlackContextPreview slackContext={ctx} />);
    const strong = document.querySelector("strong");
    expect(strong).toBeTruthy();
    expect(strong?.textContent).toBe("Bold words");
  });

  it("message permalinks have rel=noopener noreferrer", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      target_message: makeMessage("A message", aliceUser),
      recent_messages: [],
    };
    render(<SlackContextPreview slackContext={ctx} />);
    const links = document.querySelectorAll("a[rel='noopener noreferrer']");
    expect(links.length).toBeGreaterThan(0);
  });

  it("handles missing optional fields gracefully", () => {
    render(
      <SlackContextPreview
        slackContext={{ context_scope: "metadata_only" } as SlackContext}
      />,
    );
    expect(screen.getByText(/context unavailable/i)).toBeInTheDocument();
  });

  it("shows Private badge for private channels", () => {
    render(
      <SlackContextPreview
        slackContext={{
          context_scope: "metadata_only",
          channel: {
            id: "C999",
            name: "secret-team",
            is_private: true,
            is_dm: false,
            permalink: "https://example.slack.com/archives/C999",
          },
        } as SlackContext}
      />,
    );
    expect(screen.getByText("Private")).toBeInTheDocument();
  });

  it("shows truncated warning on truncated messages", () => {
    const ctx: SlackContext = {
      context_scope: "recent_channel",
      channel: publicChannel,
      target_message: {
        ...makeMessage("Truncated body", aliceUser),
        truncated: true,
      },
      recent_messages: [],
    };
    render(<SlackContextPreview slackContext={ctx} />);
    expect(screen.getByText(/message truncated/i)).toBeInTheDocument();
  });
});
