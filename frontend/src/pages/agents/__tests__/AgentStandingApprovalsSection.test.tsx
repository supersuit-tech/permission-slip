import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import type { StandingApproval } from "@/hooks/useStandingApprovals";
import { AgentStandingApprovalsSection } from "../AgentStandingApprovalsSection";

vi.mock("../../../api/client");

const mockStandingApprovals: StandingApproval[] = [
  {
    standing_approval_id: "sa_test1",
    action_type: "email.send",
    agent_id: 1,
    user_id: "user-1",
    name: "Send company emails",
    action_version: "1",
    status: "active" as const,
    starts_at: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    expires_at: null,
    constraints: { to: { $pattern: "*@mycompany.com" }, subject: "*" },
  },
  {
    standing_approval_id: "sa_test2",
    action_type: "slack.send_message",
    agent_id: 2,
    user_id: "user-1",
    name: "Post to Slack",
    action_version: "1",
    status: "active" as const,
    starts_at: "2026-01-01T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    expires_at: "2099-12-31T00:00:00Z",
    constraints: {},
  },
];

function mockApiFetch(standingApprovals = mockStandingApprovals) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
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

  it("renders standing approvals for the current agent only", async () => {
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
      expect(screen.getByText("Rule")).toBeInTheDocument();
    });
    expect(screen.queryByText("Agent")).not.toBeInTheDocument();
  });

  it("shows Manage link to the connector page", async () => {
    mockApiFetch();

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("link", { name: /Manage/i })).toHaveAttribute(
        "href",
        "/agents/1/connectors/email",
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

  it("falls back to action type when rule has no name", async () => {
    mockApiFetch([
      {
        standing_approval_id: "sa_noname",
        action_type: "email.send",
        agent_id: 1,
        user_id: "user-1",
        name: null,
        description: null,
        action_version: "1",
        status: "active",
        starts_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        expires_at: null,
        constraints: {},
        connector_instance_id: null,
      },
    ]);

    render(<AgentStandingApprovalsSection agentId={1} />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("email.send").length).toBeGreaterThan(0);
    });
  });
});
