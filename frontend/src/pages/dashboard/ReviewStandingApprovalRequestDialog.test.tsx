import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { toast } from "sonner";
import { ReviewStandingApprovalRequestDialog } from "./ReviewStandingApprovalRequestDialog";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

const approveRequest = vi.fn();
const denyRequest = vi.fn();

vi.mock("@/hooks/useApproveStandingApprovalRequest", () => ({
  useApproveStandingApprovalRequest: () => ({
    approveRequest,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useDenyStandingApprovalRequest", () => ({
  useDenyStandingApprovalRequest: () => ({
    denyRequest,
    isPending: false,
  }),
}));

vi.mock("@/hooks/useActionSchema", () => ({
  useActionSchema: () => ({
    connectorName: "Proton Mail",
    isLoading: false,
  }),
}));

const mockUseStandingApprovalInstanceAmbiguityWarning = vi.fn(() => ({
  showWarning: false,
  warningMessage: "Applies to an unspecified account",
}));

vi.mock("@/hooks/useStandingApprovalInstanceAmbiguityWarning", () => ({
  useStandingApprovalInstanceAmbiguityWarning: () =>
    mockUseStandingApprovalInstanceAmbiguityWarning(),
}));

describe("ReviewStandingApprovalRequestDialog", () => {
  beforeEach(() => {
    mockUseStandingApprovalInstanceAmbiguityWarning.mockReturnValue({
      showWarning: false,
      warningMessage: "Applies to an unspecified account",
    });
  });
  it("shows rule proposal badge, connector name, and action type", () => {
    render(
      <ReviewStandingApprovalRequestDialog
        open
        onOpenChange={() => {}}
        agentDisplayName="Test Agent"
        request={{
          request_id: "sar_test",
          agent_id: 1,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          connector_instance_display: "Personal",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }}
      />,
    );

    expect(screen.getAllByText("Rule proposal").length).toBeGreaterThan(0);
    expect(screen.getByText(/Proton Mail \(Personal\)/)).toBeInTheDocument();
    expect(screen.getByText(/protonmail.read_email/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Approve rule/i })).toBeInTheDocument();
  });

  it("shows ambiguity warning when instance is unspecified on a multi-instance connector", () => {
    mockUseStandingApprovalInstanceAmbiguityWarning.mockReturnValue({
      showWarning: true,
      warningMessage: "Applies to an unspecified account",
    });

    render(
      <ReviewStandingApprovalRequestDialog
        open
        onOpenChange={() => {}}
        agentDisplayName="Test Agent"
        request={{
          request_id: "sar_ambiguous",
          agent_id: 1,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }}
      />,
    );

    expect(
      screen.getByText("Applies to an unspecified account"),
    ).toBeInTheDocument();
  });

  it("hides ambiguity warning when instance display is frozen", () => {
    mockUseStandingApprovalInstanceAmbiguityWarning.mockReturnValue({
      showWarning: false,
      warningMessage: "Applies to an unspecified account",
    });

    render(
      <ReviewStandingApprovalRequestDialog
        open
        onOpenChange={() => {}}
        agentDisplayName="Test Agent"
        request={{
          request_id: "sar_clear",
          agent_id: 1,
          user_id: "user-1",
          action_type: "protonmail.read_email",
          action_version: "1",
          constraints: { from: "auto-confirm@amazon.com" },
          connector_name: "Proton Mail",
          connector_instance_display: "Personal",
          status: "pending",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }}
      />,
    );

    expect(
      screen.queryByText("Applies to an unspecified account"),
    ).not.toBeInTheDocument();
  });

  it("shows API error message when approve fails", async () => {
    approveRequest.mockRejectedValueOnce(new Error("Action type is not registered for connector"));

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

    fireEvent.click(screen.getByRole("button", { name: /Approve rule/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "Action type is not registered for connector",
      );
    });
  });
});
