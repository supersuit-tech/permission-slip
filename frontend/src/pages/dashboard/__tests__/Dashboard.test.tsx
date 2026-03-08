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

/** Mock GET so /v1/agents returns agents while other endpoints return empty. */
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

    await waitFor(() => {
      expect(screen.getByText("Registered Agents")).toBeInTheDocument();
    });
    expect(screen.getByText("Recent Activity")).toBeInTheDocument();

    // Onboarding hero should NOT appear
    expect(
      screen.queryByText("Control what your AI agents can do"),
    ).not.toBeInTheDocument();
  });

  it("opens invite dialog when onboarding CTA is clicked", async () => {
    setupAuthMocks({ authenticated: true });
    mockEmptyResponses();

    render(<Dashboard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Register Your First Agent" }),
      ).toBeInTheDocument();
    });

    await userEvent.click(
      screen.getByRole("button", { name: "Register Your First Agent" }),
    );

    await waitFor(() => {
      expect(screen.getByText("Add an Agent")).toBeInTheDocument();
    });
  });
});
