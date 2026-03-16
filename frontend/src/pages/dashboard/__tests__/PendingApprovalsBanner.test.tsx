import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import {
  mockGet,
  mockPost,
  resetClientMocks,
} from "../../../api/__mocks__/client";
import { PendingApprovalsBanner } from "../PendingApprovalsBanner";

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

describe("PendingApprovalsBanner", () => {
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

  it("renders nothing when no approvals", async () => {
    mockApprovalsFetch(emptyResponse);

    const { container } = render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(container.innerHTML).toBe("");
    });
  });

  it("renders nothing while loading", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {}));

    const { container } = render(<PendingApprovalsBanner />, { wrapper });

    expect(container.innerHTML).toBe("");
  });

  it("renders error state with retry button", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(
        screen.getByText(/Could not load pending approvals/),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders banner items with action type and agent name", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("email.send")).toBeInTheDocument();
    });
    expect(screen.getByText("data.delete")).toBeInTheDocument();
    expect(screen.getByText("Agent 1")).toBeInTheDocument();
    expect(screen.getByText("Agent 2")).toBeInTheDocument();
  });

  it("renders risk badges", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("low")).toBeInTheDocument();
    });
    expect(screen.getByText("high")).toBeInTheDocument();
  });

  it("renders countdown timers", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      const timerEls = screen
        .getAllByText(/^\d+:\d{2}$/)
        .map((el) => el.textContent ?? "");
      expect(
        timerEls.some((t) => t === "3:00" || t.startsWith("2:5")),
      ).toBe(true);
      expect(
        timerEls.some(
          (t) => t === "0:30" || (t.startsWith("0:2") && t !== "0:2"),
        ),
      ).toBe(true);
    });
  });

  it("renders a Review link for each banner item", async () => {
    mockApprovalsFetch();

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Pending approval/ })).toHaveLength(2);
    });
  });

  it("opens review dialog on banner click", async () => {
    mockApprovalsFetch();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("email.send")).toBeInTheDocument();
    });

    const bannerItems = screen.getAllByRole("button", { name: /Pending approval/ });
    await user.click(bannerItems[0]!);

    // The dialog should open — check for the description and action buttons
    await waitFor(() => {
      expect(
        screen.getByText("Send welcome email to new user"),
      ).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: /Approve/i })).toBeInTheDocument();
  });

  it("approves via review dialog and shows success message", async () => {
    mockApprovalsFetch();
    mockPost.mockResolvedValue({
      data: {
        approval_id: "appr_abc123",
        status: "approved",
        approved_at: "2026-02-21T10:00:05Z",
        execution_status: "success",
        execution_result: { data: "ok" },
      },
    });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Pending approval/ })).toHaveLength(2);
    });

    const bannerItems = screen.getAllByRole("button", { name: /Pending approval/ });
    await user.click(bannerItems[0]!);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Approve" }),
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(screen.getByText("Action Executed Successfully")).toBeInTheDocument();
    });
    expect(screen.getByText(/The action has been executed/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("denies via review dialog", async () => {
    mockApprovalsFetch();
    mockPost.mockResolvedValue({ data: {} });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getAllByRole("button", { name: /Pending approval/ })).toHaveLength(2);
    });

    const bannerItems = screen.getAllByRole("button", { name: /Pending approval/ });
    await user.click(bannerItems[0]!);

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

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("data.delete")).toBeInTheDocument();
    });

    const bannerItems = screen.getAllByRole("button", { name: /Pending approval/ });
    await user.click(bannerItems[1]!);

    await waitFor(() => {
      expect(
        screen.getByText(/high-risk action/),
      ).toBeInTheDocument();
    });
  });

  it("shows expired for timed-out approval", async () => {
    const expiredApproval = {
      ...mockApproval,
      expires_at: new Date(NOW.getTime() - 1_000).toISOString(),
    };
    mockApprovalsFetch({ data: [expiredApproval] });

    render(<PendingApprovalsBanner />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Expired")).toBeInTheDocument();
    });
  });
});
