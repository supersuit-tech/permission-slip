import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { Dashboard } from "../Dashboard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

/** Mock response that returns agents, approvals, etc. as empty lists. */
function mockEmptyResponses() {
  mockGet.mockResolvedValue({ data: { data: [], has_more: false } });
}

/** Mock GET so /v1/agents returns a configured agent (with connectors) and activity. */
function mockWithAgents() {
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/agents") {
      return Promise.resolve({
        data: {
          data: [
            {
              agent_id: 1,
              status: "registered",
              last_active_at: "2025-01-01T00:00:00Z",
              request_count_30d: 5,
              metadata: { name: "Test Agent" },
            },
          ],
          has_more: false,
        },
      });
    }
    if (url === "/v1/agents/{agent_id}/connectors") {
      return Promise.resolve({
        data: {
          data: [
            {
              id: "github",
              name: "GitHub",
              actions: ["github.create_issue"],
              required_credentials: [],
              enabled_at: "2026-01-01T00:00:00Z",
            },
          ],
        },
      });
    }
    if (url === "/v1/audit-events") {
      return Promise.resolve({
        data: {
          data: [
            {
              event_type: "approval.approved",
              timestamp: "2025-01-01T00:00:00Z",
              agent_id: 1,
              agent_metadata: { name: "Test Agent" },
              action: { type: "test.action", version: "1", parameters: {} },
              outcome: "approved",
            },
          ],
          has_more: false,
        },
      });
    }
    return Promise.resolve({ data: { data: [], has_more: false } });
  });
}

/** Mock GET so /v1/agents returns a single registered agent with no connectors. */
function mockUnconfiguredAgent() {
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/agents") {
      return Promise.resolve({
        data: {
          data: [
            {
              agent_id: 1,
              status: "registered",
              last_active_at: null,
              request_count_30d: 0,
              metadata: { name: "My Agent" },
            },
          ],
          has_more: false,
        },
      });
    }
    if (url === "/v1/agents/{agent_id}/connectors") {
      return Promise.resolve({ data: { data: [] } });
    }
    return Promise.resolve({ data: { data: [], has_more: false } });
  });
}

describe("Dashboard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders onboarding hero when no agents exist", async () => {
    setupAuthMocks({ authenticated: true });
    mockEmptyResponses();

    render(<Dashboard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Control what your AI agents can do"),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Register Your First Agent" }),
    ).toBeInTheDocument();

    // Normal dashboard cards should NOT appear
    expect(screen.queryByText("Recent Activity")).not.toBeInTheDocument();
    expect(screen.queryByText("Registered Agents")).not.toBeInTheDocument();
  });

  it("renders all dashboard cards when agents exist", async () => {
    setupAuthMocks({ authenticated: true });
    mockWithAgents();

    render(<Dashboard />, { wrapper });

    // Wait for both agent and connector fetches to complete so unconfiguredLoading
    // is false and all dashboard cards are rendered.
    await waitFor(() => {
      expect(screen.getByText("Registered Agents")).toBeInTheDocument();
      expect(screen.getByText("Pending Approvals")).toBeInTheDocument();
    });
    expect(screen.getByText("Recent Activity")).toBeInTheDocument();
    expect(screen.getByText("Standing Approvals")).toBeInTheDocument();

    // Onboarding hero should NOT appear
    expect(
      screen.queryByText("Control what your AI agents can do"),
    ).not.toBeInTheDocument();
  });

  it("opens invite dialog when onboarding CTA is clicked", async () => {
    setupAuthMocks({ authenticated: true });
    mockEmptyResponses();

    const user = userEvent.setup();
    render(<Dashboard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Register Your First Agent" }),
      ).toBeInTheDocument();
    });

    await user.click(
      screen.getByRole("button", { name: "Register Your First Agent" }),
    );

    await waitFor(() => {
      expect(screen.getByText("Add an Agent")).toBeInTheDocument();
    });
  });

  it("renders config hero when single agent has no connectors", async () => {
    setupAuthMocks({ authenticated: true });
    mockUnconfiguredAgent();

    render(<Dashboard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/My Agent is ready.*now give it superpowers/),
      ).toBeInTheDocument();
    });

    // Should show the agent card but not the other dashboard cards
    expect(screen.getByText("Registered Agents")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Configure My Agent" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Pending Approvals")).not.toBeInTheDocument();
    expect(screen.queryByText("Recent Activity")).not.toBeInTheDocument();
    expect(screen.queryByText("Standing Approvals")).not.toBeInTheDocument();
  });
});
