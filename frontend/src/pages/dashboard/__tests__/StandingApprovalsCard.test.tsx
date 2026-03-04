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
    standing_approval_id: 1,
    action_type: "email.send",
    agent_id: 1,
    execution_count: 3,
    max_executions: 10,
    expires_at: null,
  },
];

const freePlanResponse = {
  plan: {
    id: "free",
    name: "Free",
    max_requests_per_month: 1000 as number | null,
    max_agents: 3 as number | null,
    max_standing_approvals: 5 as number | null,
    max_credentials: 5 as number | null,
    audit_retention_days: 7,
  },
  subscription: { status: "active", can_upgrade: true, can_downgrade: false },
  usage: { requests: 10, agents: 2, standing_approvals: 1, credentials: 0 },
};

function mockApiFetch(
  standingApprovals = mockStandingApprovals,
  billingPlan = freePlanResponse,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: billingPlan });
    }
    if (url === "/v1/standing-approvals") {
      return Promise.resolve({ data: { data: standingApprovals } });
    }
    // agents endpoint
    return Promise.resolve({ data: { data: [], has_more: false } });
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
      usage: { ...freePlanResponse.usage, standing_approvals: 10 },
    };
    mockApiFetch(mockStandingApprovals, paidPlan);

    render(<StandingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("10 standing approvals")).toBeInTheDocument();
    });
  });
});
