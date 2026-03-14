import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { ReviewApprovalDialog } from "../ReviewApprovalDialog";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const futureDate = new Date(Date.now() + 600_000).toISOString();

function makeApproval(overrides?: Partial<ApprovalSummary>): ApprovalSummary {
  return {
    approval_id: "appr_test123",
    agent_id: 1,
    action: {
      type: "email.send",
      version: "1",
      parameters: { recipient: "user@example.com", subject: "Hello" },
    },
    context: {
      description: "Send an email",
      risk_level: "low",
    },
    status: "pending",
    expires_at: futureDate,
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  } as ApprovalSummary;
}

const mockAgents = [
  {
    agent_id: 1,
    status: "registered" as const,
    metadata: { name: "Test Bot" },
    confirmation_code: null,
    expires_at: null,
    created_at: "2026-01-01T00:00:00Z",
  },
];

function setupMocks({
  standingApprovals = [] as Array<{ agent_id: number; action_type: string }>,
} = {}) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/agents") {
      return Promise.resolve({ data: { data: mockAgents } });
    }
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
    }
    if (url === "/v1/action-configurations") {
      return Promise.resolve({ data: { data: [] } });
    }
    // Connector/schema lookups
    if (url.startsWith("/v1/connectors/")) {
      return Promise.resolve({ data: { id: "email", name: "Email", actions: [] } });
    }
    return Promise.resolve({ data: {} });
  });
}

describe("ReviewApprovalDialog — Always Allow This", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("shows 'Always Allow This' button after successful approval when action has parameters", async () => {
    setupMocks();
    const approval = makeApproval();

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "success",
        execution_result: null,
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Click approve
    await user.click(screen.getByText("Approve"));

    // Wait for success banner
    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    // "Always Allow This" button should be visible
    expect(screen.getByText("Always Allow This")).toBeInTheDocument();
  });

  it("does NOT show 'Always Allow This' when action has no parameters", async () => {
    setupMocks();
    const approval = makeApproval({
      action: { type: "system.noop", version: "1", parameters: {} },
    });

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "success",
        execution_result: null,
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(screen.queryByText("Always Allow This")).not.toBeInTheDocument();
  });

  it("does NOT show 'Always Allow This' when a standing approval already exists for agent+action", async () => {
    setupMocks({
      standingApprovals: [
        { agent_id: 1, action_type: "email.send" },
      ],
    });
    const approval = makeApproval();

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "success",
        execution_result: null,
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(screen.queryByText("Always Allow This")).not.toBeInTheDocument();
  });

  it("does NOT show 'Always Allow This' when execution_status is 'pending'", async () => {
    setupMocks();
    const approval = makeApproval();

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "pending",
        execution_result: null,
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Request Approved")).toBeInTheDocument();
    });

    expect(screen.queryByText("Always Allow This")).not.toBeInTheDocument();
  });

  it("does NOT show 'Always Allow This' when execution_status is 'error'", async () => {
    setupMocks();
    const approval = makeApproval();

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "error",
        execution_result: { execution_error: "Connection failed" },
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Execution Failed")).toBeInTheDocument();
    });

    expect(screen.queryByText("Always Allow This")).not.toBeInTheDocument();
  });

  it("opens CreateStandingApprovalDialog when 'Always Allow This' is clicked", async () => {
    setupMocks();
    const approval = makeApproval();

    mockPost.mockResolvedValueOnce({
      data: {
        approval_id: approval.approval_id,
        status: "approved",
        approved_at: new Date().toISOString(),
        confirmation_code: "ABC-123",
        execution_status: "success",
        execution_result: null,
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Always Allow This")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Always Allow This"));

    // The CreateStandingApprovalDialog should open
    await waitFor(() => {
      expect(screen.getByText("Create Standing Approval")).toBeInTheDocument();
    });
  });
});
