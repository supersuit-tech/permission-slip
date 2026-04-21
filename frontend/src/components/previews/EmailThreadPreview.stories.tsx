import type { Meta, StoryObj } from "@storybook/react";
import { EmailThreadPreview, type EmailThread } from "./EmailThreadPreview";

const meta: Meta<typeof EmailThreadPreview> = {
  title: "Previews/EmailThreadPreview",
  component: EmailThreadPreview,
  parameters: { layout: "centered" },
  decorators: [(Story) => <div className="w-[560px]"><Story /></div>],
};
export default meta;
type Story = StoryObj<typeof EmailThreadPreview>;

const sampleThread: EmailThread = {
  subject: "Re: Q2 roadmap — engineering sync",
  messages: [
    {
      from: "dana@example.com",
      to: ["team@example.com"],
      cc: [],
      date: "2026-04-18T09:15:00Z",
      body_html: "<p>Hi team — here is a <strong>draft</strong> of the Q2 roadmap.</p>",
      body_text: "Hi team — here is a draft of the Q2 roadmap.",
      snippet: "Hi team — here is a draft",
      message_id: "msg-1",
      truncated: false,
      attachments: [{ filename: "roadmap-draft.pdf", size_bytes: 245_760 }],
    },
    {
      from: "eli@example.com",
      to: ["dana@example.com"],
      cc: ["team@example.com"],
      date: "2026-04-19T14:22:00Z",
      body_html:
        "<p>Thanks Dana. Can we <em>slot</em> the API hardening work before the dashboard launch?</p>",
      body_text:
        "Thanks Dana. Can we slot the API hardening work before the dashboard launch?",
      snippet: "Thanks Dana. Can we slot",
      message_id: "msg-2",
      truncated: false,
    },
    {
      from: "dana@example.com",
      to: ["eli@example.com"],
      cc: ["team@example.com"],
      date: "2026-04-20T08:01:00Z",
      body_html:
        "<p>Yes — I moved hardening to sprint 3. We should be good for an April 28 cut.</p>",
      body_text: "Yes — I moved hardening to sprint 3. We should be good for an April 28 cut.",
      snippet: "Yes — I moved hardening",
      message_id: "msg-3",
      truncated: true,
      attachments: [{ filename: "sprint-board.png", size_bytes: 89_012 }],
    },
  ],
};

export const FullThread: Story = {
  args: {
    thread: sampleThread,
  },
};

export const SingleMessage: Story = {
  args: {
    thread: {
      subject: "Hello",
      messages: [
        {
          from: "support@vendor.com",
          to: ["you@company.com"],
          cc: [],
          date: "2026-04-20T10:00:00Z",
          body_html: "<p>Your ticket <a href=\"https://example.invalid/t/1\">#42</a> was updated.</p>",
          body_text: "Your ticket #42 was updated.",
          snippet: "Your ticket #42",
          message_id: "solo",
          truncated: false,
        },
      ],
    },
  },
};

export const EmptyThread: Story = {
  args: {
    thread: { subject: "(no messages)", messages: [] },
  },
};
