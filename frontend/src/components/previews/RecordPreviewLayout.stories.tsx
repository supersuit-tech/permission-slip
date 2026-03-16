import type { Meta, StoryObj } from "@storybook/react";
import { RecordPreviewLayout } from "./RecordPreviewLayout";

const meta: Meta<typeof RecordPreviewLayout> = {
  title: "Previews/RecordPreviewLayout",
  component: RecordPreviewLayout,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[480px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof RecordPreviewLayout>;

export const GitHubIssue: Story = {
  args: {
    parameters: {
      title: "Add dark mode toggle to settings page",
      repo: "permission-slip",
      owner: "supersuit-tech",
    },
    fields: { title: "title", subtitle: "repo" },
  },
};

export const WithExtraFields: Story = {
  args: {
    parameters: {
      title: "Fix login timeout on mobile",
      repo: "permission-slip",
      owner: "supersuit-tech",
      body: "Users are seeing a timeout error when logging in from mobile devices.",
    },
    fields: { title: "title", subtitle: "repo", body: "body" },
  },
};

export const LongTitle: Story = {
  args: {
    parameters: {
      title: "Implement comprehensive end-to-end testing suite for the approval workflow including edge cases for expired requests, concurrent approvals, and standing approval creation",
      repo: "permission-slip",
    },
    fields: { title: "title", subtitle: "repo" },
  },
};

export const MinimalRecord: Story = {
  args: {
    parameters: { title: "Deploy to staging" },
    fields: { title: "title" },
  },
};
