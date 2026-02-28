import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useAgent } from "../useAgent";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockAgentResponse = {
  agent_id: 42,
  status: "registered",
  metadata: { name: "My Bot" },
  registered_at: "2026-02-11T13:20:15Z",
  last_active_at: "2026-02-19T08:45:00Z",
  created_at: "2026-02-11T13:15:00Z",
};

describe("useAgent", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns null when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useAgent(42), { wrapper });

    expect(result.current.agent).toBeNull();
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches agent when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockAgentResponse });

    const { result } = renderHook(() => useAgent(42), { wrapper });

    await waitFor(() => {
      expect(result.current.agent).toEqual(mockAgentResponse);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/agents/{agent_id}", {
      headers: { Authorization: "Bearer token" },
      params: { path: { agent_id: 42 } },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useAgent(42), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.agent).toBeNull();
    expect(result.current.error).toBe(
      "Unable to load agent. Please try again later.",
    );
  });

  it("does not fetch for invalid agent ID", () => {
    setupAuthMocks({ authenticated: true });

    const { result } = renderHook(() => useAgent(0), { wrapper });

    expect(result.current.isLoading).toBe(false);
    expect(mockGet).not.toHaveBeenCalled();
  });
});
