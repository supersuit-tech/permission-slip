import React, { createElement } from "react";
import { Linking } from "react-native";
import { create, act, type ReactTestRenderer } from "react-test-renderer";
import { SlackContextPreview } from "../SlackContextPreview";
import type { components } from "../../../api/schema";

type SlackContext = components["schemas"]["SlackContext"];

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const CHANNEL: components["schemas"]["SlackContextChannel"] = {
  id: "C001",
  name: "general",
  is_private: false,
  is_dm: false,
  topic: "Company-wide announcements",
  member_count: 120,
  permalink: "https://slack.com/archives/C001",
};

const USER_REF: components["schemas"]["SlackContextUserRef"] = {
  id: "U001",
  name: "alice",
  real_name: "Alice Example",
  title: "Engineer",
};

const MESSAGE: components["schemas"]["SlackContextMessage"] = {
  user: USER_REF,
  text: "Hello *world*",
  ts: "1711234567.000100",
  permalink: "https://slack.com/archives/C001/p1711234567000100",
  is_bot: false,
  truncated: false,
  files: [],
};

const BOT_MESSAGE: components["schemas"]["SlackContextMessage"] = {
  ...MESSAGE,
  is_bot: true,
  text: "Deployment started",
};

const MESSAGE_WITH_FILES: components["schemas"]["SlackContextMessage"] = {
  ...MESSAGE,
  files: [
    { filename: "report.pdf", size_bytes: 204800 },
    { filename: "data.csv", size_bytes: 512 },
  ],
};

const TRUNCATED_MESSAGE: components["schemas"]["SlackContextMessage"] = {
  ...MESSAGE,
  truncated: true,
  text: "Very long message...",
};

