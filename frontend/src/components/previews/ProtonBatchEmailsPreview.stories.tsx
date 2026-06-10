import type { Meta, StoryObj } from "@storybook/react";
import { ProtonBatchEmailsPreview } from "./ProtonBatchEmailsPreview";

const meta: Meta<typeof ProtonBatchEmailsPreview> = {
  title: "Previews/ProtonBatchEmailsPreview",
  component: ProtonBatchEmailsPreview,
  parameters: { layout: "padded" },
};

export default meta;
type Story = StoryObj<typeof ProtonBatchEmailsPreview>;

export const ArchiveBatch: Story = {
  args: {
    emails: [
      {
        uid: "231",
        subject: "Re: [supersuit-tech/permission-slip] Make redeploy.sh sudo usage adaptive (PR #1312)",
        from: ["notifications@github.com"],
        to: ["me@proton.me"],
        date: "2026-06-09T14:30:00Z",
      },
      {
        uid: "232",
        subject: "March newsletter",
        from: ["news@example.com"],
        to: ["me@proton.me"],
        date: "2026-06-09T15:00:00Z",
      },
      {
        uid: "233",
        subject: "",
        from: ["noreply@example.com"],
        to: ["me@proton.me"],
        date: "2026-06-09T15:30:00Z",
      },
    ],
  },
};
