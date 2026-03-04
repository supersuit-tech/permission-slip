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
  invoicesResponse,
  overLimitPaidPlanResponse,
} from "./fixtures";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockBillingApi(planResponse: typeof freePlanResponse | typeof paidPlanResponse) {
  setupAuthMocks({ authenticated: true });
  mockGet.mockImplementation((url: string) => {
    if (url === "/v1/billing/plan") {
      return Promise.resolve({ data: planResponse });
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

    it("opens upgrade confirmation dialog on upgrade click", async () => {
      mockBillingApi(freePlanResponse);
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ }));

      await waitFor(() => {
        expect(screen.getByText("What you get")).toBeInTheDocument();
      });
      expect(screen.getByText("Pricing")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Continue to Checkout/ })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });

    it("redirects to Stripe after confirming upgrade dialog", async () => {
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

        // Click upgrade button to open dialog
        await user.click(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ }));

        await waitFor(() => {
          expect(screen.getByRole("button", { name: /Continue to Checkout/ })).toBeInTheDocument();
        });

        // Click "Continue to Checkout" in the dialog
        await user.click(screen.getByRole("button", { name: /Continue to Checkout/ }));

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

    it("opens downgrade confirmation dialog on click", async () => {
      mockBillingApi(paidPlanResponse);
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Downgrade to Free" }));

      await waitFor(() => {
        expect(screen.getByText("What changes")).toBeInTheDocument();
      });
      expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });

    it("shows limit warnings when over free plan limits", async () => {
      mockBillingApi(overLimitPaidPlanResponse);
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Downgrade to Free" }));

      await waitFor(() => {
        expect(screen.getByText(/over free plan limits/)).toBeInTheDocument();
      });
      expect(screen.getByText(/You have 5 agents/)).toBeInTheDocument();
      expect(screen.getByText(/You have 10 standing approvals/)).toBeInTheDocument();
      expect(screen.getByText(/You have 8 credentials/)).toBeInTheDocument();

      // Confirm button should be disabled when over limits
      expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeDisabled();
    });
  });

  describe("Success banner", () => {
    it("shows success banner when upgraded=true query param is present", async () => {
      mockBillingApi(paidPlanResponse);
      wrapper = createAuthWrapper(["/billing?upgraded=true"]);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Welcome to Pay-as-you-go!")).toBeInTheDocument();
      });
    });

    it("dismisses success banner when close button is clicked", async () => {
      mockBillingApi(paidPlanResponse);
      wrapper = createAuthWrapper(["/billing?upgraded=true"]);
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Welcome to Pay-as-you-go!")).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Dismiss success message" }));

      expect(screen.queryByText("Welcome to Pay-as-you-go!")).not.toBeInTheDocument();
    });

    it("does not show success banner without query param", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Current Plan")).toBeInTheDocument();
      });
      expect(screen.queryByText("Welcome to Pay-as-you-go!")).not.toBeInTheDocument();
    });
  });
});
