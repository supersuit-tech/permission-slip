import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks, settleAuthHydration } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useIsFullyConfigured } from "../useIsFullyConfigured";

vi.mock("../../api/client");

function mockAgentsResponse(agents: object[]) {
  return { data: { data: agents } };
}

function mockConnectorsResponse(connectors: object[]) {
  return { data: { data: connectors } };
}

const registeredAgent = {
  agent_id: 1,
  status: "registered",
  created_at: "2026-01-01T00:00:00Z",
  metadata: { name: "Test Agent" },
};

const sampleConnector = {
  id: "github",
  name: "GitHub",
  actions: ["github.create_issue"],
  required_credentials: [],
  enabled_at: "2026-01-01T00:00:00Z",
};

describe("useIsFullyConfigured", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns false when not authenticated", async () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useIsFullyConfigured(), { wrapper });

    await settleAuthHydration();
    expect(result.current.isFullyConfigured).toBe(false);
  });

  it("returns false when no agents exist", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue(mockAgentsResponse([]));

    const { result } = renderHook(() => useIsFullyConfigured(), { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.isFullyConfigured).toBe(false);
  });

  it("returns false when agent exists but has no connectors", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/agents") return Promise.resolve(mockAgentsResponse([registeredAgent]));
      return Promise.resolve(mockConnectorsResponse([]));
    });

    const { result } = renderHook(() => useIsFullyConfigured(), { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.isFullyConfigured).toBe(false);
  });

  it("returns true when agent has at least one connector", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockImplementation((url: string) => {
      if (url === "/v1/agents") return Promise.resolve(mockAgentsResponse([registeredAgent]));
      return Promise.resolve(mockConnectorsResponse([sampleConnector]));
    });

    const { result } = renderHook(() => useIsFullyConfigured(), { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(result.current.isFullyConfigured).toBe(true);
    });
  });

  it("returns false when only pending agents exist", async () => {
    setupAuthMocks({ authenticated: true });
    const pendingAgent = { ...registeredAgent, status: "pending" };
    mockGet.mockResolvedValue(mockAgentsResponse([pendingAgent]));

    const { result } = renderHook(() => useIsFullyConfigured(), { wrapper });

    await settleAuthHydration();
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.isFullyConfigured).toBe(false);
  });
});
