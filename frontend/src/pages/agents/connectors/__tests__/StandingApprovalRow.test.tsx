import { render, screen } from "@testing-library/react";
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
};

describe("StandingApprovalRow", () => {
  it("renders constraint pills for object constraints", () => {
    renderRow(baseRule);

    expect(screen.getByText("Match all issues")).toBeInTheDocument();
    expect(screen.getByText("repo:")).toBeInTheDocument();
    expect(screen.getByText("myorg/*")).toBeInTheDocument();
  });

  it("renders Match all when constraints is null", () => {
    renderRow({ ...baseRule, constraints: null as unknown as Record<string, unknown> });

    expect(screen.getByText("Match all")).toBeInTheDocument();
  });

  it("renders Match all when constraints is an empty object", () => {
    renderRow({ ...baseRule, constraints: {} });

    expect(screen.getByText("Match all")).toBeInTheDocument();
  });
});
