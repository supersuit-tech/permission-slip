import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { CreateStandingApprovalDialog } from "../CreateStandingApprovalDialog";
import type { Agent } from "../../../hooks/useAgents";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockAgents: Agent[] = [
  {
    agent_id: 1,
    status: "registered",
    metadata: { name: "Test Bot" },
    confirmation_code: null,
    expires_at: null,
    created_at: "2026-01-01T00:00:00Z",
  },
  {
    agent_id: 2,
    status: "registered",
    metadata: { name: "Deploy Bot" },
    confirmation_code: null,
    expires_at: null,
    created_at: "2026-01-01T00:00:00Z",
  },
];

const mockConfigs = [
  {
    id: "ac_config1",
    agent_id: 1,
    connector_id: "github",
    action_type: "github.create_issue",
    parameters: { repo: "supersuit-tech/webapp", title: "*", body: "*" },
    status: "active" as const,
    name: "Create bug issues",
    description: "Create issues in the webapp repo",
    created_at: "2026-01-01T00:00:00Z",
  },
];

function setupMocks() {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string, opts?: { params?: { query?: { agent_id?: number } } }) => {
    if (url === "/v1/action-configurations") {
      const agentId = opts?.params?.query?.agent_id;
      if (agentId === 1) {
        return Promise.resolve({ data: { data: mockConfigs } });
      }
      return Promise.resolve({ data: { data: [] } });
    }
    if (url === "/v1/connectors/github") {
      return Promise.resolve({
        data: {
          id: "github",
          name: "GitHub",
          actions: [
            {
              action_type: "github.create_issue",
              operation_type: "write",
              name: "Create Issue",
              parameters_schema: {
                type: "object",
                properties: {
                  repo: { type: "string", description: "Repository" },
                  title: { type: "string", description: "Issue title" },
                  body: { type: "string", description: "Issue body" },
                },
                required: ["repo"],
              },
            },
          ],
        },
      });
    }
    return Promise.resolve({ data: {} });
  });
}

describe("CreateStandingApprovalDialog", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
    setupMocks();
  });

  it("renders step 1 with agent dropdown", () => {
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    expect(screen.getByText("Create Standing Approval")).toBeInTheDocument();
    expect(screen.getByText(/Step 1 of 4/)).toBeInTheDocument();
    expect(screen.getByText("Test Bot")).toBeInTheDocument();
    expect(screen.getByText("Deploy Bot")).toBeInTheDocument();
  });

  it("shows agent display names instead of IDs", () => {
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Should show display names, not just IDs
    expect(screen.getByText("Test Bot")).toBeInTheDocument();
    expect(screen.getByText("Deploy Bot")).toBeInTheDocument();
  });

  it("navigates to step 2 after selecting an agent", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Select agent
    await user.selectOptions(screen.getByLabelText("Agent"), "1");
    // Click Next
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 4/)).toBeInTheDocument();
    });
  });

  it("shows error when trying to advance without selecting agent", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.click(screen.getByText("Next"));
    // Should still be on step 1
    expect(screen.getByText(/Step 1 of 4/)).toBeInTheDocument();
  });

  it("shows action configs grouped by connector on step 2", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Select agent
    await user.selectOptions(screen.getByLabelText("Agent"), "1");
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 4/)).toBeInTheDocument();
    });

    // Wait for configs to load
    await waitFor(() => {
      expect(
        screen.getByText("Create bug issues (github.create_issue)"),
      ).toBeInTheDocument();
    });

    expect(
      screen.queryByText("Custom action type..."),
    ).not.toBeInTheDocument();
  });

  it("navigates back from step 2 to step 1", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.selectOptions(screen.getByLabelText("Agent"), "1");
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 4/)).toBeInTheDocument();
    });

    await user.click(screen.getByText("Back"));

    expect(screen.getByText(/Step 1 of 4/)).toBeInTheDocument();
  });

  it("resets form on dialog close", async () => {
    const onOpenChange = vi.fn();
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={onOpenChange}
      />,
      { wrapper },
    );

    await user.selectOptions(screen.getByLabelText("Agent"), "1");
    await user.click(screen.getByText("Cancel"));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("shows helper message about constraints on step 3", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    // Navigate to step 3
    await user.selectOptions(screen.getByLabelText("Agent"), "1");
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 4/)).toBeInTheDocument();
    });

    // Wait for configs and select one
    await waitFor(() => {
      expect(
        screen.getByText("Create bug issues (github.create_issue)"),
      ).toBeInTheDocument();
    });

    await user.selectOptions(
      screen.getByLabelText("Action Configuration"),
      "ac_config1",
    );
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 3 of 4/)).toBeInTheDocument();
    });

    // Helper message should be visible
    expect(
      screen.getByText(/Standing approvals require parameter constraints/),
    ).toBeInTheDocument();
  });

  it("accepts initial props and skips to constraints step", () => {
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
        initialAgentId={1}
        initialActionType="email.send"
        initialConstraints={{ recipient: "*@mycompany.com" }}
      />,
      { wrapper },
    );

    // Should skip to step 3 (constraints), shown as step 1 of 2
    expect(screen.getByText(/Step 1 of 2/)).toBeInTheDocument();
    expect(screen.getByText(/Set Constraints/)).toBeInTheDocument();
  });

  it("auto-selected matching config submits create with source_action_configuration_id", async () => {
    const user = userEvent.setup();
    mockPost.mockResolvedValue({ data: { standing_approval_id: "sa_test" } });

    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
        initialAgentId={1}
        initialActionType="github.create_issue"
        initialConstraints={{
          repo: "supersuit-tech/webapp",
          title: "*",
          body: "*",
        }}
      />,
      { wrapper },
    );

    await waitFor(() => {
      expect(
        screen.getByText(/Standing approvals require parameter constraints/),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 2/)).toBeInTheDocument();
      expect(screen.getByText(/Expiry/)).toBeInTheDocument();
    });

    await user.click(screen.getByText("Create"));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalledWith(
        "/v1/standing-approvals/create",
        expect.objectContaining({
          body: expect.objectContaining({
            agent_id: 1,
            action_type: "github.create_issue",
            source_action_configuration_id: "ac_config1",
          }),
        }),
      );
    });
  });

  it("shows empty state on step 2 when agent has no action configurations", async () => {
    const user = userEvent.setup();
    render(
      <CreateStandingApprovalDialog
        agents={mockAgents}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    await user.selectOptions(screen.getByLabelText("Agent"), "2");
    await user.click(screen.getByText("Next"));

    await waitFor(() => {
      expect(screen.getByText(/Step 2 of 4/)).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        /No active action configurations found for this agent/,
      ),
    ).toBeInTheDocument();
  });

  it("filters out deactivated agents", () => {
    const agentsWithDeactivated: Agent[] = [
      ...mockAgents,
      {
        agent_id: 3,
        status: "deactivated",
        metadata: { name: "Old Bot" },
        confirmation_code: null,
        expires_at: null,
        created_at: "2026-01-01T00:00:00Z",
      },
    ];

    render(
      <CreateStandingApprovalDialog
        agents={agentsWithDeactivated}
        open={true}
        onOpenChange={vi.fn()}
      />,
      { wrapper },
    );

    expect(screen.queryByText("Old Bot")).not.toBeInTheDocument();
  });
});
