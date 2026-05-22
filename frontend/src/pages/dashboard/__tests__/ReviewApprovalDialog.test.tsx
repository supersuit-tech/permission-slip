import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { ReviewApprovalDialog } from "../ReviewApprovalDialog";
import type { ApprovalSummary } from "../../../hooks/useApprovals";

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

const mockActionConfig = {
  id: "ac_config1",
  agent_id: 1,
  connector_id: "email",
  action_type: "email.send",
  status: "active" as const,
  name: "Send email",
  parameters: {},
};

function setupMocks({
  standingApprovals = [] as Array<{ agent_id: number; action_type: string }>,
  actionConfigs = [mockActionConfig],
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
      return Promise.resolve({ data: { data: actionConfigs } });
    }
    if (url.startsWith("/v1/connectors/")) {
      return Promise.resolve({ data: { id: "email", name: "Email", actions: [] } });
    }
    return Promise.resolve({ data: {} });
  });
}

function mockApproveSuccess() {
  mockPost.mockResolvedValueOnce({
    data: {
      approval_id: "appr_test123",
      status: "approved",
      approved_at: new Date().toISOString(),
      confirmation_code: "ABC12-3DEFG",
      execution_status: "success",
      execution_result: null,
    },
  });
}

describe("ReviewApprovalDialog — auto-approve future requests", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("shows checkbox when no matching standing approval exists", async () => {
    setupMocks();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });
  });

  it("shows checkbox for parameterless actions", async () => {
    setupMocks({
      actionConfigs: [
        {
          ...mockActionConfig,
          id: "ac_list",
          action_type: "google.list_calendars",
        },
      ],
    });
    const approval = makeApproval({
      action: { type: "google.list_calendars", version: "1", parameters: {} },
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

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("Always allow this action")).not.toBeInTheDocument();
  });

  it("hides checkbox when a standing approval already exists for agent+action", async () => {
    setupMocks({
      standingApprovals: [{ agent_id: 1, action_type: "email.send" }],
    });

    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(screen.getByText("Approve")).toBeInTheDocument();
    });

    expect(
      screen.queryByLabelText("Auto-approve all future requests like this"),
    ).not.toBeInTheDocument();
  });

  it("ticking checkbox + Approve calls approve then createStandingApproval with pinned params", async () => {
    setupMocks();
    mockApproveSuccess();
    mockPost.mockResolvedValueOnce({
      data: {
        standing_approval_id: "sa_new",
        agent_id: 1,
        action_type: "email.send",
        status: "active",
      },
    });

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Auto-approve all future requests like this"),
    );
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(mockPost).toHaveBeenCalledTimes(2);
    const createCall = mockPost.mock.calls.find(
      (call) => call[0] === "/v1/standing-approvals/create",
    );
    expect(createCall).toBeDefined();
    expect(createCall?.[1]?.body).toMatchObject({
      agent_id: 1,
      action_type: "email.send",
      action_version: "1",
      constraints: {
        recipient: "user@example.com",
        subject: "Hello",
      },
      source_action_configuration_id: "ac_config1",
      expires_at: null,
    });

    expect(
      screen.getByText("Future matching requests will be auto-approved."),
    ).toBeInTheDocument();
  });

  it("standing approval failure does not block approve from succeeding", async () => {
    setupMocks();
    mockApproveSuccess();
    mockPost.mockRejectedValueOnce(new Error("Standing approval failed"));

    const user = userEvent.setup();
    render(
      <ReviewApprovalDialog
        approval={makeApproval()}
        agentDisplayName="Test Bot"
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await settleAuthHydration();

    await waitFor(() => {
      expect(
        screen.getByLabelText("Auto-approve all future requests like this"),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByLabelText("Auto-approve all future requests like this"),
    );
    await user.click(screen.getByText("Approve"));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });

    expect(
      screen.queryByText("Future matching requests will be auto-approved."),
    ).not.toBeInTheDocument();
  });
});
