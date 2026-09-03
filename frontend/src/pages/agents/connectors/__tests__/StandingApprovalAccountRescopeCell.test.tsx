import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { AgentConnectorInstance } from "@/hooks/useAgentConnectorInstances";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import { StandingApprovalAccountRescopeCell } from "../StandingApprovalAccountRescopeCell";

vi.mock("@/hooks/useUpdateStandingApproval", () => ({
  useUpdateStandingApproval: () => ({
    updateStandingApproval: vi.fn(),
    isPending: false,
  }),
}));

const namedInstance: AgentConnectorInstance = {
  connector_instance_id: "22222222-2222-2222-2222-222222222222",
  agent_id: 1,
  connector_id: "protonmail",
  display: "chiedo@chiedo.com",
  is_default: false,
  enabled_at: "2026-01-02T00:00:00Z",
};

const credentialLessInstance: AgentConnectorInstance = {
  connector_instance_id: "33333333-3333-3333-3333-333333333333",
  agent_id: 1,
  connector_id: "protonmail",
  display: "",
  is_default: true,
  enabled_at: "2026-01-01T00:00:00Z",
};

const baseRule: StandingApproval = {
  standing_approval_id: "sa_test",
  agent_id: 1,
  user_id: "user_1",
  action_type: "protonmail.send",
  action_version: "1",
  name: "Send mail",
  status: "active",
  starts_at: "2026-01-01T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
  expires_at: null,
  constraints: {},
  connector_instance_id: null,
  unrestricted: false,
};

describe("StandingApprovalAccountRescopeCell", () => {
  it("hides credential-less instances from the rescope menu", async () => {
    const user = userEvent.setup();

    render(
      <StandingApprovalAccountRescopeCell
        agentId={1}
        rule={baseRule}
        instances={[namedInstance, credentialLessInstance]}
      />,
    );

    await user.click(
      screen.getByRole("button", {
        name: /Change account scope for Send mail: All accounts/i,
      }),
    );

    expect(screen.getByRole("menuitem", { name: "All accounts" })).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: "chiedo@chiedo.com" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("menuitem", { name: "Unnamed account" }),
    ).not.toBeInTheDocument();
  });

  it("keeps the currently scoped credential-less instance in the menu", async () => {
    const user = userEvent.setup();

    render(
      <StandingApprovalAccountRescopeCell
        agentId={1}
        rule={{
          ...baseRule,
          connector_instance_id: credentialLessInstance.connector_instance_id,
        }}
        instances={[namedInstance, credentialLessInstance]}
      />,
    );

    await user.click(
      screen.getByRole("button", {
        name: /Change account scope for Send mail:/i,
      }),
    );

    expect(
      screen.getByRole("menuitem", { name: "Unnamed account" }),
    ).toBeInTheDocument();
  });
});
