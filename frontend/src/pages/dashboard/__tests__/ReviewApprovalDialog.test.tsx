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

  it("shows 'Always Allow' button on the approval screen when action has parameters", async () => {
    setupMocks();
    const approval = makeApproval();

    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // "Always Allow" button should be visible alongside Approve/Deny
    await waitFor(() => {
      expect(screen.getByText("Always allow this action")).toBeInTheDocument();
    });
    expect(screen.getByText("Approve")).toBeInTheDocument();
    expect(screen.getByText("Deny")).toBeInTheDocument();
  });

  it("does NOT show 'Always Allow' when action has no parameters", () => {
    setupMocks();
    const approval = makeApproval({
      action: { type: "system.noop", version: "1", parameters: {} },
    });

    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    expect(screen.queryByText("Always allow this action")).not.toBeInTheDocument();
  });

  it("does NOT show 'Always Allow' when a standing approval already exists for agent+action", async () => {
    setupMocks({
      standingApprovals: [
        { agent_id: 1, action_type: "email.send" },
      ],
    });
    const approval = makeApproval();

    render(
      <ReviewApprovalDialog
        approval={approval}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Wait for standing approvals to load, then verify button is absent
    await waitFor(() => {
      // Standing approvals have loaded (no loading state)
      expect(screen.getByText("Approve")).toBeInTheDocument();
    });

    expect(screen.queryByText("Always allow this action")).not.toBeInTheDocument();
  });

  it("approves request and opens CreateStandingApprovalDialog when 'Always Allow' is clicked", async () => {
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

    // Click "Always Allow" from the pre-approval screen
    await waitFor(() => {
      expect(screen.getByText("Always allow this action")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Always allow this action"));

    // The CreateStandingApprovalDialog should open (skipping to constraints step)
    await waitFor(() => {
      expect(screen.getByText(/Step 1 of 2/)).toBeInTheDocument();
    });

    // Should show step 1 of 2 (constraints), not step 1 of 4
    expect(screen.getByText(/Step 1 of 2/)).toBeInTheDocument();
  });

  it("does NOT open standing approval wizard when execution fails after 'Always Allow'", async () => {
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

    await waitFor(() => {
      expect(screen.getByText("Always allow this action")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Always allow this action"));

    // Should show execution failure, NOT the standing approval wizard
    await waitFor(() => {
      expect(screen.getByText("Execution Failed")).toBeInTheDocument();
    });

    expect(screen.queryByText(/Step 1 of 2/)).not.toBeInTheDocument();
  });
});
