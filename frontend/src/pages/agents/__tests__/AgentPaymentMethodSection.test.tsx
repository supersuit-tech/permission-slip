import { screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderWithProviders } from "../../../test-helpers";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { mockGet } from "../../../api/__mocks__/client";
import { AgentPaymentMethodSection } from "../AgentPaymentMethodSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

// Helper to set up standard mock responses
function setupMocks({
  paymentMethods = [] as Array<{
    id: string;
    brand: string;
    last4: string;
    exp_month: number;
    exp_year: number;
    is_default: boolean;
    label?: string;
    created_at: string;
    updated_at: string;
  }>,
  binding = { agent_id: 42, payment_method_id: null as string | null },
}: {
  paymentMethods?: Array<{
    id: string;
    brand: string;
    last4: string;
    exp_month: number;
    exp_year: number;
    is_default: boolean;
    label?: string;
    created_at: string;
    updated_at: string;
  }>;
  binding?: { agent_id: number; payment_method_id: string | null };
} = {}) {
  mockGet.mockImplementation(
    (path: string) => {
      if (path === "/v1/payment-methods") {
        return Promise.resolve({
          data: { payment_methods: paymentMethods, max_allowed: 10 },
        });
      }
      if (path === "/v1/agents/{agent_id}/payment-method") {
        return Promise.resolve({ data: binding });
      }
      return Promise.resolve({ data: {} });
    },
  );
}

describe("AgentPaymentMethodSection", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    setupAuthMocks({ authenticated: true });
  });

  it("shows loading state", () => {
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves
    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);
    expect(
      screen.getByRole("status", { name: "Loading payment method" }),
    ).toBeInTheDocument();
  });

  it("shows empty state when no payment methods exist", async () => {
    setupMocks();
    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(
        screen.getByText(/No payment methods added yet/),
      ).toBeInTheDocument();
    });
  });

  it("shows dropdown with payment methods", async () => {
    setupMocks({
      paymentMethods: [
        {
          id: "pm-1",
          brand: "visa",
          last4: "4242",
          exp_month: 12,
          exp_year: 2028,
          is_default: true,
          label: "Work Visa",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
        {
          id: "pm-2",
          brand: "mastercard",
          last4: "5555",
          exp_month: 6,
          exp_year: 2027,
          is_default: false,
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ],
    });

    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(screen.getByText("Not set")).toBeInTheDocument();
    });

    const select = screen.getByRole("combobox");
    expect(select).toBeInTheDocument();
    // Should have "Select a payment method…" placeholder + 2 options
    const options = select.querySelectorAll("option");
    expect(options.length).toBe(3);
  });

  it("shows Assigned badge when payment method is set", async () => {
    setupMocks({
      paymentMethods: [
        {
          id: "pm-1",
          brand: "visa",
          last4: "4242",
          exp_month: 12,
          exp_year: 2028,
          is_default: true,
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ],
      binding: { agent_id: 42, payment_method_id: "pm-1" },
    });

    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(screen.getByText("Assigned")).toBeInTheDocument();
    });
  });

  it("shows Optional badge and subtitle in card header", async () => {
    setupMocks();
    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(screen.getByText("Optional")).toBeInTheDocument();
      expect(
        screen.getByText(/Only needed if you want this agent/),
      ).toBeInTheDocument();
    });
  });

  it("shows unassigned description when payment methods exist but none assigned", async () => {
    setupMocks({
      paymentMethods: [
        {
          id: "pm-1",
          brand: "visa",
          last4: "4242",
          exp_month: 12,
          exp_year: 2028,
          is_default: true,
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ],
    });

    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(
        screen.getByText(/No payment method assigned/),
      ).toBeInTheDocument();
    });
  });

  it("shows manage button", async () => {
    setupMocks({
      paymentMethods: [
        {
          id: "pm-1",
          brand: "visa",
          last4: "4242",
          exp_month: 12,
          exp_year: 2028,
          is_default: true,
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      ],
    });

    renderWithProviders(<AgentPaymentMethodSection agentId={42} />);

    await waitFor(() => {
      expect(
        screen.getByText("Manage payment methods"),
      ).toBeInTheDocument();
    });
  });
});
