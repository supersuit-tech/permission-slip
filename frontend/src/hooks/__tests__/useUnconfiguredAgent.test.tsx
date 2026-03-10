import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useUnconfiguredAgent } from "../useUnconfiguredAgent";
import type { Agent } from "../useAgents";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

function makeAgent(overrides: Partial<Agent> = {}): Agent {
  return {
    agent_id: 1,
    status: "registered",
    created_at: "2026-01-01T00:00:00Z",
    metadata: { name: "Test Agent" },
    ...overrides,
  };
}

describe("useUnconfiguredAgent", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
    setupAuthMocks({ authenticated: true });
  });

  it("returns isUnconfigured false when agents are still loading", () => {
    const { result } = renderHook(
      () => useUnconfiguredAgent([], true),
      { wrapper },
    );

    expect(result.current.isUnconfigured).toBe(false);
    // isLoading is false because the hook only tracks connector loading,
    // not the agents loading state (Dashboard handles that separately).
    expect(result.current.isLoading).toBe(false);
  });

  it("returns isUnconfigured false when no agents exist", () => {
    const { result } = renderHook(
      () => useUnconfiguredAgent([], false),
      { wrapper },
    );

    expect(result.current.isUnconfigured).toBe(false);
    expect(result.current.isLoading).toBe(false);
  });

  it("returns isUnconfigured true for single registered agent with no connectors", async () => {
    mockGet.mockResolvedValue({ data: { data: [] } });

    const agents = [makeAgent()];
    const { result } = renderHook(
      () => useUnconfiguredAgent(agents, false),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isUnconfigured).toBe(true);
    });
    expect(result.current.agentId).toBe(1);
    expect(result.current.agentName).toBe("Test Agent");
  });

  it("returns isUnconfigured false for single registered agent with connectors", async () => {
    mockGet.mockResolvedValue({
      data: {
        data: [
          {
            id: "github",
            name: "GitHub",
            actions: ["github.create_issue"],
            required_credentials: [],
            enabled_at: "2026-01-01T00:00:00Z",
          },
        ],
      },
    });

    const agents = [makeAgent()];
    const { result } = renderHook(
      () => useUnconfiguredAgent(agents, false),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.isUnconfigured).toBe(false);
  });

  it("returns isUnconfigured false when multiple registered agents exist", () => {
    const agents = [
      makeAgent({ agent_id: 1 }),
      makeAgent({ agent_id: 2 }),
    ];
    const { result } = renderHook(
      () => useUnconfiguredAgent(agents, false),
      { wrapper },
    );

    expect(result.current.isUnconfigured).toBe(false);
  });

  it("returns isUnconfigured false when single agent is pending", () => {
    const agents = [makeAgent({ status: "pending" })];
    const { result } = renderHook(
      () => useUnconfiguredAgent(agents, false),
      { wrapper },
    );

    expect(result.current.isUnconfigured).toBe(false);
    // Should not attempt to fetch connectors (agentId is 0)
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("returns isUnconfigured false on connector fetch error", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));

    const agents = [makeAgent()];
    const { result } = renderHook(
      () => useUnconfiguredAgent(agents, false),
      { wrapper },
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.isUnconfigured).toBe(false);
  });
});
