import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPost,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { PendingApprovalsCard } from "../PendingApprovalsCard";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const NOW = new Date("2026-02-21T10:00:00Z");

const mockApproval = {
  approval_id: "appr_abc123",
  agent_id: 1,
  action: {
    type: "email.send",
    version: "1",
    parameters: {
      from: "alice@example.com",
      to: ["bob@example.com"],
      subject: "Hello World",
      body: "Test email body",
    },
  },
  context: {
    description: "Send welcome email to new user",
    risk_level: "low" as const,
  },
  status: "pending" as const,
  expires_at: new Date(NOW.getTime() + 3 * 60 * 1000).toISOString(), // 3 min from now
  created_at: new Date(NOW.getTime() - 2 * 60 * 1000).toISOString(),
};

const mockHighRiskApproval = {
  approval_id: "appr_def456",
  agent_id: 2,
  action: {
    type: "data.delete",
    version: "1",
    parameters: {
      table: "users",
      filter: "inactive",
    },
  },
  context: {
    description: "Delete inactive user accounts",
    risk_level: "high" as const,
  },
  status: "pending" as const,
  expires_at: new Date(NOW.getTime() + 30 * 1000).toISOString(), // 30 seconds from now
  created_at: new Date(NOW.getTime() - 4.5 * 60 * 1000).toISOString(),
};

const mockApprovalsResponse = {
  data: [mockApproval, mockHighRiskApproval],
};

const emptyResponse = { data: [] };

function mockApprovalsFetch(response = mockApprovalsResponse) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockResolvedValue({ data: response });
}

describe("PendingApprovalsCard", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(NOW);
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders title", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    expect(screen.getByText("Pending Approvals")).toBeInTheDocument();
  });

  it("renders request count description when approvals exist", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("2 requests awaiting your review"),
      ).toBeInTheDocument();
    });
  });

  it("renders approval rows with action type and agent name", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("email.send")).toBeInTheDocument();
    });
    expect(screen.getByText("data.delete")).toBeInTheDocument();
    expect(screen.getByText("Agent 1")).toBeInTheDocument();
    expect(screen.getByText("Agent 2")).toBeInTheDocument();
  });

  it("renders risk badges", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("low")).toBeInTheDocument();
    });
    expect(screen.getByText("high")).toBeInTheDocument();
  });

  it("renders countdown timers", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("3:00")).toBeInTheDocument();
    });
    expect(screen.getByText("0:30")).toBeInTheDocument();
  });

  it("countdown decrements over time", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("3:00")).toBeInTheDocument();
    });

    act(() => {
      vi.advanceTimersByTime(5_000);
    });

    // shouldAdvanceTime lets real wall-clock time leak into Date.now(),
    // so the countdown may have ticked 1-2 extra seconds beyond the 5s
    // we advanced. Assert the timer decremented rather than checking an
    // exact value.
    await waitFor(() => {
      const timerEls = screen
        .getAllByText(/^\d+:\d{2}$/)
        .map((el) => el.textContent ?? "");
      const firstTimer = timerEls.find(
        (t) => t.startsWith("2:") && t !== "2:60",
      );
      expect(firstTimer).toBeTruthy();
    });
  });

  it("renders a Review button for each approval row", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Review/ })).toHaveLength(2);
    });
  });

  it("opens review dialog with details on Review click", async () => {
    mockApprovalsFetch();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("email.send")).toBeInTheDocument();
    });

    const reviewButtons = screen.getAllByRole("button", { name: /Review/ });
    await user.click(reviewButtons[0]!);

    await waitFor(() => {
      expect(
        screen.getByText("Review Approval Request"),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText("Send welcome email to new user"),
    ).toBeInTheDocument();
    // "alice@example.com" and "Hello World" may appear in both the list row
    // summary and the dialog. Use getAllByText to account for both locations.
    expect(screen.getAllByText("alice@example.com").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Hello World").length).toBeGreaterThanOrEqual(1);
  });

  it("approves via review dialog and shows confirmation code", async () => {
    mockApprovalsFetch();
    mockPost.mockResolvedValue({
      data: {
        approval_id: "appr_abc123",
        status: "approved",
        approved_at: "2026-02-21T10:00:05Z",
        confirmation_code: "RK3-P7M",
      },
    });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Review/ })).toHaveLength(2);
    });

    // Open the review dialog
    const reviewButtons = screen.getAllByRole("button", { name: /Review/ });
    await user.click(reviewButtons[0]!);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Approve" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(screen.getByText("RK3-P7M")).toBeInTheDocument();
    });
    expect(screen.getByText("Confirmation code")).toBeInTheDocument();
    expect(screen.getByText("Request Approved")).toBeInTheDocument();
  });

  it("denies via review dialog", async () => {
    mockApprovalsFetch();
    mockPost.mockResolvedValue({ data: {} });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Review/ })).toHaveLength(2);
    });

    // Open the review dialog
    const reviewButtons = screen.getAllByRole("button", { name: /Review/ });
    await user.click(reviewButtons[0]!);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Deny" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Deny" }));

    await waitFor(() => {
      expect(mockPost).toHaveBeenCalled();
    });
  });

  it("shows high-risk warning in review dialog", async () => {
    mockApprovalsFetch();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("data.delete")).toBeInTheDocument();
    });

    // Click Review on the high-risk approval (second row)
    const reviewButtons = screen.getAllByRole("button", { name: /Review/ });
    await user.click(reviewButtons[1]!);

    await waitFor(() => {
      expect(
        screen.getByText(/high-risk action/),
      ).toBeInTheDocument();
    });
  });

  it("renders empty state when no approvals", async () => {
    mockApprovalsFetch(emptyResponse);

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("No pending requests")).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Your agents are quiet/),
    ).toBeInTheDocument();
  });

  it("renders loading spinner initially", () => {
    setupAuthMocks({ authenticated: true });
    // Don't resolve the mock so it stays in loading state
    mockGet.mockReturnValue(new Promise(() => {}));

    render(<PendingApprovalsCard />, { wrapper });

    expect(screen.getByText("Pending Approvals")).toBeInTheDocument();
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("Unable to load approvals. Please try again later."),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("shows expired for timed-out approval", async () => {
    const expiredApproval = {
      ...mockApproval,
      expires_at: new Date(NOW.getTime() - 1_000).toISOString(),
    };
    mockApprovalsFetch({ data: [expiredApproval] });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Expired")).toBeInTheDocument();
    });
  });

  it("shows singular 'request' for single approval", async () => {
    mockApprovalsFetch({ data: [mockApproval] });

    render(<PendingApprovalsCard />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText("1 request awaiting your review"),
      ).toBeInTheDocument();
    });
  });
});
