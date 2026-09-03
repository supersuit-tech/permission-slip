import { fireEvent, render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import {
  Table,
  TableBody,
} from "@/components/ui/table";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import type { ConnectorAction } from "@/hooks/useConnectorDetail";
import { StandingApprovalRow } from "../StandingApprovalRow";

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

function renderRow(rule: StandingApproval) {
  return render(
    <Table>
      <TableBody>
        <StandingApprovalRow
          agentId={42}
          rule={rule}
          actions={actions}
          instances={[]}
          showAccountColumn={false}
          onEdit={vi.fn()}
          onRevoke={vi.fn()}
        />
      </TableBody>
    </Table>,
  );
}

const baseRule: StandingApproval = {
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
  constraints: { repo: "myorg/*", title: "*" },
  unrestricted: false,
};

describe("StandingApprovalRow", () => {
  it("renders constraint badges for object constraints", () => {
    renderRow(baseRule);

    expect(screen.getByText("Match all issues")).toBeInTheDocument();
    expect(screen.getByText("repo")).toBeInTheDocument();
    expect(screen.getByText("myorg/*")).toBeInTheDocument();
  });

  it("renders No constraints when constraints is null", () => {
    renderRow({ ...baseRule, constraints: null as unknown as Record<string, unknown> });

    expect(screen.getByText("No constraints")).toBeInTheDocument();
  });

  it("renders No constraints when constraints is an empty object", () => {
    renderRow({ ...baseRule, constraints: {} });

    expect(screen.getByText("No constraints")).toBeInTheDocument();
  });

  it("renders Unrestricted badge from the stored flag", () => {
    renderRow({
      ...baseRule,
      unrestricted: true,
      constraints: { repo: "*", title: "*" },
    });

    expect(screen.getByText("Unrestricted")).toBeInTheDocument();
    expect(screen.queryByText("repo")).not.toBeInTheDocument();
  });

  it("renders verified sender constraints from $meta", () => {
    renderRow({
      ...baseRule,
      name: "Read Email from automated@airbnb.com",
      description:
        "Created automatically when approving a standing auto-approve rule proposal",
      constraints: {
        message_id: "*",
        folder: "*",
        $meta: { from: "automated@airbnb.com" },
      },
    });

    fireEvent.click(screen.getByText("+1 more"));
    expect(screen.getByText("Verified sender")).toBeInTheDocument();
    expect(screen.getByText("automated@airbnb.com")).toBeInTheDocument();
    expect(
      screen.queryByText(/Created automatically when approving/i),
    ).not.toBeInTheDocument();
  });
});
