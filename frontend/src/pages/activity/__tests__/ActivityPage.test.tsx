import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import {
  FIXED_NOW,
  mockAuditEventsResponse,
  emptyAuditResponse,
  getLastAuditQuery,
  type MockAuditResponse,
} from "../../../lib/__tests__/auditEventFixtures";
import { ActivityPage } from "../ActivityPage";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const mockAgents = {
  data: [
    { agent_id: 1, metadata: { name: "My Bot" }, status: "registered" },
    { agent_id: 2, metadata: { name: "Data Agent" }, status: "registered" },
  ],
};

/**
 * Set up the API mock for both audit events and agents endpoints.
 * The mock distinguishes calls by the URL argument.
 */
function mockApiFetch(auditResponse: MockAuditResponse = mockAuditEventsResponse) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/audit-events") {
      return Promise.resolve({ data: auditResponse });
    }
    if (url === "/v1/agents") {
      return Promise.resolve({ data: mockAgents });
    }
    return Promise.resolve({ data: null });
  });
}

describe("ActivityPage", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(FIXED_NOW);
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/activity"]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders page title and back link", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Activity Log")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("link", { name: "Back to Dashboard" }),
    ).toHaveAttribute("href", "/");
  });

  it("renders table headers when events exist", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Timestamp")).toBeInTheDocument();
    });
    // "Agent" appears as both a filter label and a column header
    expect(screen.getAllByText("Agent").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Action")).toBeInTheDocument();
    expect(screen.getByText("Outcome")).toBeInTheDocument();

    // Verify the table header specifically
    const table = screen.getByRole("table");
    const headers = within(table).getAllByRole("columnheader");
    const headerTexts = headers.map((h) => h.textContent);
    expect(headerTexts).toEqual(["Timestamp", "Agent", "Action", "Outcome"]);
  });

  it("renders agent names from metadata", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    // "My Bot" appears in 2 table rows + possibly 1 dropdown option
    await waitFor(() => {
      expect(screen.getAllByText("My Bot").length).toBeGreaterThanOrEqual(2);
    });
    // "Data Agent" appears in 1 table row + possibly 1 dropdown option
    expect(screen.getAllByText("Data Agent").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("New Agent")).toBeInTheDocument();
  });

  it("renders outcome badges", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      // "Approved" appears as a filter tab and a badge
      expect(screen.getAllByText("Approved").length).toBeGreaterThanOrEqual(2);
    });
    // "Denied" appears as a filter tab and a badge
    expect(screen.getAllByText("Denied").length).toBeGreaterThanOrEqual(2);
  });

  it("renders action summaries with parameters", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/email\.send \(to: alice@example\.com/),
      ).toBeInTheDocument();
    });
  });

  it("renders empty state when no events", async () => {
    mockApiFetch(emptyAuditResponse);

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("No activity found")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Try adjusting your filters/),
    ).toBeInTheDocument();
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Failed to load activity"));

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Unable to load activity. Please try again later."),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders loading state", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<ActivityPage />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading activity" }),
    ).toBeInTheDocument();
  });

  it("shows Load More button when has_more is true", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Load More" }),
      ).toBeInTheDocument();
    });
  });

  it("does not show Load More when has_more is false", async () => {
    mockApiFetch({ ...emptyAuditResponse, data: mockAuditEventsResponse.data, has_more: false });

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("My Bot").length).toBeGreaterThanOrEqual(2);
    });
    expect(screen.queryByRole("button", { name: "Load More" })).not.toBeInTheDocument();
  });

  it("shows outcome filter tabs including all outcomes", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "All" })).toBeInTheDocument();
    });
    expect(screen.getByRole("radio", { name: "Approved" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Denied" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Auto-executed" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Cancelled" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Registered" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Deactivated" })).toBeInTheDocument();
  });

  it("calls API with outcome filter when tab is clicked", async () => {
    mockApiFetch();

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByText("My Bot").length).toBeGreaterThanOrEqual(2);
    });

    await user.click(screen.getByRole("radio", { name: "Denied" }));

    await waitFor(() => {
      expect(getLastAuditQuery(mockGet)?.outcome).toBe("denied");
    });
  });

  it("shows agent filter dropdown populated with agents", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Agent")).toBeInTheDocument();
    });

    const select = screen.getByLabelText("Agent");
    const options = within(select).getAllByRole("option");
    expect(options[0]).toHaveTextContent("All Agents");
    // Agent options may take a moment to load
    await waitFor(() => {
      const agentOptions = within(select).getAllByRole("option");
      expect(agentOptions.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("calls API with agent_id filter when agent is selected", async () => {
    mockApiFetch();

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Agent")).toBeInTheDocument();
    });

    // Wait for agents to load in dropdown
    await waitFor(() => {
      const select = screen.getByLabelText("Agent");
      const options = within(select).getAllByRole("option");
      expect(options.length).toBeGreaterThanOrEqual(3); // All + 2 agents
    });

    await user.selectOptions(screen.getByLabelText("Agent"), "1");

    await waitFor(() => {
      expect(getLastAuditQuery(mockGet)?.agent_id).toBe(1);
    });
  });

  it("shows event type filter dropdown", async () => {
    mockApiFetch();

    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Event Type")).toBeInTheDocument();
    });

    const select = screen.getByLabelText("Event Type");
    const options = within(select).getAllByRole("option");
    expect(options[0]).toHaveTextContent("All Types");
    expect(options.length).toBe(8); // All + 7 event types
  });

  it("calls API with event_type filter when event type is selected", async () => {
    mockApiFetch();

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByLabelText("Event Type")).toBeInTheDocument();
    });

    await user.selectOptions(
      screen.getByLabelText("Event Type"),
      "approval.denied",
    );

    await waitFor(() => {
      expect(getLastAuditQuery(mockGet)?.event_type).toBe("approval.denied");
    });
  });

  it("fetches next page when Load More is clicked", async () => {
    mockApiFetch();

    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ActivityPage />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Load More" }),
      ).toBeInTheDocument();
    });

    // Update mock for second page
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/audit-events") {
        return Promise.resolve({
          data: {
            data: [
              {
                event_type: "approval.approved",
                timestamp: "2026-02-20T08:00:00Z",
                agent_id: 5,
                agent_metadata: { name: "Extra Bot" },
                action: { type: "test.op", version: "1", parameters: {} },
                outcome: "approved",
              },
            ],
            has_more: false,
          },
        });
      }
      if (url === "/v1/agents") {
        return Promise.resolve({ data: mockAgents });
      }
      return Promise.resolve({ data: null });
    });

    await user.click(screen.getByRole("button", { name: "Load More" }));

    await waitFor(() => {
      expect(getLastAuditQuery(mockGet)?.after).toBe("cursor123");
    });
  });
});
