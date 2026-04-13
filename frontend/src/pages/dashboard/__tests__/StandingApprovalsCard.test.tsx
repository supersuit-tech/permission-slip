import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { StandingApprovalsCard } from "../StandingApprovalsCard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockStandingApprovals = [
  {
    standing_approval_id: "sa_test1",
    action_type: "email.send",
    agent_id: 1,
    status: "active" as const,
    execution_count: 3,
    max_executions: 10,
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

const freePlanLimits = {
  max_requests_per_month: 1000 as number | null,
  max_agents: 3 as number | null,
  max_standing_approvals: 5 as number | null,
  max_credentials: 5 as number | null,
  audit_retention_days: 7,
};

const freePlanResponse = {
  plan: {
    id: "free",
    name: "Free",
    ...freePlanLimits,
  },
  effective_limits: freePlanLimits,
  subscription: {
    status: "active",
    can_upgrade: true,
    can_downgrade: false,
    can_end_quota_grace_now: false,
  },
  usage: { requests: 10, agents: 2, standing_approvals: 1, credentials: 0 },
};

const paidEffectiveLimits = {
  max_requests_per_month: null as number | null,
  max_agents: null as number | null,
  max_standing_approvals: null as number | null,
  max_credentials: null as number | null,
  audit_retention_days: 90,
};

function mockApiFetch(
  standingApprovals = mockStandingApprovals,
  billingPlan = freePlanResponse,
  agents = mockAgents,
  actionConfigs = mockActionConfigs,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: billingPlan });
    }
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

  it("shows limit badge with plan info", async () => {
    mockApiFetch();

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("1 / 5 standing approvals"),
      ).toBeInTheDocument();
    });
  });

  it("shows upgrade prompt when at standing approval limit", async () => {
    const atLimitPlan = {
      ...freePlanResponse,
      usage: { ...freePlanResponse.usage, standing_approvals: 5 },
    };
    mockApiFetch(mockStandingApprovals, atLimitPlan);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/Upgrade to create more standing approvals/),
      ).toBeInTheDocument();
    });
  });

  it("shows no limit for paid plan", async () => {
    const paidPlan = {
      ...freePlanResponse,
      plan: { ...freePlanResponse.plan, id: "pay_as_you_go", max_standing_approvals: null },
      effective_limits: paidEffectiveLimits,
      usage: { ...freePlanResponse.usage, standing_approvals: 10 },
    };
    mockApiFetch(mockStandingApprovals, paidPlan);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("10 standing approvals")).toBeInTheDocument();
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
    mockApiFetch(mockStandingApprovals, freePlanResponse, agentsNoName);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent 1")).toBeInTheDocument();
    });
  });

  it("shows unknown configuration when source config is not found", async () => {
    mockApiFetch(mockStandingApprovals, freePlanResponse, mockAgents, []);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Unknown configuration")).toBeInTheDocument();
    });
  });
});
