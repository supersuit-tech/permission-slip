import type { Meta, StoryObj } from "@storybook/react";
import { SlackContextPreview } from "./SlackContextPreview";
import type { components } from "@/api/schema";

type SlackContext = components["schemas"]["SlackContext"];

const meta: Meta<typeof SlackContextPreview> = {
  title: "Previews/SlackContextPreview",
  component: SlackContextPreview,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[520px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof SlackContextPreview>;

const aliceUser: components["schemas"]["SlackContextUserRef"] = {
  id: "U001",
  name: "alice",
  real_name: "Alice Chen",
  title: "Staff Engineer",
};

const bobUser: components["schemas"]["SlackContextUserRef"] = {
  id: "U002",
  name: "bob",
  real_name: "Bob Smith",
};

const deployBotUser: components["schemas"]["SlackContextUserRef"] = {
  id: "U003",
  name: "deploybot",
  real_name: "Deploy Bot",
};

const generalChannel: components["schemas"]["SlackContextChannel"] = {
  id: "C001",
  name: "general",
  is_private: false,
  is_dm: false,
  topic: "Company-wide announcements and discussions",
  member_count: 142,
  permalink: "https://example.slack.com/archives/C001",
};

const engineeringChannel: components["schemas"]["SlackContextChannel"] = {
  id: "C002",
  name: "engineering",
  is_private: true,
  is_dm: false,
  topic: "Engineering team sync",
  purpose: "Discuss technical work and coordinate eng efforts",
  member_count: 18,
  permalink: "https://example.slack.com/archives/C002",
};

const dmChannel: components["schemas"]["SlackContextChannel"] = {
  id: "D001",
  is_dm: true,
  is_private: true,
  permalink: "https://example.slack.com/archives/D001",
};

function makeMessage(
  text: string,
  user: components["schemas"]["SlackContextUserRef"],
  tsOffset = 0,
  extras?: Partial<components["schemas"]["SlackContextMessage"]>,
): components["schemas"]["SlackContextMessage"] {
  return {
    text,
    user,
    ts: String(1711200000 + tsOffset),
    permalink: `https://example.slack.com/archives/C001/p${1711200000 + tsOffset}`,
    ...extras,
  };
}

export const RecentChannel: Story = {
  name: "Recent channel activity (send_message)",
  args: {
    slackContext: {
      context_scope: "recent_channel",
      channel: generalChannel,
      recent_messages: [
        makeMessage("Anyone have the Q1 numbers ready?", bobUser, 0),
        makeMessage(
          "Just sent them over to @alice — *check your DMs*!",
          aliceUser,
          120,
        ),
        makeMessage(
          "Deploy to staging finished. All health checks ✅",
          deployBotUser,
          240,
          { is_bot: true },
        ),
        makeMessage(
          "Heads up: standup moved to 10am tomorrow.",
          bobUser,
          360,
        ),
      ],
      context_window: { message_count: 4, hours: 24, truncated: false },
    } satisfies SlackContext,
  },
};

export const ThreadReply: Story = {
  name: "Thread reply (send_message with thread_ts)",
  args: {
    slackContext: {
      context_scope: "thread",
      channel: engineeringChannel,
      thread: {
        parent: makeMessage(
          "Shipping v2.4.1 today. Release notes in Notion. Let me know if you see anything weird.",
          aliceUser,
          0,
        ),
        replies: [
          makeMessage("LGTM — I reviewed the diff this morning.", bobUser, 300),
          makeMessage(
            "Can we also bump the mobile version in this release?",
            { id: "U004", name: "charlie", real_name: "Charlie Davis" },
            600,
          ),
        ],
        truncated: false,
      },
    } satisfies SlackContext,
  },
};

export const UpdateMessage: Story = {
  name: "Update/delete target message",
  args: {
    slackContext: {
      context_scope: "recent_channel",
      channel: generalChannel,
      target_message: makeMessage(
        "Reminder: *all-hands meeting* Friday at 3pm PST. Link: <https://meet.example.com/all-hands|Join meeting>",
        aliceUser,
        0,
        {
          files: [
            { filename: "agenda.pdf", size_bytes: 24576 },
            { filename: "slides.pptx", size_bytes: 1048576 },
          ],
        },
      ),
      recent_messages: [
        makeMessage("Thanks for the reminder!", bobUser, 180),
        makeMessage("Will the recording be posted afterwards?", aliceUser, 360),
      ],
      context_window: { message_count: 2, hours: 24, truncated: false },
    } satisfies SlackContext,
  },
};

export const DirectMessage: Story = {
  name: "DM history (send_dm)",
  args: {
    slackContext: {
      context_scope: "recent_dm",
      channel: dmChannel,
      recipient: aliceUser,
      recent_messages: [
        makeMessage("Hey, did you get a chance to review the PR?", bobUser, 0),
        makeMessage(
          "Not yet, been slammed with the infra migration. Will do it tonight.",
          aliceUser,
          600,
        ),
        makeMessage("No rush, just wanted to check in!", bobUser, 900),
      ],
      context_window: { message_count: 3, hours: 24, truncated: false },
    } satisfies SlackContext,
  },
};

export const SelfDm: Story = {
  name: "Self-DM (Note to self)",
  args: {
    slackContext: {
      context_scope: "self_dm",
      channel: dmChannel,
    } satisfies SlackContext,
  },
};

export const FirstContactDm: Story = {
  name: "First-contact DM (no prior messages)",
  args: {
    slackContext: {
      context_scope: "first_contact_dm",
      channel: dmChannel,
      recipient: {
        id: "U999",
        name: "newjoinee",
        real_name: "Taylor Newcomer",
        title: "Product Manager",
      },
    } satisfies SlackContext,
  },
};

export const MetadataOnly: Story = {
  name: "Rate-limit degrade (metadata_only)",
  args: {
    slackContext: {
      context_scope: "metadata_only",
      channel: {
        id: "C003",
        name: "incidents",
        is_private: true,
        is_dm: false,
        member_count: 7,
        permalink: "https://example.slack.com/archives/C003",
      },
    } satisfies SlackContext,
  },
};

export const ArchiveChannel: Story = {
  name: "Archive channel (channel metadata emphasis)",
  args: {
    slackContext: {
      context_scope: "recent_channel",
      channel: {
        id: "C004",
        name: "old-project-2024",
        is_private: false,
        is_dm: false,
        topic: "Project Nighthawk — wrapped up Jan 2025",
        member_count: 31,
        last_activity_at: "2025-01-15T14:23:00Z",
        permalink: "https://example.slack.com/archives/C004",
      },
      recent_messages: [
        makeMessage("Wrapping up — all tasks closed. Archiving this channel.", aliceUser, 0),
      ],
      context_window: { message_count: 1, hours: 24, truncated: false },
    } satisfies SlackContext,
  },
};

export const TruncatedThread: Story = {
  name: "Large thread (truncated)",
  args: {
    slackContext: {
      context_scope: "thread",
      channel: generalChannel,
      thread: {
        parent: makeMessage(
          "Big announcement: we're migrating to a new CI/CD pipeline next week. Detailed docs in Confluence.",
          aliceUser,
          0,
        ),
        replies: Array.from({ length: 5 }, (_, i) =>
          makeMessage(
            `Reply ${i + 1}: ${["Exciting!", "Will our custom scripts still work?", "Who's the DRI for this?", "Is there a rollback plan?", "Will there be a dry run first?"][i]}`,
            [aliceUser, bobUser, deployBotUser][i % 3] ?? aliceUser,
            300 * (i + 1),
          ),
        ),
        truncated: true,
      },
    } satisfies SlackContext,
  },
};

export const MrkdwnFormatting: Story = {
  name: "Mrkdwn formatting showcase",
  args: {
    slackContext: {
      context_scope: "recent_channel",
      channel: generalChannel,
      recent_messages: [
        makeMessage(
          "*Bold text*, _italic_, ~strikethrough~, and `inline code`.\n>Quoted text from someone else.\n```\nconst x = 42;\n```\nLink: <https://example.com|Example site>",
          aliceUser,
          0,
        ),
      ],
      context_window: { message_count: 1, hours: 24, truncated: false },
    } satisfies SlackContext,
  },
};
