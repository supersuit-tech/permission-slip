import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ReviewStandingApprovalRequestDialog } from "./ReviewStandingApprovalRequestDialog";

vi.mock("@/hooks/useApproveStandingApprovalRequest", () => ({
  useApproveStandingApprovalRequest: () => ({
    approveRequest: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("@/hooks/useDenyStandingApprovalRequest", () => ({
  useDenyStandingApprovalRequest: () => ({
    denyRequest: vi.fn(),
    isPending: false,
  }),
}));

describe("ReviewStandingApprovalRequestDialog", () => {
  it("shows rule proposal badge and action type", () => {
    render(
      <ReviewStandingApprovalRequestDialog
        open
        onOpenChange={() => {}}
        agentDisplayName="Test Agent"
        request={{
          request_id: "sar_test",
          agent_id: 1,
          user_id: "user-1",
          action_type: "email.send",
          action_version: "1",
          constraints: { to: "*@example.com" },
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }}
      />,
    );

    expect(screen.getAllByText("Rule proposal").length).toBeGreaterThan(0);
    expect(screen.getByText(/email.send/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Approve rule/i })).toBeInTheDocument();
  });
});
