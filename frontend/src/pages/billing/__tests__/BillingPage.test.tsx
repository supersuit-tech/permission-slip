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
  paidUnderAllowanceUsageResponse,
  invoicesResponse,
  agentsResponse,
  overLimitPaidPlanResponse,
  atLimitPaidPlanResponse,
} from "./fixtures";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

function mockBillingApi(
  planResponse: typeof freePlanResponse | typeof paidPlanResponse,
  usageResponse: typeof usageDetailResponse | typeof freeUsageDetailResponse | typeof paidUnderAllowanceUsageResponse = usageDetailResponse,
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

    it("dismisses upgrade dialog when cancel is clicked", async () => {
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

      await user.click(screen.getByRole("button", { name: "Cancel" }));

      await waitFor(() => {
        expect(screen.queryByText("What you get")).not.toBeInTheDocument();
      });
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

    it("shows error toast when upgrade API fails", async () => {
      mockBillingApi(freePlanResponse);
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { code: "internal_error", message: "Stripe unavailable" } },
      });

      const user = userEvent.setup();
      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: /Upgrade to Pay-as-you-go/ }));
      await waitFor(() => {
        expect(screen.getByRole("button", { name: /Continue to Checkout/ })).toBeInTheDocument();
      });
      await user.click(screen.getByRole("button", { name: /Continue to Checkout/ }));

      // Dialog should close on error (upgrade CTA button reappears without dialog)
      await waitFor(() => {
        expect(screen.queryByText("What you get")).not.toBeInTheDocument();
      });
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

    it("shows unlimited usage for paid plan resources", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage")).toBeInTheDocument();
      });
      // Agents, Standing Approvals, Credentials show "Unlimited"
      const unlimitedTexts = screen.getAllByText(/Unlimited/);
      expect(unlimitedTexts.length).toBeGreaterThanOrEqual(1);
    });

    it("shows free request allowance for paid plan", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Usage")).toBeInTheDocument();
      });
      // Paid plan shows requests against the 1000 free allowance with billed count
      expect(screen.getByText(/billed/)).toBeInTheDocument();
      expect(screen.getByText(/\$0.005\/request/)).toBeInTheDocument();
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
      expect(screen.getByText(/You have 10 agents/)).toBeInTheDocument();
      expect(screen.getByText(/You have 25 standing approvals/)).toBeInTheDocument();
      expect(screen.getByText(/You have 20 credentials/)).toBeInTheDocument();

      // Confirm button should be disabled when over limits
      expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeDisabled();

      // Should show "Manage" links for each over-limit resource
      expect(screen.getByText("Manage agents")).toBeInTheDocument();
      expect(screen.getByText("Manage standing approvals")).toBeInTheDocument();
      expect(screen.getByText("Manage credentials")).toBeInTheDocument();
    });

    it("does not show limit warnings when at free plan limits", async () => {
      mockBillingApi(atLimitPaidPlanResponse);
      const user = userEvent.setup();

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Downgrade to Free" }));

      await waitFor(() => {
        expect(screen.getByText("What changes")).toBeInTheDocument();
      });
      expect(screen.queryByText(/over free plan limits/)).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeEnabled();
    });

    it("shows error in downgrade dialog when downgrade API fails", async () => {
      mockBillingApi(atLimitPaidPlanResponse);
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { code: "downgrade_limit_exceeded", message: "Too many active agents" } },
      });

      const user = userEvent.setup();
      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Downgrade to Free" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Downgrade to Free" }));
      await waitFor(() => {
        expect(screen.getByRole("button", { name: "Confirm Downgrade" })).toBeInTheDocument();
      });

      await user.click(screen.getByRole("button", { name: "Confirm Downgrade" }));

      await waitFor(() => {
        expect(screen.getByText("Too many active agents")).toBeInTheDocument();
      });
    });

    it("dismisses downgrade dialog when cancel is clicked", async () => {
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

      await user.click(screen.getByRole("button", { name: "Cancel" }));

      await waitFor(() => {
        expect(screen.queryByText("What changes")).not.toBeInTheDocument();
      });
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

    it("success banner has accessible role", async () => {
      mockBillingApi(paidPlanResponse);
      wrapper = createAuthWrapper(["/billing?upgraded=true"]);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Welcome to Pay-as-you-go!")).toBeInTheDocument();
      });
      const banner = screen.getByText("Welcome to Pay-as-you-go!").closest("[role='status']");
      expect(banner).toBeInTheDocument();
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

    it("shows free requests remaining when under allowance", async () => {
      mockBillingApi(paidPlanResponse, paidUnderAllowanceUsageResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Estimated cost")).toBeInTheDocument();
      });
      expect(screen.getAllByText("$0.00").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText(/free requests remaining/)).toBeInTheDocument();
    });

    it("shows billed requests count when over allowance", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Estimated cost")).toBeInTheDocument();
      });
      expect(screen.getByText(/requests billed/)).toBeInTheDocument();
    });

    it("shows days remaining in billing period", async () => {
      mockBillingApi(paidPlanResponse);

      render(<BillingPage />, { wrapper });

      await waitFor(() => {
        expect(screen.getByText("Days remaining")).toBeInTheDocument();
      });
    });

    it("shows per-agent breakdown table with agent names and links", async () => {
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
      // Agent names link to detail pages
      expect(screen.getByRole("link", { name: "Gmail Bot" })).toHaveAttribute("href", "/agents/1");
      expect(screen.getByRole("link", { name: "Stripe Bot" })).toHaveAttribute("href", "/agents/2");
      // Paid plan shows estimated cost per agent
      expect(screen.getByText("Est. cost")).toBeInTheDocument();
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
