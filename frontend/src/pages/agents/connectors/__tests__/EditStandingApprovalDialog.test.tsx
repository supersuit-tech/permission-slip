import { screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../../auth/__tests__/fixtures";
import { renderWithProviders } from "../../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../../api/__mocks__/client";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { EditStandingApprovalDialog } from "../EditStandingApprovalDialog";

vi.mock("../../../../api/client");

const actions: ConnectorAction[] = [
  {
    action_type: "github.create_issue",
    operation_type: "write",
    name: "Create Issue",
    description: "Create a new issue",
    risk_level: "low",
    requires_payment_method: false,
    parameters_schema: {
      type: "object",
      required: ["repo"],
      properties: {
        repo: { type: "string" },
      },
    },
  },
];

const nullConstraintRule: StandingApproval = {
  standing_approval_id: "sa_test",
  agent_id: 42,
  user_id: "user_1",
  action_type: "github.create_issue",
  action_version: "1",
  name: "Match all issues",
  status: "active",
  starts_at: "2026-01-01T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
  expires_at: null,
  constraints: null as unknown as Record<string, unknown>,
};

describe("EditStandingApprovalDialog", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: { data: [] } });
  });

  it("opens without crashing when constraints is null", async () => {
    renderWithProviders(
      <EditStandingApprovalDialog
        open
        onOpenChange={vi.fn()}
        rule={nullConstraintRule}
        agentId={42}
        connectorId="github"
        actions={actions}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText("Edit Standing Approval")).toBeInTheDocument();
    });
    expect(screen.getByLabelText(/Name/i)).toHaveValue("Match all issues");
  });
});
