import type { Meta, StoryObj } from "@storybook/react";
import { SchemaParameterDetails } from "./SchemaParameterDetails";

const meta: Meta<typeof SchemaParameterDetails> = {
  title: "Components/SchemaParameterDetails",
  component: SchemaParameterDetails,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div className="bg-muted/50 w-[480px] rounded-lg border p-4">
        <Story />
      </div>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof SchemaParameterDetails>;

export const WithSchema: Story = {
  args: {
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      title: "Add dark mode toggle to settings page",
      body: "Implement dark mode toggle using the existing theme context.",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "title"],
      properties: {
        owner: { type: "string", description: "Repository owner (user or organization)" },
        repo: { type: "string", description: "Repository name" },
        title: { type: "string", description: "Issue title" },
        body: { type: "string", description: "Issue body (Markdown supported)" },
      },
    },
  },
};

export const WithEnumValues: Story = {
  args: {
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      pull_number: 142,
      merge_method: "squash",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "pull_number"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        pull_number: { type: "integer", description: "Pull request number" },
        merge_method: {
          type: "string",
          description: "Merge strategy to use",
          enum: ["merge", "squash", "rebase"],
          default: "merge",
        },
      },
    },
  },
};

export const WithMissingRequired: Story = {
  args: {
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
    },
    schema: {
      type: "object",
      required: ["owner", "repo", "title"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        title: { type: "string", description: "Issue title" },
      },
    },
  },
};

export const WithDatetimeValues: Story = {
  args: {
    parameters: {
      summary: "Q1 Planning Review",
      start_time: "2026-03-20T14:00:00-05:00",
      end_time: "2026-03-20T15:30:00-05:00",
      attendees: ["alice@example.com", "bob@example.com", "carol@example.com"],
      calendar_id: "primary",
    },
    schema: {
      type: "object",
      required: ["summary", "start_time", "end_time"],
      properties: {
        summary: { type: "string", description: "Event title" },
        start_time: { type: "string", description: "Start time in RFC 3339 format" },
        end_time: { type: "string", description: "End time in RFC 3339 format" },
        attendees: { type: "array", description: "List of attendee email addresses" },
        calendar_id: { type: "string", description: "Calendar ID", default: "primary" },
      },
    },
  },
};

export const WithDefaultValues: Story = {
  args: {
    parameters: {
      spreadsheet_id: "1abc2def3ghi",
      range: "Sheet1!A1:D10",
    },
    schema: {
      type: "object",
      required: ["spreadsheet_id", "range"],
      properties: {
        spreadsheet_id: { type: "string", description: "The ID of the spreadsheet" },
        range: { type: "string", description: "A1 notation range including sheet name" },
      },
    },
  },
};

export const NoSchema: Story = {
  args: {
    parameters: {
      event_id: "fifrnnp6iai8qi5klOhl9cspms",
      calendar_id: "primary",
      custom_field: "some value",
    },
    schema: null,
  },
};

export const WithMultilineBody: Story = {
  args: {
    parameters: {
      to: "alice@example.com",
      subject: "Weekly standup notes — March 16",
      body: "Hi Alice,\n\nHere are this week's standup notes.\n\nThe team made great progress on the API migration. All endpoints are now versioned and the legacy routes have been deprecated with a 90-day sunset window.\n\nThe new dashboard is ready for QA — please review it before Thursday's release window. Key things to check: permissions, empty states, and the mobile breakpoints.\n\nFinally, a reminder that next Monday is a company holiday, so the standup will be moved to Tuesday at 10am EST.\n\nBest,\nChiedobot",
    },
    schema: {
      type: "object",
      required: ["to", "subject", "body"],
      properties: {
        to: { type: "string", description: "Recipient email address" },
        subject: { type: "string", description: "Email subject line" },
        body: { type: "string", description: "Email body (plain text)" },
      },
    },
  },
};

export const ManyParameters: Story = {
  args: {
    parameters: {
      owner: "supersuit-tech",
      repo: "permission-slip",
      workflow_id: "deploy.yml",
      ref: "main",
      status: "completed",
      branch: "feature/dark-mode",
      event: "push",
      actor: "chiedobot",
      per_page: 30,
      page: 1,
    },
    schema: {
      type: "object",
      required: ["owner", "repo"],
      properties: {
        owner: { type: "string", description: "Repository owner" },
        repo: { type: "string", description: "Repository name" },
        workflow_id: { type: "string", description: "Workflow file name or ID" },
        status: {
          type: "string",
          description: "Filter by run status",
          enum: ["completed", "action_required", "cancelled", "failure", "success", "in_progress", "queued"],
        },
        branch: { type: "string", description: "Filter by branch name" },
        event: { type: "string", description: "Filter by triggering event" },
        actor: { type: "string", description: "Filter by GitHub username" },
        ref: { type: "string", description: "Branch or tag" },
        per_page: { type: "integer", description: "Results per page", default: 30 },
        page: { type: "integer", description: "Page number", default: 1 },
      },
    },
  },
};
