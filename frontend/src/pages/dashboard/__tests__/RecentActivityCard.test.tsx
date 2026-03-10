import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import {
  FIXED_NOW,
  mockAuditEventsResponse,
  emptyAuditResponse,
  type MockAuditResponse,
} from "../../../lib/__tests__/auditEventFixtures";
import { RecentActivityCard } from "../RecentActivityCard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockAuditFetch(response: MockAuditResponse = mockAuditEventsResponse) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockResolvedValue({ data: response });
}

describe("RecentActivityCard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(FIXED_NOW);
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders title", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Recent Activity")).toBeInTheDocument();
    });
  });

  it("renders table headers when events exist", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("When")).toBeInTheDocument();
    });
    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("Action")).toBeInTheDocument();
    expect(screen.getByText("Outcome")).toBeInTheDocument();
  });

  it("renders agent names from metadata", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("My Bot")).toHaveLength(2);
    });
    expect(screen.getByText("Data Agent")).toBeInTheDocument();
    expect(screen.getByText("New Agent")).toBeInTheDocument();
  });

  it("renders outcome badges", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      // "Approved" appears once as a filter tab and once as a badge
      expect(screen.getAllByText("Approved")).toHaveLength(2);
    });
    // "Denied" appears once as a filter tab and once as a badge
    expect(screen.getAllByText("Denied")).toHaveLength(2);
    // "Auto-executed" appears once as a filter tab and once as a badge
    expect(screen.getAllByText("Auto-executed")).toHaveLength(2);
    expect(screen.getByText("Registered")).toBeInTheDocument();
  });

  it("renders action summaries with parameters", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/email\.send \(to: alice@example\.com/),
      ).toBeInTheDocument();
    });
  });

  it("renders Agent registered text for agent lifecycle events", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent registered")).toBeInTheDocument();
    });
  });

  it("renders empty state when no events", async () => {
    mockAuditFetch(emptyAuditResponse);

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("No activity yet")).toBeInTheDocument();
    });
    expect(
      screen.getByText(
        /Every approval, denial, and agent action is logged here/,
      ),
    ).toBeInTheDocument();
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Failed to load activity"));

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Unable to load activity. Please try again later."),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders loading state", () => {
    setupAuthMocks({ authenticated: true });
    // Don't resolve the mock so it stays loading
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<RecentActivityCard />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading activity" }),
    ).toBeInTheDocument();
  });

  it("shows filter tabs", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("tab", { name: "All" })).toBeInTheDocument();
    });
    expect(
      screen.getByRole("tab", { name: "Approved" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Denied" })).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "Auto-executed" }),
    ).toBeInTheDocument();
  });

  it("calls API with outcome filter when tab is clicked", async () => {
    mockAuditFetch();

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("My Bot")).toHaveLength(2);
    });

    await user.click(screen.getByRole("tab", { name: "Denied" }));

    await waitFor(() => {
      // The mock should have been called with the outcome filter
      const calls = mockGet.mock.calls;
      const lastCall = calls[calls.length - 1] as unknown[];
      expect(lastCall?.[0]).toBe("/v1/audit-events");
      const opts = lastCall?.[1] as { params?: { query?: { outcome?: string } } };
      expect(opts?.params?.query?.outcome).toBe("denied");
    });
  });

  it("shows View All Activity link when has_more is true", async () => {
    mockAuditFetch();

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      const link = screen.getByRole("link", { name: "View All Activity" });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute("href", "/activity");
    });
  });

  it("hides View All Activity link when has_more is false", async () => {
    mockAuditFetch({ ...mockAuditEventsResponse, has_more: false });

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("My Bot")).toHaveLength(2);
    });
    expect(screen.queryByRole("link", { name: "View All Activity" })).not.toBeInTheDocument();
  });

  it("falls back to Agent <id> when metadata has no name", async () => {
    mockAuditFetch({
      data: [
        {
          event_type: "approval.approved",
          timestamp: "2026-02-20T11:59:00Z",
          agent_id: 99,
          agent_metadata: null,
          action: { type: "test.action", version: "1", parameters: { key: "val" } },
          outcome: "approved",
        },
      ],
      has_more: false,
    });

    render(<RecentActivityCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent 99")).toBeInTheDocument();
    });
  });
});
