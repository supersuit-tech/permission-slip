import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { StandingApprovalsCard } from "../StandingApprovalsCard";

vi.mock("../../../api/client");

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
];

const mockAgents = [
  {
    agent_id: 1,
    status: "registered" as const,
    metadata: { name: "Email Bot" } as Record<string, unknown> | null,
    created_at: "2026-01-01T00:00:00Z",
  },
];

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
];

function mockApiFetch(
  standingApprovals = mockStandingApprovals,
  agents = mockAgents,
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
    return Promise.resolve({ data: { data: agents, has_more: false } });
  });
}

describe("StandingApprovalsCard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("shows standing approval count badge", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("1 standing approvals")).toBeInTheDocument();
    });
  });

  it("shows agent display name from metadata", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Email Bot")).toBeInTheDocument();
    });
  });

  it("shows Configuration column header", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Configuration")).toBeInTheDocument();
    });
  });

  it("shows Manage link for each row", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("link", { name: /Manage/i })).toBeInTheDocument();
    });
  });

  it("shows source action config name when available", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Send company emails")).toBeInTheDocument();
    });
  });

  it("falls back to Agent ID when no metadata name", async () => {
    const agentsNoName = [
      { agent_id: 1, status: "registered" as const, metadata: null as Record<string, unknown> | null, created_at: "2026-01-01T00:00:00Z" },
    ];
    mockApiFetch(mockStandingApprovals, agentsNoName);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent 1")).toBeInTheDocument();
    });
  });

  it("shows unknown configuration when source config is not found", async () => {
    mockApiFetch(mockStandingApprovals, mockAgents, []);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Unknown configuration")).toBeInTheDocument();
    });
  });
});
