import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, mockPost, resetClientMocks } from "../../../api/__mocks__/client";
import { BillingPage } from "../BillingPage";
import {
  freePlanResponse,
  paidPlanResponse,
  usageDetailResponse,
  freeUsageDetailResponse,
  invoicesResponse,
  agentsResponse,
} from "./fixtures";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockBillingApi(
  planResponse: typeof freePlanResponse | typeof paidPlanResponse,
  usageResponse: typeof usageDetailResponse | typeof freeUsageDetailResponse = usageDetailResponse,
) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: planResponse });
    }
    if (url === "/v1/billing/usage") {
      return Promise.resolve({ data: usageResponse });
    }
    if (url === "/v1/billing/invoices") {
      return Promise.resolve({ data: invoicesResponse });
    }
    if (url === "/v1/agents") {
      return Promise.resolve({ data: agentsResponse });
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
    mockBillingApi(freePlanResponse);

    render(<BillingPage />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Billing")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("link", { name: "Back to Dashboard" }),
    ).toHaveAttribute("href", "/");
  });

  it("shows loading skeleton", () => {
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
      mockBillingApi(freePlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.getAllByText("Free").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("Active")).toBeInTheDocument();
    });

    it("renders usage summary with progress indicators", async () => {
      mockBillingApi(freePlanResponse);

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
      mockBillingApi(freePlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Upgrade Your Plan")).toBeInTheDocument();
      });
      expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
    });

    it("does not show plan details card for free plan", async () => {
      mockBillingApi(freePlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.queryByText("Plan Details")).not.toBeInTheDocument();
    });

    it("redirects to Stripe on upgrade click", async () => {
      mockBillingApi(freePlanResponse);
      mockPost.mockResolvedValue({
        data: { checkout_url: "https://checkout.stripe.com/test" },
      });

      const originalLocation = window.location;
      Object.defineProperty(window, "location", {
        writable: true,
        value: { ...originalLocation, href: "" },
      });

      try {
        const user = userEvent.setup();
        render(<BillingPage />, { wrapper });

        await waitFor(() => {
          expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
        });

        await user.click(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ }));

        await waitFor(() => {
          expect(window.location.href).toBe("https://checkout.stripe.com/test");
        });
      } finally {
        Object.defineProperty(window, "location", {
          writable: true,
          value: originalLocation,
        });
      }
    });
  });

  describe("Paid plan", () => {
    it("renders plan card with Pay-as-you-go badge", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.getByText("Pay-as-you-go")).toBeInTheDocument();
    });

    it("shows unlimited usage for paid plan", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage")).toBeInTheDocument();
      });
      const unlimitedTexts = screen.getAllByText(/Unlimited/);
      expect(unlimitedTexts.length).toBeGreaterThanOrEqual(1);
    });

    it("does not show upgrade CTA for paid plan", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.queryByText("Upgrade Your Plan")).not.toBeInTheDocument();
    });

    it("shows plan details card with invoices", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Plan Details")).toBeInTheDocument();
      });
      expect(screen.getByText("Recent Invoices")).toBeInTheDocument();
      expect(screen.getByText("Estimated Cost (this month)")).toBeInTheDocument();
    });

    it("shows downgrade button on paid plan", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });
    });

    it("shows downgrade confirmation on click", async () => {
      mockBillingApi(paidPlanResponse);
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

  describe("Usage Dashboard", () => {
    it("renders usage details card", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage Details")).toBeInTheDocument();
      });
    });

    it("shows requests progress bar for free plan", async () => {
      mockBillingApi(freePlanResponse, freeUsageDetailResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Requests this period")).toBeInTheDocument();
      });
      expect(screen.getByText("Requests remaining")).toBeInTheDocument();
    });

    it("shows estimated cost for paid plan", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Estimated cost")).toBeInTheDocument();
      });
      // $2.71 requests + $0.05 SMS = $2.76 (shown in both UsageDashboard and PlanDetailsCard)
      expect(screen.getAllByText("$2.76").length).toBeGreaterThanOrEqual(1);
    });

    it("shows days remaining in billing period", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Days remaining")).toBeInTheDocument();
      });
    });

    it("shows per-agent breakdown table with agent names", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage by Agent")).toBeInTheDocument();
      });
      // Agent names load from a separate query, so wait for them
      await waitFor(() => {
        expect(screen.getByText("Gmail Bot")).toBeInTheDocument();
      });
      expect(screen.getByText("Stripe Bot")).toBeInTheDocument();
      expect(screen.getByText("% of total")).toBeInTheDocument();
    });

    it("shows SMS usage section for paid plan with SMS", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("SMS Usage")).toBeInTheDocument();
      });
      expect(screen.getByText("Messages sent")).toBeInTheDocument();
    });

    it("does not show SMS section for free plan with no SMS", async () => {
      mockBillingApi(freePlanResponse, freeUsageDetailResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage Details")).toBeInTheDocument();
      });
      expect(screen.queryByText("SMS Usage")).not.toBeInTheDocument();
    });
  });
});
