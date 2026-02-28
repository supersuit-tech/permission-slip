import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { PendingAgentBanners } from "../PendingAgentBanners";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

// Use a large buffer (1 hour) to prevent flaky tests in slow CI environments
const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();

const responseWithPending = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      agent_id: 2,
      status: "pending",
      metadata: { name: "My Bot" },
      confirmation_code: "XK7-M9P",
      expires_at: futureDate,
      created_at: "2026-02-01T00:00:00Z",
    },
  ],
  has_more: false,
};

const responseNoPending = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      created_at: "2026-01-01T00:00:00Z",
    },
  ],
  has_more: false,
};

describe("PendingAgentBanners", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("renders nothing when no pending agents exist", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseNoPending });

    const { container } = render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(mockGet).toHaveBeenCalled();
    });

    // Should not render any banner content
    expect(container.querySelector("[role='status']")).not.toBeInTheDocument();
  });

  it("renders nothing when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { container } = render(<PendingAgentBanners />, { wrapper });

    expect(container.querySelector("[role='status']")).not.toBeInTheDocument();
  });

  it("renders a banner for each pending agent", async () => {
    const twoAgentsPending = {
      data: [
        {
          agent_id: 2,
          status: "pending",
          metadata: { name: "Bot A" },
          confirmation_code: "XK7-M9P",
          expires_at: futureDate,
          created_at: "2026-02-01T00:00:00Z",
        },
        {
          agent_id: 3,
          status: "pending",
          metadata: { name: "Bot B" },
          confirmation_code: "AB3-Z4Q",
          expires_at: futureDate,
          created_at: "2026-02-02T00:00:00Z",
        },
      ],
      has_more: false,
    };
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: twoAgentsPending });

    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Bot A")).toBeInTheDocument();
    });
    expect(screen.getByText("Bot B")).toBeInTheDocument();
  });

  it("displays agent name in the banner", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseWithPending });

    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("My Bot")).toBeInTheDocument();
    });
  });

  it("displays confirmation code in the banner", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseWithPending });

    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("XK7-M9P")).toBeInTheDocument();
    });
  });

  it("displays a Review link in the banner", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseWithPending });

    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Review")).toBeInTheDocument();
    });
  });

  it("opens review dialog when banner is clicked", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseWithPending });

    const user = userEvent.setup();
    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Review")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /Pending agent registration/ }));

    await waitFor(() => {
      expect(screen.getByText("Complete Agent Registration")).toBeInTheDocument();
    });
    expect(
      screen.getByText("Share this code with the agent to complete registration"),
    ).toBeInTheDocument();
  });

  it("falls back to Agent <id> when metadata.name is missing", async () => {
    const responseNoName = {
      data: [
        {
          agent_id: 7,
          status: "pending",
          confirmation_code: "QR5-ST6",
          expires_at: futureDate,
          created_at: "2026-02-01T00:00:00Z",
        },
      ],
      has_more: false,
    };
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: responseNoName });

    render(<PendingAgentBanners />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Agent 7")).toBeInTheDocument();
    });
  });
});
