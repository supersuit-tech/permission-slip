import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { AgentStandingApprovalsSection } from "../AgentStandingApprovalsSection";

vi.mock("../../../api/client");

const mockActionConfigs = [
  {
    id: "ac_config1",
    agent_id: 1,
    connector_id: "gmail",
    action_type: "email.send",
    parameters: {},
    status: "active" as const,
    name: "Send company emails",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
  {
    id: "ac_config2",
    agent_id: 2,
    connector_id: "slack",
    action_type: "slack.send_message",
    parameters: {},
    status: "active" as const,
    name: "Post to Slack",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  },
];

const mockStandingApprovals = [
  {
    standing_approval_id: "sa_test1",
    action_type: "email.send",
    agent_id: 1,
    status: "active" as const,
    expires_at: null,
    constraints: { to: { $pattern: "*@mycompany.com" }, subject: "*" },
    source_action_configuration_id: "ac_config1",
  },
  {
    standing_approval_id: "sa_test2",
    action_type: "slack.send_message",
    agent_id: 2,
    status: "active" as const,
    expires_at: "2099-12-31T00:00:00Z",
    constraints: {},
    source_action_configuration_id: "ac_config2",
  },
];

function mockApiFetch(
  standingApprovals = mockStandingApprovals,
  actionConfigs = mockActionConfigs,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
    }
    if (url === "/v1/action-configurations") {
      return Promise.resolve({ data: { data: actionConfigs } });
    }
    return Promise.resolve({ data: { data: [], has_more: false } });
  });
}

describe("AgentStandingApprovalsSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders linked standing approval for the current agent only", async () => {
    mockApiFetch();

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Send company emails")).toBeInTheDocument();
    });
    expect(screen.queryByText("Post to Slack")).not.toBeInTheDocument();
  });

  it("does not show Agent column", async () => {
    mockApiFetch();

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Configuration")).toBeInTheDocument();
    });
    expect(screen.queryByText("Agent")).not.toBeInTheDocument();
  });

  it("shows Manage link to the connector page", async () => {
    mockApiFetch();

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("link", { name: /Manage/i })).toHaveAttribute(
        "href",
        "/agents/1/connectors/gmail",
      );
    });
  });

  it("renders empty state when agent has no standing approvals", async () => {
    mockApiFetch([]);

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("No active standing approvals"),
      ).toBeInTheDocument();
    });
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/standing-approvals") {
        return Promise.reject(new Error("Failed to load standing approvals"));
      }
      return Promise.resolve({ data: { data: [] } });
    });

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(
          "Unable to load standing approvals. Please try again later.",
        ),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders loading state", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/standing-approvals") {
        return new Promise(() => {});
      }
      return Promise.resolve({ data: { data: [] } });
    });

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(document.querySelector(".animate-spin")).toBeInTheDocument();
    });
    expect(
      screen.queryByText("No active standing approvals"),
    ).not.toBeInTheDocument();
  });

  it("shows unknown configuration when source config is not found", async () => {
    mockApiFetch(mockStandingApprovals.filter((sa) => sa.agent_id === 1), []);

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Unknown configuration")).toBeInTheDocument();
    });
  });
});
