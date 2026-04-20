import type { Meta, StoryObj } from "@storybook/react";
import { NotifyAboutAutoApprovalsRow } from "./NotifyAboutAutoApprovalsRow";

const meta: Meta<typeof NotifyAboutAutoApprovalsRow> = {
  title: "Settings/NotifyAboutAutoApprovalsRow",
  component: NotifyAboutAutoApprovalsRow,
  parameters: { layout: "padded" },
};
export default meta;

type Story = StoryObj<typeof NotifyAboutAutoApprovalsRow>;

export const Enabled: Story = {
  args: {
    enabled: true,
    disabled: false,
    onCheckedChange: () => {},
  },
};

export const Silenced: Story = {
  args: {
    enabled: false,
    disabled: false,
    onCheckedChange: () => {},
  },
};

export const Disabled: Story = {
  args: {
    enabled: true,
    disabled: true,
    onCheckedChange: () => {},
  },
};
