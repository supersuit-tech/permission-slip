import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../../api/__mocks__/client";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { StandingApprovalsSection } from "../StandingApprovalsSection";

vi.mock("../../../../api/client");

const actions: ConnectorAction[] = [
  {
    action_type: "github.create_issue",
    operation_type: "write",
    name: "Create Issue",
    description: "Create a new issue",
    risk_level: "low",
    requires_payment_method: false,
    parameters_schema: { type: "object", properties: {} },
  },
];

const baseRule: StandingApproval = {
  standing_approval_id: "sa_active",
  agent_id: 42,
  user_id: "user_1",
  action_type: "github.create_issue",
  action_version: "1",
  name: "Active rule",
  status: "active",
  starts_at: "2026-01-01T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
  expires_at: null,
  constraints: { repo: "myorg/*" },
  unrestricted: false,
};

function mockSectionApis(standingApprovals: StandingApproval[]) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
    }
    if (url === "/v1/standing-approval-templates") {
      return Promise.resolve({ data: { data: [] } });
    }
    if (url === "/v1/agents/{agent_id}/connectors/{connector_id}/instances") {
      return Promise.resolve({ data: { data: [] } });
    }
    return Promise.resolve({ data: { data: [] } });
  });
}

function renderSection(standingApprovals: StandingApproval[]) {
  mockSectionApis(standingApprovals);
  const wrapper = createAuthWrapper();
  return render(
    <StandingApprovalsSection
      agentId={42}
      connectorId="github"
      connectorName="GitHub"
      actions={actions}
    />,
    { wrapper },
  );
}

describe("StandingApprovalsSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
  });

  it("does not render revoked standing approvals", async () => {
    renderSection([
      baseRule,
      {
        ...baseRule,
        standing_approval_id: "sa_revoked",
        name: "Revoked rule",
        status: "revoked",
      },
    ]);

    await waitFor(() => {
      expect(screen.getByText("Active rule")).toBeInTheDocument();
    });
    expect(screen.queryByText("Revoked rule")).not.toBeInTheDocument();
    expect(screen.queryByText("revoked")).not.toBeInTheDocument();
  });

  it("still renders expired standing approvals", async () => {
    renderSection([
      baseRule,
      {
        ...baseRule,
        standing_approval_id: "sa_expired",
        name: "Expired rule",
        status: "expired",
        expires_at: "2020-01-01T00:00:00Z",
      },
    ]);

    await waitFor(() => {
      expect(screen.getByText("Expired rule")).toBeInTheDocument();
    });
    expect(screen.getByText("expired")).toBeInTheDocument();
  });

  it("shows empty state when all connector rules are revoked", async () => {
    renderSection([
      {
        ...baseRule,
        standing_approval_id: "sa_revoked_only",
        name: "Only revoked",
        status: "revoked",
      },
    ]);

    await settleAuthHydration();
    await waitFor(() => {
      expect(
        screen.getByText(
          /Every request from this agent will ask for your approval/i,
        ),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("Only revoked")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Add Standing Approval/i }),
    ).toBeInTheDocument();
  });

  it("removes a row after revoke confirmation and refetch", async () => {
    const user = userEvent.setup();
    let standingApprovals: StandingApproval[] = [baseRule];

    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/standing-approvals") {
        return Promise.resolve({ data: { data: standingApprovals } });
      }
      if (url === "/v1/standing-approval-templates") {
        return Promise.resolve({ data: { data: [] } });
      }
      if (url === "/v1/agents/{agent_id}/connectors/{connector_id}/instances") {
        return Promise.resolve({ data: { data: [] } });
      }
      return Promise.resolve({ data: { data: [] } });
    });
    mockPost.mockImplementation((url: string) => {
      if (url === "/v1/standing-approvals/{standing_approval_id}/revoke") {
        standingApprovals = [{ ...baseRule, status: "revoked" }];
        return Promise.resolve({ data: { data: standingApprovals[0] } });
      }
      return Promise.resolve({ data: {} });
    });

    const wrapper = createAuthWrapper();
    render(
      <StandingApprovalsSection
        agentId={42}
        connectorId="github"
        connectorName="GitHub"
        actions={actions}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Active rule")).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: /Revoke Active rule/i }),
    );
    await user.click(screen.getByRole("button", { name: /^Revoke$/i }));

    await waitFor(() => {
      expect(screen.queryByText("Active rule")).not.toBeInTheDocument();
    });
    expect(
      screen.getByText(
        /Every request from this agent will ask for your approval/i,
      ),
    ).toBeInTheDocument();
  });
});