function makeContext(overrides: Partial<SlackContext>): SlackContext {
  return {
    context_scope: "recent_channel",
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Render helper
// ---------------------------------------------------------------------------

function render(ctx: SlackContext): ReactTestRenderer {
  return create(createElement(SlackContextPreview, { slackContext: ctx }));
}

function hasTestId(r: ReactTestRenderer, id: string): boolean {
  return r.root.findAll((n) => n.props.testID === id).length > 0;
}

function findByTestId(r: ReactTestRenderer, id: string) {
  return r.root.findAll((n) => n.props.testID === id)[0];
}

function textContent(r: ReactTestRenderer): string {
  return JSON.stringify(r.toJSON());
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("SlackContextPreview", () => {
  let renderer: ReactTestRenderer;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  afterEach(async () => {
    await act(async () => {
      renderer?.unmount();
    });
  });

  // --- channel header ---

  it("renders channel name", async () => {
    await act(async () => {
      renderer = render(makeContext({ channel: CHANNEL }));
    });
    expect(textContent(renderer)).toContain("#general");
  });

  it("renders channel topic", async () => {
    await act(async () => {
      renderer = render(makeContext({ channel: CHANNEL }));
    });
    expect(hasTestId(renderer, "channel-topic")).toBe(true);
    expect(textContent(renderer)).toContain("Company-wide announcements");
  });

  it("renders member count", async () => {
    await act(async () => {
      renderer = render(makeContext({ channel: CHANNEL }));
    });
    expect(hasTestId(renderer, "member-count")).toBe(true);
    expect(textContent(renderer)).toContain("120");
  });

  it("renders private badge for private channels", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          channel: { ...CHANNEL, is_private: true },
        }),
      );
    });
    expect(hasTestId(renderer, "private-badge")).toBe(true);
  });

  it("does not render private badge for public channels", async () => {
    await act(async () => {
      renderer = render(makeContext({ channel: CHANNEL }));
    });
    expect(hasTestId(renderer, "private-badge")).toBe(false);
  });

  it("open-in-slack link fires Linking.openURL", async () => {
    const openURL = jest.spyOn(Linking, "openURL").mockResolvedValue(undefined);

    await act(async () => {
      renderer = render(makeContext({ channel: CHANNEL }));
    });

    const link = findByTestId(renderer, "open-in-slack");
    await act(async () => {
      link?.props.onPress();
    });

    expect(openURL).toHaveBeenCalledWith("https://slack.com/archives/C001");
  });

  // --- scope badges ---

  it("renders self-dm badge for self_dm scope", async () => {
    await act(async () => {
      renderer = render(makeContext({ context_scope: "self_dm" }));
    });
    expect(hasTestId(renderer, "badge-self-dm")).toBe(true);
    expect(textContent(renderer)).toContain("Note to self");
  });

  it("renders first-contact badge for first_contact_dm scope", async () => {
    await act(async () => {
      renderer = render(makeContext({ context_scope: "first_contact_dm" }));
    });
    expect(hasTestId(renderer, "badge-first-contact")).toBe(true);
    expect(textContent(renderer)).toContain("No prior messages with this user");
  });

  it("renders metadata-only badge for metadata_only scope", async () => {
    await act(async () => {
      renderer = render(makeContext({ context_scope: "metadata_only" }));
    });
    expect(hasTestId(renderer, "badge-metadata-only")).toBe(true);
    expect(textContent(renderer)).toContain("Context unavailable");
  });

  // --- recipient card ---

  it("renders recipient card for DM scope", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ context_scope: "recent_dm", recipient: USER_REF }),
      );
    });
    expect(hasTestId(renderer, "recipient-card")).toBe(true);
    expect(textContent(renderer)).toContain("Alice Example");
  });

  it("does not render recipient card for self_dm scope", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ context_scope: "self_dm", recipient: USER_REF }),
      );
    });
    expect(hasTestId(renderer, "recipient-card")).toBe(false);
  });

  it("renders recipient title when available", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ context_scope: "recent_dm", recipient: USER_REF }),
      );
    });
    expect(textContent(renderer)).toContain("Engineer");
  });

  // --- target message ---

  it("renders target message section", async () => {
    await act(async () => {
      renderer = render(makeContext({ target_message: MESSAGE }));
    });
    expect(hasTestId(renderer, "target-message-section")).toBe(true);
  });

  it("strips mrkdwn from message text", async () => {
    await act(async () => {
      renderer = render(makeContext({ target_message: MESSAGE }));
    });
    // "Hello *world*" → "Hello world"
    expect(textContent(renderer)).toContain("Hello world");
    expect(textContent(renderer)).not.toContain("*world*");
  });

  it("renders bot badge for bot messages", async () => {
    await act(async () => {
      renderer = render(makeContext({ target_message: BOT_MESSAGE }));
    });
    expect(hasTestId(renderer, "bot-badge")).toBe(true);
  });

  it("renders file attachments with name and size", async () => {
    await act(async () => {
      renderer = render(makeContext({ target_message: MESSAGE_WITH_FILES }));
    });
    expect(hasTestId(renderer, "file-list")).toBe(true);
    expect(textContent(renderer)).toContain("report.pdf");
    expect(textContent(renderer)).toContain("200 KB");
    expect(textContent(renderer)).toContain("data.csv");
    expect(textContent(renderer)).toContain("512 B");
  });

  it("shows truncated indicator for truncated messages", async () => {
    await act(async () => {
      renderer = render(makeContext({ target_message: TRUNCATED_MESSAGE }));
    });
    expect(textContent(renderer)).toContain("[truncated]");
  });

  it("message permalink fires Linking.openURL", async () => {
    const openURL = jest.spyOn(Linking, "openURL").mockResolvedValue(undefined);

    await act(async () => {
      renderer = render(makeContext({ target_message: MESSAGE }));
    });

    const link = findByTestId(renderer, "message-permalink");
    await act(async () => {
      link?.props.onPress();
    });

    expect(openURL).toHaveBeenCalledWith(MESSAGE.permalink);
  });

  // --- thread scope ---

  it("renders thread section for thread scope", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "thread",
          thread: { parent: MESSAGE, replies: [BOT_MESSAGE], truncated: false },
        }),
      );
    });
    expect(hasTestId(renderer, "thread-section")).toBe(true);
    expect(hasTestId(renderer, "thread-replies")).toBe(true);
  });

  it("shows thread-truncated notice when thread is truncated", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "thread",
          thread: { parent: MESSAGE, replies: [], truncated: true },
        }),
      );
    });
    expect(hasTestId(renderer, "thread-truncated")).toBe(true);
  });

  it("does not render thread section when scope is not thread", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "recent_channel",
          thread: { parent: MESSAGE, replies: [] },
        }),
      );
    });
    expect(hasTestId(renderer, "thread-section")).toBe(false);
  });

  it("hides target-message block when it duplicates thread.parent (same ts)", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "thread",
          target_message: MESSAGE,
          thread: { parent: MESSAGE, replies: [BOT_MESSAGE] },
        }),
      );
    });
    // thread-section shows the parent — standalone target-message block should be absent
    expect(hasTestId(renderer, "thread-section")).toBe(true);
    const targetSections = renderer.root.findAll(
      (n) => n.props.testID === "target-message-section",
    );
    expect(targetSections.length).toBe(0);
  });

  it("shows target-message block when it differs from thread.parent ts", async () => {
    const differentTarget: typeof MESSAGE = { ...MESSAGE, ts: "1711234000.000000" };
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "thread",
          target_message: differentTarget,
          thread: { parent: MESSAGE, replies: [] },
        }),
      );
    });
    expect(hasTestId(renderer, "target-message-section")).toBe(true);
    expect(hasTestId(renderer, "thread-section")).toBe(true);
  });

  // --- recent messages accordion ---

  it("renders recent-messages section when messages present", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ recent_messages: [MESSAGE, BOT_MESSAGE] }),
      );
    });
    expect(hasTestId(renderer, "recent-messages-section")).toBe(true);
  });

  it("recent messages are collapsed by default", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ recent_messages: [MESSAGE] }),
      );
    });
    expect(hasTestId(renderer, "recent-messages-list")).toBe(false);
  });

  it("expands recent messages on toggle press", async () => {
    await act(async () => {
      renderer = render(
        makeContext({ recent_messages: [MESSAGE] }),
      );
    });

    const toggle = findByTestId(renderer, "recent-messages-toggle");
    await act(async () => {
      toggle?.props.onPress();
    });

    expect(hasTestId(renderer, "recent-messages-list")).toBe(true);
  });

  it("shows channel label with message count and hours", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "recent_channel",
          recent_messages: [MESSAGE, BOT_MESSAGE],
          context_window: { message_count: 2, hours: 24 },
        }),
      );
    });
    expect(textContent(renderer)).toContain("Channel activity");
    expect(textContent(renderer)).toContain("last 24h");
    expect(textContent(renderer)).toContain("2 messages");
  });

  it("shows DM label for recent_dm scope", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          context_scope: "recent_dm",
          recent_messages: [MESSAGE],
          context_window: { message_count: 1, hours: 24 },
        }),
      );
    });
    expect(textContent(renderer)).toContain("DM activity");
    expect(textContent(renderer)).not.toContain("Channel activity");
  });

  // --- missing fields ---

  it("renders without crashing when only context_scope is set", async () => {
    await act(async () => {
      renderer = render(makeContext({}));
    });
    expect(renderer.toJSON()).toBeTruthy();
    expect(hasTestId(renderer, "slack-context-preview")).toBe(true);
  });

  it("renders without crashing when channel has no name", async () => {
    await act(async () => {
      renderer = render(
        makeContext({
          channel: { id: "C999", permalink: "https://slack.com/archives/C999" },
        }),
      );
    });
    expect(renderer.toJSON()).toBeTruthy();
    expect(textContent(renderer)).toContain("C999");
  });

  it("renders without crashing when message has no user", async () => {
    const msg: components["schemas"]["SlackContextMessage"] = {
      text: "Anonymous message",
      ts: "1711234567.000000",
      permalink: "https://slack.com/archives/C001/p1",
    };
    await act(async () => {
      renderer = render(makeContext({ target_message: msg }));
    });
    expect(textContent(renderer)).toContain("Anonymous message");
  });
});
