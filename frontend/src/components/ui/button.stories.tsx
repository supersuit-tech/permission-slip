import type { Meta, StoryObj } from "@storybook/react";
import { Button } from "./button";

const meta: Meta<typeof Button> = {
  title: "UI/Button",
  component: Button,
  argTypes: {
    variant: {
      control: "select",
      options: [
        "default",
        "destructive",
        "outline",
        "secondary",
        "ghost",
        "link",
      ],
    },
    size: {
      control: "select",
      options: ["default", "sm", "lg", "icon"],
    },
    disabled: { control: "boolean" },
  },
};
export default meta;
type Story = StoryObj<typeof Button>;

export const Default: Story = {
  args: { children: "Button", variant: "default", size: "default" },
};

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-3">
      {(
        [
          "default",
          "destructive",
          "outline",
          "secondary",
          "ghost",
          "link",
        ] as const
      ).map((variant) => (
        <Button key={variant} variant={variant}>
          {variant}
        </Button>
      ))}
    </div>
  ),
};

export const AllSizes: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <Button size="sm">Small</Button>
      <Button size="default">Default</Button>
      <Button size="lg">Large</Button>
    </div>
  ),
};

export const Disabled: Story = {
  render: () => (
    <div className="flex flex-wrap gap-3">
      {(
        [
          "default",
          "destructive",
          "outline",
          "secondary",
          "ghost",
          "link",
        ] as const
      ).map((variant) => (
        <Button key={variant} variant={variant} disabled>
          {variant}
        </Button>
      ))}
    </div>
  ),
};
