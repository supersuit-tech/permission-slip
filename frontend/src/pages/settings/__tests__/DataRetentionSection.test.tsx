import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../../test-helpers";
import { mockGet, resetClientMocks } from "../../../api/__mocks__/client";
import { DataRetentionSection } from "../DataRetentionSection";

vi.mock("../../../lib/supabaseClient");
vi.mock("../../../api/client");

describe("DataRetentionSection", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper(["/settings"]);
    setupAuthMocks({ authenticated: true });
  });

  it("shows free plan retention", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/profile/data-retention") {
        return Promise.resolve({
          data: {
            plan_id: "free",
            plan_name: "Free",
            audit_retention_days: 7,
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    render(<DataRetentionSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Free")).toBeInTheDocument();
    });
    expect(screen.getByText("7 days")).toBeInTheDocument();
  });

  it("shows paid plan retention", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/profile/data-retention") {
        return Promise.resolve({
          data: {
            plan_id: "pay_as_you_go",
            plan_name: "Pay As You Go",
            audit_retention_days: 90,
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    render(<DataRetentionSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Pay As You Go")).toBeInTheDocument();
    });
    expect(screen.getByText("90 days")).toBeInTheDocument();
  });

  it("shows grace period info when recently downgraded", async () => {
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/profile/data-retention") {
        return Promise.resolve({
          data: {
            plan_id: "free",
            plan_name: "Free",
            audit_retention_days: 7,
            effective_retention_days: 90,
            grace_period_ends_at: "2026-03-08T14:30:00Z",
          },
        });
      }
      return Promise.resolve({ data: null });
    });

    render(<DataRetentionSection />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText("Free")).toBeInTheDocument();
    });
    expect(screen.getByText("90 days")).toBeInTheDocument();
    expect(screen.getByText(/90-day audit history is temporarily preserved/)).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves

    render(<DataRetentionSection />, { wrapper });

    expect(
      screen.getByRole("status", { name: "Loading data retention policy" }),
    ).toBeInTheDocument();
  });
});
