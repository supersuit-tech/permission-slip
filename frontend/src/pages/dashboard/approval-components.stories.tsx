import type { Meta, StoryObj } from "@storybook/react";
import { RiskBadge, ConfirmationCodeBanner } from "./approval-components";

// ---------------------------------------------------------------------------
// RiskBadge
// ---------------------------------------------------------------------------

const riskMeta: Meta<typeof RiskBadge> = {
  title: "Components/RiskBadge",
  component: RiskBadge,
  parameters: { layout: "centered" },
};
export default riskMeta;
type RiskStory = StoryObj<typeof RiskBadge>;

export const Low: RiskStory = { args: { level: "low" } };
export const Medium: RiskStory = { args: { level: "medium" } };
export const High: RiskStory = { args: { level: "high" } };
export const NoLevel: RiskStory = { args: { level: null } };

// ---------------------------------------------------------------------------
// ConfirmationCodeBanner — exported as named stories under same file
// ---------------------------------------------------------------------------

export const ConfirmationCode: StoryObj<typeof ConfirmationCodeBanner> = {
  render: () => (
    <div className="w-[400px]">
      <ConfirmationCodeBanner code="A7X-3KM" />
    </div>
  ),
};

export const ConfirmationCodeCopyable: StoryObj<typeof ConfirmationCodeBanner> = {
  render: () => (
    <div className="w-[400px]">
      <ConfirmationCodeBanner code="B9Y-2LP" copyable />
    </div>
  ),
};

export const ConfirmationCodeCustomDescription: StoryObj<typeof ConfirmationCodeBanner> = {
  render: () => (
    <div className="w-[400px]">
      <ConfirmationCodeBanner
        code="Z1W-8QR"
        description="Enter this code in your agent's terminal to complete registration"
      />
    </div>
  ),
};
