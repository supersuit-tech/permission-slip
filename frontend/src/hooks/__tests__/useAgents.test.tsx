import { renderHook, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { setupAuthMocks } from "../../auth/__tests__/fixtures";
import { createAuthWrapper } from "../../test-helpers";
import { mockGet, resetClientMocks } from "../../api/__mocks__/client";
import { useAgents } from "../useAgents";

vi.mock("../../lib/supabaseClient");
vi.mock("../../api/client");

const mockAgentsResponse = {
  data: [
    {
      agent_id: 1,
      status: "registered",
      last_active_at: "2026-02-19T10:00:00Z",
      request_count_30d: 42,
      created_at: "2026-01-01T00:00:00Z",
    },
    {
      agent_id: 2,
      status: "pending",
      request_count_30d: 0,
      created_at: "2026-02-01T00:00:00Z",
    },
  ],
  has_more: false,
};

describe("useAgents", () => {
  let wrapper: ReturnType<typeof createAuthWrapper>;

  beforeEach(() => {
    vi.restoreAllMocks();
    resetClientMocks();
    wrapper = createAuthWrapper();
  });

  it("returns empty agents when not authenticated", () => {
    setupAuthMocks({ authenticated: false });

    const { result } = renderHook(() => useAgents(), {
      wrapper,
    });

    expect(result.current.agents).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it("fetches agents when authenticated", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockAgentsResponse });

    const { result } = renderHook(() => useAgents(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.agents).toEqual(mockAgentsResponse.data);
    });

    expect(mockGet).toHaveBeenCalledWith("/v1/agents", {
      headers: { Authorization: "Bearer token" },
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on fetch failure", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useAgents(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.agents).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load agents. Please try again later.",
    );
  });

  it("sets error on API error response", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: "internal_error", message: "Server error" } },
    });

    const { result } = renderHook(() => useAgents(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.agents).toEqual([]);
    expect(result.current.error).toBe(
      "Unable to load agents. Please try again later.",
    );
  });

  it("provides a refetch function", async () => {
    setupAuthMocks({ authenticated: true });
    mockGet.mockResolvedValue({ data: mockAgentsResponse });

    const { result } = renderHook(() => useAgents(), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.agents).toHaveLength(2);
    });

    expect(mockGet).toHaveBeenCalledTimes(1);

    // Call refetch
    await result.current.refetch();

    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});
