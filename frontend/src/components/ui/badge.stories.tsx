import type { Meta, StoryObj } from "@storybook/react";
import { Badge } from "./badge";

const allVariants = [
  "default",
  "secondary",
  "destructive",
  "outline",
  "success",
  "warning",
  "info",
  "success-soft",
  "warning-soft",
  "info-soft",
  "destructive-soft",
] as const;

const meta: Meta<typeof Badge> = {
  title: "UI/Badge",
  component: Badge,
  argTypes: {
    variant: {
      control: "select",
      options: [...allVariants],
    },
  },
};
export default meta;
type Story = StoryObj<typeof Badge>;

export const Default: Story = {
  args: { children: "Badge", variant: "default" },
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      {allVariants.map((v) => (
        <Badge key={v} variant={v}>
          {v}
        </Badge>
      ))}
    </div>
  ),
};

export const SoftVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge variant="success-soft">Approved</Badge>
      <Badge variant="warning-soft">Pending</Badge>
      <Badge variant="info-soft">In Review</Badge>
      <Badge variant="destructive-soft">Denied</Badge>
    </div>
  ),
};
