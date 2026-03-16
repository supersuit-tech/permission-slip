import type { Meta, StoryObj } from "@storybook/react";
import { MessagePreviewLayout } from "./MessagePreviewLayout";

const meta: Meta<typeof MessagePreviewLayout> = {
  title: "Previews/MessagePreviewLayout",
  component: MessagePreviewLayout,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[480px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof MessagePreviewLayout>;

export const EmailMessage: Story = {
  args: {
    parameters: {
      to: "alice@example.com",
      subject: "Weekly standup notes — March 16",
      body: "Hi Alice,\n\nHere are this week's standup notes. The team made great progress on the API migration and the new dashboard is nearly ready for QA.\n\nBest,\nChiedobot",
    },
    fields: { to: "to", subject: "subject", body: "body" },
  },
};

export const SlackMessage: Story = {
  args: {
    parameters: {
      channel: "#engineering",
      message: "Deploy v2.4.1 completed successfully. All health checks passing.",
    },
    fields: { to: "channel", body: "message" },
  },
};

export const LongBody: Story = {
  args: {
    parameters: {
      to: "team@example.com",
      subject: "Sprint Retrospective Summary",
      body: "Hey team, here's a summary of our sprint retro:\n\n1. We shipped all planned features on time, including the API migration and dashboard v2.\n2. Test coverage improved from 72% to 85%.\n3. We need to improve our code review turnaround time — several PRs waited 3+ days.\n4. The new monitoring setup caught two issues before they reached production.\n5. Action items: Set up automated PR reminders, schedule a tech debt sprint for Q2.",
    },
    fields: { to: "to", subject: "subject", body: "body" },
  },
};

export const NoSubject: Story = {
  args: {
    parameters: {
      channel: "#random",
      message: "Anyone up for lunch? :taco:",
    },
    fields: { to: "channel", body: "message" },
  },
};

export const NoBody: Story = {
  args: {
    parameters: {
      to: "boss@company.com",
      subject: "Re: PTO Request — Approved",
    },
    fields: { to: "to", subject: "subject", body: "body" },
  },
};
