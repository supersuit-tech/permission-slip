import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useResourceLimit } from "../useResourceLimit";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const freePlanLimits = {
  max_requests_per_month: 1000,
  max_agents: 3,
  max_standing_approvals: 5,
  max_credentials: 5,
  audit_retention_days: 7,
};

const freePlan = {
  plan: {
    id: "free",
    name: "Free",
    ...freePlanLimits,
  },
  effective_limits: freePlanLimits,
  subscription: { status: "active" },
  usage: { requests: 10, agents: 2, standing_approvals: 4, credentials: 5 },
};

const paidPlanLimits = {
  max_requests_per_month: null,
  max_agents: null,
  max_standing_approvals: null,
  max_credentials: null,
  audit_retention_days: 90,
};

const paidPlan = {
  plan: {
    id: "pay_as_you_go",
    name: "Pay as you go",
    ...paidPlanLimits,
  },
  effective_limits: paidPlanLimits,
  subscription: { status: "active" },
  usage: { requests: 100, agents: 10, standing_approvals: 20, credentials: 15 },
};

describe("useResourceLimit", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("extracts agent limit from free plan", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: freePlan });

    const { result } = renderHook(
      () => useResourceLimit("max_agents", 0),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });
    expect(result.current.max).toBe(3);
    expect(result.current.current).toBe(2);
    expect(result.current.atLimit).toBe(false);
  });

  it("detects at-limit state for credentials", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: freePlan });

    const { result } = renderHook(
      () => useResourceLimit("max_credentials", 0),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });
    expect(result.current.max).toBe(5);
    expect(result.current.current).toBe(5);
    expect(result.current.atLimit).toBe(true);
  });

  it("returns null max for unlimited paid plan", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: paidPlan });

    const { result } = renderHook(
      () => useResourceLimit("max_agents", 0),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.hasData).toBe(true);
    });
    expect(result.current.max).toBeNull();
    expect(result.current.current).toBe(10);
    expect(result.current.atLimit).toBe(false);
  });

  it("uses fallback count when billing data is not loaded", () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockReturnValue(new Promise(() => {})); // never resolves

    const { result } = renderHook(
      () => useResourceLimit("max_agents", 7),
      { wrapper },
    );

    expect(result.current.hasData).toBe(false);
    expect(result.current.max).toBeNull();
    expect(result.current.current).toBe(7);
    expect(result.current.atLimit).toBe(false);
  });
});
