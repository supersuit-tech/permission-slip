import type { Meta, StoryObj } from "@storybook/react";
import { ActionPreviewCard } from "./ActionPreviewCard";

const meta: Meta<typeof ActionPreviewCard> = {
  title: "Previews/ActionPreviewCard",
  component: ActionPreviewCard,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div className="w-[480px]">
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof ActionPreviewCard>;

// ---------------------------------------------------------------------------
// Event layout
// ---------------------------------------------------------------------------

export const EventLayout: Story = {
  args: {
    preview: {
      layout: "event",
      fields: { title: "summary", start: "start_time", end: "end_time" },
    },
    parameters: {
      summary: "Q1 Planning Review",
      start_time: "2026-03-16T14:00:00-05:00",
      end_time: "2026-03-16T15:30:00-05:00",
      attendees: ["alice@example.com", "bob@example.com"],
    },
    actionType: "google.create_calendar_event",
    schema: null,
    actionName: "Create Calendar Event",
  },
};

// ---------------------------------------------------------------------------
// Message layout
// ---------------------------------------------------------------------------

export const MessageLayout: Story = {
  args: {
    preview: {
      layout: "message",
      fields: { to: "to", subject: "subject", body: "body" },
    },
    parameters: {
      to: "team@example.com",
      subject: "Weekly standup notes",
      body: "Hi team,\n\nHere are the standup notes from today's meeting. Please review and add any items I may have missed.\n\nBest,\nChiedobot",
    },
    actionType: "google.send_email",
    schema: null,
    actionName: "Send Email",
  },
};

export const SlackMessageLayout: Story = {
  args: {
    preview: {
      layout: "message",
      fields: { to: "channel", body: "message" },
    },
    parameters: {
      channel: "#engineering",
      message:
        "Deploy v2.4.1 completed successfully. All health checks passing.",
    },
    actionType: "slack.send_message",
    schema: null,
    actionName: "Send Message",
  },
};

// ---------------------------------------------------------------------------
// Record layout
// ---------------------------------------------------------------------------

export const RecordLayout: Story = {
  args: {
    preview: {
      layout: "record",
      fields: { title: "title", subtitle: "repo" },
    },
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      title: "Add dark mode toggle to settings page",
      body: "Implement dark mode toggle using the existing theme context.",
    },
    actionType: "github.create_issue",
    schema: null,
    actionName: "Create Issue",
  },
};

// ---------------------------------------------------------------------------
// Fallback (no preview config)
// ---------------------------------------------------------------------------

export const FallbackSummary: Story = {
  args: {
    preview: null,
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      pull_number: 42,
      merge_method: "squash",
    },
    actionType: "github.merge_pr",
    schema: {
      type: "object",
      required: ["owner", "repo", "pull_number"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        pull_number: { type: "integer", description: "Pull request number" },
        merge_method: {
          type: "string",
          description: "Merge strategy",
          enum: ["merge", "squash", "rebase"],
        },
      },
    },
    actionName: "Merge Pull Request",
  },
};

export const FallbackWithDisplayTemplate: Story = {
  args: {
    preview: null,
    parameters: {
      to: "alice@example.com",
      subject: "Meeting follow-up",
      body: "Thanks for the meeting today!",
    },
    actionType: "google.send_email",
    schema: null,
    actionName: "Send Email",
    displayTemplate: "Send email to {{to}} — {{subject}}",
  },
};

export const FallbackWithResourceDetails: Story = {
  args: {
    preview: null,
    parameters: {
      event_id: "fifrnnp6iai8qi5klOhl9cspms",
      calendar_id: "primary",
    },
    actionType: "google.delete_calendar_event",
    schema: {
      type: "object",
      required: ["event_id"],
      properties: {
        event_id: { type: "string", description: "Calendar event ID" },
        calendar_id: {
          type: "string",
          description: "Calendar ID",
          default: "primary",
        },
      },
    },
    actionName: "Delete Calendar Event",
    resourceDetails: {
      title: "Q1 Planning Review",
      start_time: "2026-03-16T14:00:00-05:00",
    },
  },
};
