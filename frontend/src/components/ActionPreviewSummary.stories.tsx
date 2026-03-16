import type { Meta, StoryObj } from "@storybook/react";
import { ActionPreviewSummary } from "./ActionPreviewSummary";

const meta: Meta<typeof ActionPreviewSummary> = {
  title: "Components/ActionPreviewSummary",
  component: ActionPreviewSummary,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[480px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof ActionPreviewSummary>;

// --- GitHub ---

export const GitHubCreateIssue: Story = {
  name: "github.create_issue",
  args: {
    actionType: "github.create_issue",
    parameters: { owner: "supersuit-tech", repo: "permission-slip", title: "Add dark mode toggle" },
    schema: null,
    actionName: "Create Issue",
  },
};

export const GitHubMergePR: Story = {
  name: "github.merge_pr",
  args: {
    actionType: "github.merge_pr",
    parameters: { owner: "supersuit-tech", repo: "permission-slip", pull_number: 142, merge_method: "squash" },
    schema: null,
    actionName: "Merge Pull Request",
  },
};

// --- Google ---

export const GoogleDeleteCalendarEvent: Story = {
  name: "google.delete_calendar_event (with resource details)",
  args: {
    actionType: "google.delete_calendar_event",
    parameters: { event_id: "fifrnnp6iai8qi5klOhl9cspms" },
    schema: null,
    actionName: "Delete Calendar Event",
    resourceDetails: { title: "Q1 Planning Review", start_time: "2026-03-16T14:00:00-05:00" },
  },
};

export const GoogleDeleteCalendarEventNoDetails: Story = {
  name: "google.delete_calendar_event (no resource details)",
  args: {
    actionType: "google.delete_calendar_event",
    parameters: { event_id: "fifrnnp6iai8qi5klOhl9cspms" },
    schema: {
      type: "object",
      required: ["event_id"],
      properties: {
        event_id: { type: "string", description: "Calendar event ID" },
      },
    },
    actionName: "Delete Calendar Event",
  },
};

export const GoogleSendEmail: Story = {
  name: "google.send_email (display template)",
  args: {
    actionType: "google.send_email",
    parameters: { to: "alice@example.com", subject: "Meeting follow-up", body: "Thanks for the chat!" },
    schema: null,
    actionName: "Send Email",
    displayTemplate: "Send email to {{to}} — {{subject}}",
  },
};

export const GoogleSheetsReadRange: Story = {
  name: "google.sheets_read_range",
  args: {
    actionType: "google.sheets_read_range",
    parameters: { spreadsheet_id: "1abc2def", range: "Sheet1!A1:D10" },
    schema: null,
    actionName: "Read Spreadsheet Range",
    resourceDetails: { title: "Q1 Revenue Tracker" },
  },
};

export const GoogleReadEmail: Story = {
  name: "google.read_email",
  args: {
    actionType: "google.read_email",
    parameters: { message_id: "18e3a9f2b1c4d567" },
    schema: null,
    actionName: "Read Email",
    resourceDetails: { subject: "Re: Q1 budget approval", from: "finance@company.com" },
  },
};

// --- Slack ---

export const SlackSendMessage: Story = {
  name: "slack.send_message",
  args: {
    actionType: "slack.send_message",
    parameters: { channel: "#engineering", message: "Deploy v2.4.1 completed. All checks passing." },
    schema: null,
    actionName: "Send Message",
  },
};

export const SlackCreateChannel: Story = {
  name: "slack.create_channel",
  args: {
    actionType: "slack.create_channel",
    parameters: { name: "project-phoenix", is_private: false },
    schema: null,
    actionName: "Create Channel",
  },
};

export const SlackCreatePrivateChannel: Story = {
  name: "slack.create_channel (private)",
  args: {
    actionType: "slack.create_channel",
    parameters: { name: "security-incidents", is_private: true },
    schema: null,
    actionName: "Create Channel",
  },
};

// --- Generic fallback ---

export const GenericWithSchema: Story = {
  name: "Generic fallback (schema-driven)",
  args: {
    actionType: "custom.do_something",
    parameters: { target: "production", count: 5, dry_run: true },
    schema: {
      type: "object",
      required: ["target"],
      properties: {
        target: { type: "string", description: "Deployment target" },
        count: { type: "integer", description: "Number of instances" },
        dry_run: { type: "boolean", description: "Dry run mode" },
      },
    },
    actionName: "Do Something",
  },
};

export const GenericNoSchema: Story = {
  name: "Generic fallback (no schema)",
  args: {
    actionType: "unknown.action",
    parameters: { key1: "value1", key2: 42 },
    schema: null,
    actionName: null,
  },
};
