import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { BillingPage } from "../BillingPage";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

const freePlanResponse = {
  plan: {
    id: "free",
    name: "Free",
    max_requests_per_month: 1000,
    max_agents: 3,
    max_standing_approvals: 5,
    max_credentials: 5,
    audit_retention_days: 7,
  },
  subscription: {
    status: "active" as const,
    current_period_start: "2026-03-01T00:00:00Z",
    current_period_end: "2026-04-01T00:00:00Z",
    has_payment_method: false,
    can_upgrade: true,
    can_downgrade: false,
    grace_period_ends_at: null,
  },
  usage: {
    requests: 450,
    agents: 2,
    standing_approvals: 3,
    credentials: 1,
  },
};

const paidPlanResponse = {
  plan: {
    id: "pay_as_you_go",
    name: "Pay As You Go",
    max_requests_per_month: null,
    max_agents: null,
    max_standing_approvals: null,
    max_credentials: null,
    audit_retention_days: 90,
  },
  subscription: {
    status: "active" as const,
    current_period_start: "2026-03-01T00:00:00Z",
    current_period_end: "2026-04-01T00:00:00Z",
    has_payment_method: true,
    can_upgrade: false,
    can_downgrade: true,
    grace_period_ends_at: null,
  },
  usage: {
    requests: 1542,
    agents: 5,
    standing_approvals: 10,
    credentials: 8,
  },
};

const usageDetailResponse = {
  period_start: "2026-03-01T00:00:00Z",
  period_end: "2026-04-01T00:00:00Z",
  requests: { total: 1542, included: 1000, overage: 542, cost_cents: 271 },
  sms: { total: 5, cost_cents: 5 },
};

const invoicesResponse = {
  invoices: [
    {
      id: "inv_001",
      date: "2026-02-01T00:00:00Z",
      period_start: "2026-02-01T00:00:00Z",
      period_end: "2026-03-01T00:00:00Z",
      amount_cents: 271,
      status: "paid",
      stripe_invoice_url: "https://invoice.stripe.com/i/test",
    },
  ],
  has_more: false,
};

function mockFreePlanApi() {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: freePlanResponse });
    }
    if (url === "/v1/billing/usage") {
      return Promise.resolve({ data: usageDetailResponse });
    }
    if (url === "/v1/billing/invoices") {
      return Promise.resolve({ data: invoicesResponse });
    }
    return Promise.resolve({ data: null });
  });
}

function mockPaidPlanApi() {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: paidPlanResponse });
    }
    if (url === "/v1/billing/usage") {
      return Promise.resolve({ data: usageDetailResponse });
    }
    if (url === "/v1/billing/invoices") {
      return Promise.resolve({ data: invoicesResponse });
    }
    return Promise.resolve({ data: null });
  });
}

function mockErrorApi() {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({
        data: undefined,
        error: { error: { code: "internal_error", message: "Server error" } },
      });
    }
    return Promise.resolve({ data: null });
  });
}

describe("BillingPage", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/billing"]);
  });

  it("renders page title and back link", async () => {
    mockFreePlanApi();

    render(<BillingPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Billing")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("link", { name: "Back to Dashboard" }),
    ).toHaveAttribute("href", "/");
  });

  it("shows loading state", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation(() => new Promise(() => {}));

    render(<BillingPage />, { wrapper });

    expect(screen.getByRole("status", { name: "Loading billing information" })).toBeInTheDocument();
  });

  it("shows error state with retry button", async () => {
    mockErrorApi();

    render(<BillingPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText(/Unable to load billing plan/)).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Try Again" })).toBeInTheDocument();
  });

  describe("Free plan", () => {
    it("renders plan card with Free badge", async () => {
      mockFreePlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.getAllByText("Free").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("Active")).toBeInTheDocument();
    });

    it("renders usage summary with progress indicators", async () => {
      mockFreePlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage")).toBeInTheDocument();
      });
      expect(screen.getByText("Requests")).toBeInTheDocument();
      expect(screen.getByText("Agents")).toBeInTheDocument();
      expect(screen.getByText("Standing Approvals")).toBeInTheDocument();
      expect(screen.getByText("Credentials")).toBeInTheDocument();
      expect(screen.getByText("Audit Retention")).toBeInTheDocument();
      expect(screen.getByText("7 days")).toBeInTheDocument();
    });

    it("shows upgrade CTA for free plan", async () => {
      mockFreePlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Upgrade Your Plan")).toBeInTheDocument();
      });
      expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
    });

    it("does not show plan details card for free plan", async () => {
      mockFreePlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.queryByText("Plan Details")).not.toBeInTheDocument();
    });

    it("redirects to Stripe on upgrade click", async () => {
      mockFreePlanApi();
      mockPost.mockResolvedValue({
        data: { checkout_url: "https://checkout.stripe.com/test" },
      });

      // Mock window.location for redirect
      const originalLocation = window.location;
      Object.defineProperty(window, "location", {
        writable: true,
        value: { ...originalLocation, href: "" },
      });

      const user = userEvent.setup();
      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ }));

      await waitFor(() => {
        expect(window.location.href).toBe("https://checkout.stripe.com/test");
      });

      Object.defineProperty(window, "location", {
        writable: true,
        value: originalLocation,
      });
    });
  });

  describe("Paid plan", () => {
    it("renders plan card with Pay-as-you-go badge", async () => {
      mockPaidPlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.getByText("Pay-as-you-go")).toBeInTheDocument();
    });

    it("shows unlimited usage for paid plan", async () => {
      mockPaidPlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage")).toBeInTheDocument();
      });
      // Paid plan has null limits, so usage rows show "Unlimited"
      const unlimitedTexts = screen.getAllByText(/Unlimited/);
      expect(unlimitedTexts.length).toBeGreaterThanOrEqual(1);
    });

    it("does not show upgrade CTA for paid plan", async () => {
      mockPaidPlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.queryByText("Upgrade Your Plan")).not.toBeInTheDocument();
    });

    it("shows plan details card with invoices", async () => {
      mockPaidPlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Plan Details")).toBeInTheDocument();
      });
      expect(screen.getByText("Recent Invoices")).toBeInTheDocument();
      expect(screen.getByText("Estimated Cost (this month)")).toBeInTheDocument();
    });

    it("shows downgrade button on paid plan", async () => {
      mockPaidPlanApi();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });
    });

    it("shows downgrade confirmation on click", async () => {
      mockPaidPlanApi();
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Downgrade to Free" }));

      expect(screen.getByText("Are you sure?")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });
  });
});
