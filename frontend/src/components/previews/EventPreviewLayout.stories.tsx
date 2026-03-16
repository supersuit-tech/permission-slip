import type { Meta, StoryObj } from "@storybook/react";
import { EventPreviewLayout } from "./EventPreviewLayout";

const meta: Meta<typeof EventPreviewLayout> = {
  title: "Previews/EventPreviewLayout",
  component: EventPreviewLayout,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[480px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof EventPreviewLayout>;

export const Default: Story = {
  args: {
    parameters: {
      summary: "Q1 Planning Review",
      start_time: "2026-03-20T14:00:00-05:00",
      end_time: "2026-03-20T15:30:00-05:00",
    },
    fields: { title: "summary", start: "start_time", end: "end_time" },
  },
};

export const ShortMeeting: Story = {
  args: {
    parameters: {
      summary: "Daily Standup",
      start_time: "2026-03-16T09:00:00-05:00",
      end_time: "2026-03-16T09:15:00-05:00",
    },
    fields: { title: "summary", start: "start_time", end: "end_time" },
  },
};

export const AllDayEvent: Story = {
  args: {
    parameters: {
      summary: "Team Offsite",
      start_time: "2026-03-25T00:00:00Z",
      end_time: "2026-03-25T23:59:59Z",
    },
    fields: { title: "summary", start: "start_time", end: "end_time" },
  },
};

export const LongTitle: Story = {
  args: {
    parameters: {
      summary: "Annual cross-functional strategic planning workshop with leadership team and department heads — please prepare your OKR proposals",
      start_time: "2026-04-01T10:00:00-05:00",
      end_time: "2026-04-01T16:00:00-05:00",
    },
    fields: { title: "summary", start: "start_time", end: "end_time" },
  },
};

export const NoTitle: Story = {
  args: {
    parameters: {
      start_time: "2026-03-20T14:00:00-05:00",
      end_time: "2026-03-20T15:30:00-05:00",
    },
    fields: { title: "summary", start: "start_time", end: "end_time" },
  },
};
