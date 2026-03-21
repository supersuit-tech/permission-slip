import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { RegisteredAgentsCard } from "../RegisteredAgentsCard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockAgentsResponse = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      metadata: { name: "My Bot" },
      last_active_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      request_count_30d: 42,
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      agent_id: 2,
      status: "registered",
      request_count_30d: 0,
      created_at: "2026-02-01T00:00:00Z",
    },
  ],
  has_more: false,
};

const emptyResponse = { data: [], has_more: false };

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

function mockAgentsFetch(response = mockAgentsResponse, billingPlan = freePlanResponse) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: billingPlan });
    }
    return Promise.resolve({ data: response });
  });
}

describe("RegisteredAgentsCard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders title", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Registered Agents")).toBeInTheDocument();
    });
  });

  it("renders table headers when agents exist", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent")).toBeInTheDocument();
    });
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Last Active")).toBeInTheDocument();
    expect(screen.getByText("Requests")).toBeInTheDocument();
  });

  it("renders agent rows with display names", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("My Bot")).toBeInTheDocument();
    });
    // Agent without metadata.name shows "Agent <id>"
    expect(screen.getByText("Agent 2")).toBeInTheDocument();
  });

  it("defaults to showing only active (registered) agents", async () => {
    const responseWithMixed = {
      data: [
        {
          agent_id: 1,
          status: "registered",
          request_count_30d: 42,
          created_at: "2026-01-01T00:00:00Z",
        },
        {
          agent_id: 2,
          status: "pending",
          request_count_30d: 0,
          created_at: "2026-02-01T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithMixed);

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent 1")).toBeInTheDocument();
    });
    // Pending agent should not be visible in default "active" filter
    expect(screen.queryByText("Agent 2")).not.toBeInTheDocument();
  });

  it("renders Add an Agent button when agents exist", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Add an Agent" })
      ).toBeInTheDocument();
    });
  });

  it("renders empty state when no agents", async () => {
    mockAgentsFetch(emptyResponse);

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("No registered agents yet")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Register an agent to start controlling what it can do/)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Add an Agent" })
    ).toBeInTheDocument();
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Failed to load agents"));

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Unable to load agents. Please try again later."),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Retry" })
    ).toBeInTheDocument();
  });

  it("renders request count for agents", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("42")).toBeInTheDocument();
    });
  });

  it("renders Configure links instead of Deactivate buttons", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      const configureLinks = screen.getAllByRole("link", { name: /Configure/ });
      expect(configureLinks).toHaveLength(2);
    });
    expect(screen.queryByRole("button", { name: "Deactivate" })).not.toBeInTheDocument();
  });

  it("Configure link navigates to agent config page", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      const configureLinks = screen.getAllByRole("link", { name: /Configure/ });
      expect(configureLinks[0]).toHaveAttribute("href", "/agents/1");
    });
  });

  it("shows status filter tabs", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Active" })).toBeInTheDocument();
    });
    expect(screen.getByRole("radio", { name: "Pending" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Deactivated" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "All" })).toBeInTheDocument();
  });

  it("filters agents by status when clicking filter tabs", async () => {
    const responseWithMixed = {
      data: [
        {
          agent_id: 1,
          status: "registered",
          request_count_30d: 42,
          created_at: "2026-01-01T00:00:00Z",
        },
        {
          agent_id: 2,
          status: "pending",
          request_count_30d: 0,
          created_at: "2026-02-01T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithMixed);

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    // Default: shows only active (registered)
    await waitFor(() => {
      expect(screen.getByText("Agent 1")).toBeInTheDocument();
    });
    expect(screen.queryByText("Agent 2")).not.toBeInTheDocument();

    // Click "All" to show everything
    await user.click(screen.getByRole("radio", { name: "All" }));
    await waitFor(() => {
      expect(screen.getByText("Agent 2")).toBeInTheDocument();
    });
    expect(screen.getByText("Agent 1")).toBeInTheDocument();
  });

  it("opens invite dialog when Add an Agent is clicked", async () => {
    mockAgentsFetch();

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Add an Agent" })
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Add an Agent" }));

    await waitFor(() => {
      expect(
        screen.getByText(/Generate a one-time invite command/)
      ).toBeInTheDocument();
    });
  });

  it("renders confirmation code badge for pending agents", async () => {
    const responseWithPending = {
      data: [
        {
          agent_id: 3,
          status: "pending",
          confirmation_code: "XK7-M9P",
          expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
          request_count_30d: 0,
          created_at: "2026-02-20T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithPending);

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    // Switch to "Pending" tab to see the pending agent.
    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Pending" })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("radio", { name: "Pending" }));

    await waitFor(() => {
      expect(screen.getByText("XK7-M9P")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Copy confirmation code" })
    ).toBeInTheDocument();
  });

  it("shows expiration time for pending agents", async () => {
    const responseWithPending = {
      data: [
        {
          agent_id: 5,
          status: "pending",
          confirmation_code: "QR5-ST6",
          expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
          request_count_30d: 0,
          created_at: "2026-02-20T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithPending);

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Pending" })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("radio", { name: "Pending" }));

    await waitFor(() => {
      expect(screen.getByText(/Expires/)).toBeInTheDocument();
    });
  });

  it("opens review dialog when Review button is clicked for pending agent", async () => {
    const responseWithPending = {
      data: [
        {
          agent_id: 3,
          status: "pending",
          confirmation_code: "XK7-M9P",
          expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
          request_count_30d: 0,
          created_at: "2026-02-20T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithPending);

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Pending" })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("radio", { name: "Pending" }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Review/ })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("button", { name: /Review/ }));

    await waitFor(() => {
      expect(screen.getByText("Complete Agent Registration")).toBeInTheDocument();
    });
    expect(
      screen.getByText("Confirmation code (included in the command below)"),
    ).toBeInTheDocument();
  });

  it("shows limit badge with plan info", async () => {
    mockAgentsFetch();

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("2 / 3 agents")).toBeInTheDocument();
    });
  });

  it("shows upgrade prompt when at agent limit", async () => {
    const atLimitPlan = {
      ...freePlanResponse,
      usage: { ...freePlanResponse.usage, agents: 3 },
    };
    mockAgentsFetch(mockAgentsResponse, atLimitPlan);

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/Upgrade to add more agents/),
      ).toBeInTheDocument();
    });
    // "Add an Agent" button should not be present when at limit
    expect(
      screen.queryByRole("button", { name: "Add an Agent" }),
    ).not.toBeInTheDocument();
  });

  it("shows no limit badge for paid plan", async () => {
    const paidPlan = {
      plan: { ...freePlanResponse.plan, id: "pay_as_you_go", max_agents: null },
      effective_limits: paidEffectiveLimits,
      subscription: {
        status: "active",
        can_upgrade: false,
        can_downgrade: true,
        can_end_quota_grace_now: false,
      },
      usage: { requests: 10, agents: 5, standing_approvals: 1, credentials: 0 },
    };
    mockAgentsFetch(mockAgentsResponse, paidPlan);

    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("5 agents")).toBeInTheDocument();
    });
  });

  it("shows 'Awaiting verification' for pending agents without expiry", async () => {
    const responseWithPending = {
      data: [
        {
          agent_id: 6,
          status: "pending",
          request_count_30d: 0,
          created_at: "2026-02-20T00:00:00Z",
        },
      ],
      has_more: false,
    };
    mockAgentsFetch(responseWithPending);

    const user = userEvent.setup();
    render(<RegisteredAgentsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Pending" })).toBeInTheDocument();
    });
    await user.click(screen.getByRole("radio", { name: "Pending" }));

    await waitFor(() => {
      expect(screen.getByText("Awaiting verification")).toBeInTheDocument();
    });
  });
});
